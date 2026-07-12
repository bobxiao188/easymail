/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * For commercial licensing inquiries, please contact: 3680010825@qq.com
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package filter

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"easymail/internal/domain/filter/classifier"
	bertmodel "easymail/internal/infrastructure/model"
	"easymail/internal/infrastructure/persistence/mysql"
)

// modelConf is the JSON structure for model.conf inside the export zip.
type modelConf struct {
	Name          string                 `json:"name"`
	Algorithm     string                 `json:"algorithm"`
	Tokenizer     string                 `json:"tokenizer"`
	Languages     []string               `json:"languages"`
	Labels        []string               `json:"labels"`
	Parameters    classifier.ModelParams `json:"parameters"`
	MaxTextLength int                    `json:"maxTextLength"`
	EmailFields   []string               `json:"emailFields"`
}

// ExportModel packages model binary + config into a zip and returns the bytes.
func (s *classifyModelService) ExportModel(ctx context.Context, id int64) ([]byte, string, error) {
	m, err := s.loadClassifyModelByID(ctx, id)
	if err != nil {
		return nil, "", err
	}

	// Determine binary path based on algorithm
	binPath, binName := s.resolveBinaryPath(m)
	if binPath == "" {
		return nil, "", ErrClassifyModelExportNoBin
	}

	// Verify binary exists
	if _, err := os.Stat(binPath); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrClassifyModelExportNoBin, err)
	}

	// Read binary content
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrClassifyModelExportNoBin, err)
	}

	// Build model.conf
	conf := modelConf{
		Name:          m.Name,
		Algorithm:     string(m.Algorithm),
		Tokenizer:     string(m.Tokenizer),
		Languages:     m.Languages,
		Labels:        m.ClassLabels,
		Parameters:    m.Params,
		MaxTextLength: m.MaxTextLength,
		EmailFields:   m.EmailFields,
	}
	confData, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("marshal conf: %w", err)
	}

	// Create zip in memory
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Write model.conf
	confFile, err := w.Create("model.conf")
	if err != nil {
		_ = w.Close()
		return nil, "", fmt.Errorf("create zip entry: %w", err)
	}
	if _, err := confFile.Write(confData); err != nil {
		_ = w.Close()
		return nil, "", fmt.Errorf("write conf to zip: %w", err)
	}

	// Write binary
	binEntry, err := w.Create(binName)
	if err != nil {
		_ = w.Close()
		return nil, "", fmt.Errorf("create zip entry: %w", err)
	}
	if _, err := binEntry.Write(binData); err != nil {
		_ = w.Close()
		return nil, "", fmt.Errorf("write binary to zip: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("close zip: %w", err)
	}

	filename := SanitizeClassifyModelDirStem(m.Name) + ".zip"
	return buf.Bytes(), filename, nil
}

// resolveBinaryPath returns the absolute path to the model binary and the name to use inside the zip.
func (s *classifyModelService) resolveBinaryPath(m *classifier.Model) (absPath, zipName string) {
	savePath := strings.TrimSpace(m.SavePath)
	if savePath == "" {
		return "", ""
	}

	switch m.Algorithm {
	case classifier.AlgorithmFastText:
		// SavePath points to the .bin file directly
		abs, err := filepath.Abs(filepath.Clean(savePath))
		if err != nil {
			return "", ""
		}
		return abs, "model.bin"
	case classifier.AlgorithmDistilBERT:
		// SavePath points to the directory
		dir := filepath.Clean(savePath)
		abs := filepath.Join(dir, distilBERTOnnxFileName)
		absAbs, err := filepath.Abs(abs)
		if err != nil {
			return "", ""
		}
		return absAbs, "model.onnx"
	default:
		return "", ""
	}
}

