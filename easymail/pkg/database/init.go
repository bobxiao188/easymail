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

package database

import (
	"context"
	"fmt"
	"log"
	"sync"

	"easymail/pkg/config"
)

var initOnce sync.Once
var initErr error
var appConfig *config.AppConfig

func Initialize(configPath string) (*config.AppConfig, error) {
	initOnce.Do(func() {
		appConfig, initErr = config.ReadAppConfig(configPath)
		if initErr != nil {
			initErr = fmt.Errorf("read app config failed: %w", initErr)
			return
		}
		if initErr = config.ValidateConfig(appConfig); initErr != nil {
			initErr = fmt.Errorf("validate app config failed: %w", initErr)
			return
		}
		switch appConfig.DBDriver {
		case "sqlite3", "postgres":
			if initErr = initDB(context.Background(), appConfig.DBDriver, appConfig.DSN); initErr != nil {
				initErr = fmt.Errorf("connect %s failed: %w", appConfig.DBDriver, initErr)
				return
			}
		case "", "mysql":
			if initErr = initMySQL(appConfig.DSN); initErr != nil {
				initErr = fmt.Errorf("connect mysql failed: %w", initErr)
				return
			}
		default:
			initErr = fmt.Errorf("unsupported database driver: %s", appConfig.DBDriver)
			return
		}
		if appConfig.Redis.Addr != "" {
			if appConfig.Redis.Required {
				// Blocking: fail fast when Redis is required but unreachable
				if err := initRedisWithPassword(appConfig.Redis.Addr, appConfig.Redis.Password); err != nil {
					initErr = fmt.Errorf("connect redis failed: %w", err)
					return
				}
			} else {
				log.Println("INFO: redis not required, using memory fallback")
			}
		}
	})
	return appConfig, initErr
}

func GetAppConfig() *config.AppConfig {
	return appConfig
}
