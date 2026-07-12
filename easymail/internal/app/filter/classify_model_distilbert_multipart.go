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
 * Author: bob.xiao
 * License: AGPLv3
 */

package filter

import (
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

	"gorm.io/gorm"
)

// distilBERTOnnxFileName is the single ONNX artifact under model_root/<SanitizeClassifyModelDirStem(name)>/.
const distilBERTOnnxFileName = "model.onnx"

func classifyModelPathsEqual(a, b string) bool {
	ca, e1 := filepath.Abs(filepath.Clean(strings.TrimSpace(a)))
	cb, e2 := filepath.Abs(filepath.Clean(strings.TrimSpace(b)))
	if e1 != nil || e2 != nil {
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	return strings.EqualFold(ca, cb)
}

func (s *classifyModelService) distilbertAbsModelRoot() (string, error) {
	root := strings.TrimSpace(s.modelRoot)
	if root == "" {
		return "", ErrClassifyModelModelRootNotConfigured
	}
	return filepath.Abs(filepath.Clean(root))
}

func writeDistilBERTOnnxToPath(fh *multipart.FileHeader, destAbs string) error {
	if fh == nil {
		return ErrClassifyModelOnnxRequired
	}
	if !strings.EqualFold(filepath.Ext(fh.Filename), ".onnx") {
		return ErrClassifyModelOnnxInvalidExt
	}
	src, err := fh.Open()
	if err != nil {
		return ErrClassifyModelOnnxWriteFailed
	}
	defer src.Close()
	out, err := os.Create(destAbs)
	if err != nil {
		return ErrClassifyModelOnnxWriteFailed
	}
	if _, err := io.Copy(out, src); err != nil {
		_ = out.Close()
		_ = os.Remove(destAbs)
		return ErrClassifyModelOnnxWriteFailed
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(destAbs)
		return ErrClassifyModelOnnxWriteFailed
	}
	return nil
}

func applyDistilBERTParams(model *classifier.Model, paramsJSON string) {
	if strings.TrimSpace(paramsJSON) != "" {
		var mp classifier.ModelParams
		if err := json.Unmarshal([]byte(paramsJSON), &mp); err == nil {
			model.Params = mp
		}
	}
	if strings.TrimSpace(model.Params.Algorithm) == "" {
		model.Params.Algorithm = string(classifier.AlgorithmDistilBERT)
	}
	if strings.TrimSpace(model.Params.ModelFile) == "" {
		model.Params.ModelFile = distilBERTOnnxFileName
	}
}

// CreateDistilBERTWithONNXFile writes model.onnx under model_root/<dir(name)>/, then inserts the row in a DB transaction.
// If the transaction fails, the directory is removed.
func (s *classifyModelService) CreateDistilBERTWithONNXFile(ctx context.Context, name, tokenizer string, languages classifier.Languages, maxTextLength int, emailFields classifier.EmailFields, params string, onnx *multipart.FileHeader) error {
	if onnx == nil {
		return ErrClassifyModelOnnxRequired
	}
	if err := ValidateClassifyModelDisplayName(ctx, s.db, name, 0); err != nil {
		return err
	}
	absRoot, err := s.distilbertAbsModelRoot()
	if err != nil {
		return err
	}
	destDir := filepath.Join(absRoot, SanitizeClassifyModelDirStem(name))
	destOnnx := filepath.Join(destDir, distilBERTOnnxFileName)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ErrClassifyModelOnnxWriteFailed
	}
	if err := writeDistilBERTOnnxToPath(onnx, destOnnx); err != nil {
		_ = os.RemoveAll(destDir)
		return err
	}

	classLabels, err := bertmodel.ExtractDistilBERTClassLabelsFromONNX(destOnnx, s.onnxRuntimeLib)
	if err != nil {
		_ = os.RemoveAll(destDir)
		return fmt.Errorf("%w: %v", ErrClassifyModelOnnxLabelsParse, err)
	}
	if len(classLabels) < 2 {
		_ = os.RemoveAll(destDir)
		return fmt.Errorf("%w: need at least 2 classes", ErrClassifyModelOnnxLabelsParse)
	}

	model := classifier.Model{
		Name:          name,
		Algorithm:     classifier.AlgorithmDistilBERT,
		Tokenizer:     classifier.Tokenizer(tokenizer),
		Languages:     languages,
		SavePath:      destDir,
		ClassLabels:   classifier.ClassLabels(classLabels),
		MaxTextLength: maxTextLength,
		EmailFields:   emailFields,
		Enabled:       false,
		IsDeleted:     false,
		TrainStatus:   classifier.TrainStatusCompleted,
		TrainTime:     time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	applyDistilBERTParams(&model, params)

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&model).Error
	}); err != nil {
		_ = os.RemoveAll(destDir)
		return err
	}
	return nil
}