// ImportModel extracts a zip, validates structure, and registers the model.
func (s *classifyModelService) ImportModel(ctx context.Context, zipFile *multipart.FileHeader, expectedAlgorithm string) error {
	// Open the uploaded zip
	src, err := zipFile.Open()
	if err != nil {
		return ErrClassifyModelImportInvalidZip
	}
	defer src.Close()

	// Read zip into memory for parsing
	zipData, err := io.ReadAll(src)
	if err != nil {
		return ErrClassifyModelImportInvalidZip
	}

	// Open zip reader
	reader, err := zip.NewReader(bytes.NewReader(zipData), zipFile.Size)
	if err != nil {
		return ErrClassifyModelImportInvalidZip
	}

	// Create temp directory for extraction
	tmpDir, err := os.MkdirTemp("", "model_import_*")
	if err != nil {
		return ErrClassifyModelImportWriteFailed
	}
	defer os.RemoveAll(tmpDir)

	// Extract all files
	for _, f := range reader.File {
		target := filepath.Join(tmpDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := extractZipFile(f, target); err != nil {
			return ErrClassifyModelImportWriteFailed
		}
	}

	// Validate model.conf exists
	confPath := filepath.Join(tmpDir, "model.conf")
	if _, err := os.Stat(confPath); err != nil {
		return ErrClassifyModelImportMissingConf
	}

	// Parse model.conf
	confData, err := os.ReadFile(confPath)
	if err != nil {
		return ErrClassifyModelImportConfParse
	}
	var conf modelConf
	if err := json.Unmarshal(confData, &conf); err != nil {
		return fmt.Errorf("%w: %v", ErrClassifyModelImportConfParse, err)
	}

	// Validate required fields
	if strings.TrimSpace(conf.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrClassifyModelImportConfParse)
	}
	if strings.TrimSpace(conf.Algorithm) == "" {
		return fmt.Errorf("%w: algorithm is required", ErrClassifyModelImportConfParse)
	}

	// Validate algorithm matches expected type from frontend
	if expectedAlgorithm != "" && !strings.EqualFold(conf.Algorithm, expectedAlgorithm) {
		return ErrClassifyModelImportAlgorithmMismatch
	}

	// Determine binary file based on algorithm
	zipBinName := s.expectedBinaryName(conf.Algorithm)
	binPath := filepath.Join(tmpDir, zipBinName)
	if _, err := os.Stat(binPath); err != nil {
		// Check if the other type exists to give a better error
		altName := s.expectedBinaryName(s.invertAlgorithm(conf.Algorithm))
		altPath := filepath.Join(tmpDir, altName)
		if _, statErr := os.Stat(altPath); statErr == nil {
			return ErrClassifyModelImportAlgorithmMismatch
		}
		return ErrClassifyModelImportMissingBin
	}

	// Validate name uniqueness
	if err := ValidateClassifyModelDisplayName(ctx, s.db, conf.Name, 0); err != nil {
		return ErrClassifyModelImportNameConflict
	}

	// Get model root
	absRoot, err := s.distilbertAbsModelRoot()
	if err != nil {
		return err
	}

	algorithm := classifier.Algorithm(conf.Algorithm)
	switch algorithm {
	case classifier.AlgorithmFastText:
		return s.importFastText(ctx, &conf, tmpDir, binPath, absRoot)
	case classifier.AlgorithmDistilBERT:
		return s.importDistilBERT(ctx, &conf, tmpDir, binPath, absRoot)
	default:
		return ErrClassifyModelImportAlgorithmMismatch
	}
}

func (s *classifyModelService) expectedBinaryName(algorithm string) string {
	switch strings.ToLower(algorithm) {
	case "fasttext":
		return "model.bin"
	case "distilbert":
		return "model.onnx"
	default:
		return "model.bin"
	}
}

func (s *classifyModelService) invertAlgorithm(algorithm string) string {
	switch strings.ToLower(algorithm) {
	case "fasttext":
		return "distilbert"
	case "distilbert":
		return "fasttext"
	default:
		return algorithm
	}
}

