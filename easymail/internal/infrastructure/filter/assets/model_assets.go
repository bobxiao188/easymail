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

package assets

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"easymail/internal/domain/filter/classifier"
)

// Repository implements classifier.ModelAssetRepository using the local filesystem.
type Repository struct{}

func (Repository) AssetsReady(m classifier.Model) bool {
	switch m.Algorithm {
	case classifier.AlgorithmDistilBERT:
		onnxPath, err := DistilBERTAssetPaths(m)
		if err != nil {
			return false
		}
		return onnxLikelyHasEmbeddedDistilBERTAssets(onnxPath)
	case classifier.AlgorithmFastText:
		return fastTextAssetPresent(m)
	case classifier.AlgorithmXGBoost:
		return false
	default:
		return false
	}
}

func DistilBERTAssetPaths(m classifier.Model) (string, error) {
	baseDir := filepath.Clean(strings.TrimSpace(m.SavePath))
	if baseDir == "" {
		return "", errors.New("empty save_path")
	}
	fi, err := os.Stat(baseDir)
	if err != nil {
		return "", fmt.Errorf("save_path: %w", err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("save_path must be the model directory (is_dir=false): %s", baseDir)
	}
	p := m.Params
	onnxPath := joinClassifyAssetPath(baseDir, p.ModelFile, "model.onnx")
	stOnnx, errOnnx := os.Stat(onnxPath)
	if errOnnx != nil || stOnnx.IsDir() {
		mf := strings.TrimSpace(p.ModelFile)
		return "", fmt.Errorf("missing onnx model at %s (save_path=%q params.modelFile=%q; default model.onnx)",
			onnxPath, baseDir, mf)
	}
	return onnxPath, nil
}

func joinClassifyAssetPath(baseDir, rel, defaultName string) string {
	r := strings.TrimSpace(rel)
	if r == "" {
		r = defaultName
	}
	if filepath.IsAbs(r) {
		return filepath.Clean(r)
	}
	return filepath.Join(baseDir, r)
}

func fastTextAssetPresent(m classifier.Model) bool {
	base := strings.TrimSpace(m.SavePath)
	if base == "" {
		return false
	}
	clean := filepath.Clean(base)
	fi, err := os.Stat(clean)
	if err != nil {
		return false
	}
	if !fi.IsDir() {
		n := strings.ToLower(filepath.Base(clean))
		return strings.HasSuffix(n, ".bin") || strings.HasSuffix(n, ".ftz")
	}
	entries, err := os.ReadDir(clean)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := strings.ToLower(e.Name())
		if strings.HasSuffix(n, ".bin") || strings.HasSuffix(n, ".ftz") {
			return true
		}
	}
	return false
}

func onnxLikelyHasEmbeddedDistilBERTAssets(onnxPath string) bool {
	b, err := os.ReadFile(onnxPath)
	if err != nil || len(b) < 128 {
		return false
	}
	return bytes.Contains(b, []byte("vocab_txt")) &&
		bytes.Contains(b, []byte("tokenizer_config_json")) &&
		bytes.Contains(b, []byte("id2label"))
}