// UpdateDistilBERTWithOptionalONNXFile updates a DistilBERT row. If onnx is set, writes model.onnx (after a successful DB save,
// replaces the file from a temp path). If onnx is nil and the sanitized directory changes, renames the old directory to the new one.
func (s *classifyModelService) UpdateDistilBERTWithOptionalONNXFile(ctx context.Context, id int64, name, tokenizer string, languages classifier.Languages, maxTextLength int, emailFields classifier.EmailFields, enabled bool, params string, onnx *multipart.FileHeader) error {
	model, err := s.loadClassifyModelByID(ctx, id)
	if err != nil {
		return err
	}
	if model.Algorithm != classifier.AlgorithmDistilBERT {
		return ErrClassifyModelNotDistilBERT
	}

	model.Name = name
	model.Tokenizer = classifier.Tokenizer(tokenizer)
	model.Languages = languages
	model.MaxTextLength = maxTextLength
	model.EmailFields = emailFields
	model.Enabled = enabled
	model.UpdatedAt = time.Now()
	model.TrainStatus = classifier.TrainStatusCompleted
	model.TrainTime = time.Now()
	applyDistilBERTParams(model, params)

	if model.Enabled && onnx == nil && !classifyModelAssetsReady(model) {
		return ErrClassifyModelActivationNotReady
	}

	if err := ValidateClassifyModelDisplayName(ctx, s.db, name, model.ID); err != nil {
		return err
	}

	absRoot, err := s.distilbertAbsModelRoot()
	if err != nil {
		return err
	}
	newDir := filepath.Join(absRoot, SanitizeClassifyModelDirStem(name))
	oldDir := filepath.Clean(strings.TrimSpace(model.SavePath))

	if onnx != nil {
		if err := os.MkdirAll(newDir, 0o755); err != nil {
			return ErrClassifyModelOnnxWriteFailed
		}
		destOnnx := filepath.Join(newDir, distilBERTOnnxFileName)
		tmpOnnx := destOnnx + ".new"
		if err := writeDistilBERTOnnxToPath(onnx, tmpOnnx); err != nil {
			_ = os.Remove(tmpOnnx)
			if !classifyModelPathsEqual(oldDir, newDir) {
				_ = os.RemoveAll(newDir)
			}
			return err
		}
		classLabels, err := bertmodel.ExtractDistilBERTClassLabelsFromONNX(tmpOnnx, s.onnxRuntimeLib)
		if err != nil {
			_ = os.Remove(tmpOnnx)
			if !classifyModelPathsEqual(oldDir, newDir) {
				_ = os.RemoveAll(newDir)
			}
			return fmt.Errorf("%w: %v", ErrClassifyModelOnnxLabelsParse, err)
		}
		if len(classLabels) < 2 {
			_ = os.Remove(tmpOnnx)
			if !classifyModelPathsEqual(oldDir, newDir) {
				_ = os.RemoveAll(newDir)
			}
			return fmt.Errorf("%w: need at least 2 classes", ErrClassifyModelOnnxLabelsParse)
		}
		model.ClassLabels = classifier.ClassLabels(classLabels)
		model.SavePath = newDir
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Save(model).Error
		}); err != nil {
			_ = os.Remove(tmpOnnx)
			return err
		}
		_ = os.Remove(destOnnx)
		if err := os.Rename(tmpOnnx, destOnnx); err != nil {
			return ErrClassifyModelOnnxWriteFailed
		}
		if !classifyModelPathsEqual(oldDir, newDir) && oldDir != "" {
			_ = os.RemoveAll(oldDir)
		}
		return nil
	}

	if !classifyModelPathsEqual(oldDir, newDir) {
		if oldDir == "" {
			return ErrClassifyModelOnnxRequired
		}
		if _, err := os.Stat(oldDir); err != nil {
			return ErrClassifyModelDirRenameFailed
		}
		if err := os.Rename(oldDir, newDir); err != nil {
			return ErrClassifyModelDirRenameFailed
		}
		model.SavePath = newDir
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Save(model).Error
	})
}



