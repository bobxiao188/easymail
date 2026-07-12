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

package postgres

import (
	"context"
	"time"

	"easymail/internal/infrastructure/persistence"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultMaxOpenConns    = 50
	defaultMaxIdleConns    = 10
	defaultConnMaxLifetime = time.Hour
	defaultConnMaxIdleTime = 10 * time.Minute
)

func init() {
	persistence.Register(ConnectionFactory{})
}

type ConnectionFactory struct{}

func (ConnectionFactory) Driver() string { return "postgres" }

func (ConnectionFactory) Open(ctx context.Context, dsn string) (persistence.DBProvider, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Warn),
		NowFunc:     func() time.Time { return time.Now().Local() },
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(defaultMaxOpenConns)
	sqlDB.SetMaxIdleConns(defaultMaxIdleConns)
	sqlDB.SetConnMaxLifetime(defaultConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	return persistence.NewGormProvider(gdb), nil
}
