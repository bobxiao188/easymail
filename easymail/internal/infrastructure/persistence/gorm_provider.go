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

package persistence

import (
	"context"

	"gorm.io/gorm"
)

// GormProvider is a DBProvider backed by a *gorm.DB connection.
type GormProvider struct {
	gdb *gorm.DB
}

func NewGormProvider(gdb *gorm.DB) *GormProvider {
	return &GormProvider{gdb: gdb}
}

func (p *GormProvider) DB(_ context.Context) (any, error) {
	return p.gdb, nil
}

func (p *GormProvider) Ping(ctx context.Context) error {
	sqlDB, err := p.gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (p *GormProvider) Close() error {
	sqlDB, err := p.gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
