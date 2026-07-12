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
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveClassifyModelSavePath_empty(t *testing.T) {
	if err := RemoveClassifyModelSavePath("  "); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveClassifyModelSavePath_missing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nope")
	if err := RemoveClassifyModelSavePath(p); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveClassifyModelSavePath_dir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "model_root")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(sub, "model.onnx")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveClassifyModelSavePath(sub); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Fatalf("expected dir removed, stat err=%v", err)
	}
}

func TestRemoveClassifyModelSavePath_file(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "single.bin")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveClassifyModelSavePath(f); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err=%v", err)
	}
}

func TestIsUnsafeClassifyModelRemovalPath(t *testing.T) {
	if !isUnsafeClassifyModelRemovalPath(string(filepath.Separator)) {
		t.Fatal("root should be unsafe")
	}
	if !isUnsafeClassifyModelRemovalPath(".") {
		t.Fatal(". should be unsafe")
	}
}
