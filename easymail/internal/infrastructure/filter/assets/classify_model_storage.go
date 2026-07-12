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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RemoveClassifyModelSavePath deletes the on-disk model root for a classify model.
// SavePath is the model directory (see ClassifyModel.SavePath). If the path is empty
// or does not exist, it returns nil. Asset files referenced by absolute paths in
// ModelParams outside this directory are not removed.
func RemoveClassifyModelSavePath(raw string) error {
	p := strings.TrimSpace(raw)
	if p == "" {
		return nil
	}
	clean := filepath.Clean(p)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return fmt.Errorf("classify model save path: %w", err)
	}
	abs = filepath.Clean(abs)
	if isUnsafeClassifyModelRemovalPath(abs) {
		return fmt.Errorf("refusing to remove path %q (root or current directory)", abs)
	}
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if st.IsDir() {
		return os.RemoveAll(abs)
	}
	return os.Remove(abs)
}

func isUnsafeClassifyModelRemovalPath(abs string) bool {
	abs = filepath.Clean(abs)
	if abs == "." {
		return true
	}
	if vol := filepath.VolumeName(abs); vol != "" {
		rest := strings.TrimPrefix(abs, vol)
		rest = filepath.Clean(rest)
		return rest == "." || rest == `\` || rest == `/`
	}
	return abs == string(filepath.Separator)
}