func extractZipFile(f *zip.File, target string) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func (s *classifyModelService) importFastText(ctx context.Context, conf *modelConf, tmpDir, binPath, absRoot string) error {
	destDir := filepath.Join(absRoot, SanitizeClassifyModelDirStem(conf.Name))
	destBin := filepath.Join(destDir, fastTextTrainOutputPrefix+".bin")

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ErrClassifyModelImportWriteFailed
	}

	// Copy binary to destination
	if err := copyFile(binPath, destBin); err != nil {
		_ = os.RemoveAll(destDir)
		return ErrClassifyModelImportWriteFailed
	}

	model := classifier.Model{
		Name:          conf.Name,
		Algorithm:     classifier.AlgorithmFastText,
		Tokenizer:     classifier.Tokenizer(conf.Tokenizer),
		Languages:     conf.Languages,
		SavePath:      destBin,
		ClassLabels:   conf.Labels,
		Params:        conf.Parameters,
		MaxTextLength: conf.MaxTextLength,
		EmailFields:   conf.EmailFields,
		Enabled:       false,
		IsDeleted:     false,
		TrainStatus:   classifier.TrainStatusCompleted,
		TrainTime:     time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	po := mysql.ClassifyModelToPO(&model)
	if err := s.db.WithContext(ctx).Create(po).Error; err != nil {
		_ = os.RemoveAll(destDir)
		return err
	}

	return nil
}

func (s *classifyModelService) importDistilBERT(ctx context.Context, conf *modelConf, tmpDir, binPath, absRoot string) error {
	destDir := filepath.Join(absRoot, SanitizeClassifyModelDirStem(conf.Name))
	destOnnx := filepath.Join(destDir, distilBERTOnnxFileName)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ErrClassifyModelImportWriteFailed
	}

	// Copy ONNX to a temp location first for label extraction
	tmpOnnx := destOnnx + ".tmp"
	if err := copyFile(binPath, tmpOnnx); err != nil {
		_ = os.RemoveAll(destDir)
		return ErrClassifyModelImportWriteFailed
	}

	// Extract labels from ONNX
	classLabels, err := bertmodel.ExtractDistilBERTClassLabelsFromONNX(tmpOnnx, s.onnxRuntimeLib)
	if err != nil {
		_ = os.Remove(tmpOnnx)
		_ = os.RemoveAll(destDir)
		return fmt.Errorf("%w: %v", ErrClassifyModelOnnxLabelsParse, err)
	}
	if len(classLabels) < 2 {
		_ = os.Remove(tmpOnnx)
		_ = os.RemoveAll(destDir)
		return fmt.Errorf("%w: need at least 2 classes", ErrClassifyModelOnnxLabelsParse)
	}

	// Move temp to final
	if err := os.Rename(tmpOnnx, destOnnx); err != nil {
		_ = os.Remove(tmpOnnx)
		_ = os.RemoveAll(destDir)
		return ErrClassifyModelImportWriteFailed
	}

	model := classifier.Model{
		Name:          conf.Name,
		Algorithm:     classifier.AlgorithmDistilBERT,
		Tokenizer:     classifier.Tokenizer(conf.Tokenizer),
		Languages:     conf.Languages,
		SavePath:      destDir,
		ClassLabels:   classifier.ClassLabels(classLabels),
		Params:        conf.Parameters,
		MaxTextLength: conf.MaxTextLength,
		EmailFields:   conf.EmailFields,
		Enabled:       false,
		IsDeleted:     false,
		TrainStatus:   classifier.TrainStatusCompleted,
		TrainTime:     time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Apply DistilBERT-specific params defaults
	if strings.TrimSpace(model.Params.Algorithm) == "" {
		model.Params.Algorithm = string(classifier.AlgorithmDistilBERT)
	}
	if strings.TrimSpace(model.Params.ModelFile) == "" {
		model.Params.ModelFile = distilBERTOnnxFileName
	}

	po := mysql.ClassifyModelToPO(&model)
	if err := s.db.WithContext(ctx).Create(po).Error; err != nil {
		_ = os.RemoveAll(destDir)
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		_ = os.Remove(dst)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	return nil
}
