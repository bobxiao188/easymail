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

package distilbert

import (
	"strings"
	"sync"

	inframodel "easymail/internal/infrastructure/model"
)

var onnxLibMu sync.RWMutex
var onnxLibPath string

// SetONNXRuntimeLib stores YAML classify_model_service.onnx_runtime_lib for lazy DistilBERT ONNX init.
func SetONNXRuntimeLib(path string) {
	onnxLibMu.Lock()
	defer onnxLibMu.Unlock()
	onnxLibPath = strings.TrimSpace(path)
}

func ensureONNXRuntime() error {
	onnxLibMu.RLock()
	p := onnxLibPath
	onnxLibMu.RUnlock()
	return inframodel.InitONNXRuntime(p)
}
