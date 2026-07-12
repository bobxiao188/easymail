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

// Package servicelog 涓哄悇涓氬姟妯″潡鎻愪緵鍙鐢ㄧ殑鏃ュ織锛氬彲閫夌嫭绔嬫枃浠舵棩蹇? module 瀛楁锛屽惁鍒欏洖閫€鍒颁富杩涚▼鍏变韩鏃ュ織
package servicelog

import (
	"fmt"
	"strings"

	"easymail/pkg/config"
	"easymail/pkg/logger/easylog"
)

func Open(shared *easylog.Logger, cfg config.ServiceLogConfig, module string) (*easylog.Logger, error) {
	name := strings.TrimSpace(module)
	if cfg.Enable && strings.TrimSpace(cfg.File) != "" {
		lg, err := easylog.New(cfg.File)
		if err != nil {
			return nil, fmt.Errorf("servicelog %s: %w", name, err)
		}
		lg.SetLevelString(cfg.Level)
		return lg.WithModule(name), nil
	}
	if shared == nil {
		lg, err := easylog.New("")
		if err != nil {
			return nil, err
		}
		return lg.WithModule(name), nil
	}
	return shared.WithModule(name), nil
}
