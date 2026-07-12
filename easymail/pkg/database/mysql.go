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
	"os"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// 鍙€夌幆澧冨彉閲忥紙鏈缃垯浣跨敤榛樿鍊硷級锛欵ASYMAIL_MYSQL_DIAL_TIMEOUT銆丒ASYMAIL_MYSQL_READ_TIMEOUT銆丒ASYMAIL_MYSQL_WRITE_TIMEOUT銆丒ASYMAIL_GORM_SLOW_QUERY_LOG
const (
	defaultMySQLDialTimeout  = 30 * time.Second
	defaultMySQLReadTimeout  = 60 * time.Second
	defaultMySQLWriteTimeout = 60 * time.Second
	defaultMaxOpenConns      = 50
	defaultMaxIdleConns      = 10
	defaultConnMaxLifetime   = time.Hour
	defaultConnMaxIdleTime   = 10 * time.Minute
)

// hasMySQLQueryParam 浠呯敤ParseDSN 澶辫触鏃剁殑鍏滃簳鎷兼帴
func hasMySQLQueryParam(dsnLower, paramLower string) bool {
	p := paramLower + "="
	return strings.Contains(dsnLower, "?"+p) || strings.Contains(dsnLower, "&"+p)
}

func legacyNormalizeMySQLDSN(dsn string) string {
	lower := strings.ToLower(dsn)
	readTO := getenvDuration("EASYMAIL_MYSQL_READ_TIMEOUT", defaultMySQLReadTimeout)
	writeTO := getenvDuration("EASYMAIL_MYSQL_WRITE_TIMEOUT", defaultMySQLWriteTimeout)
	dialTO := getenvDuration("EASYMAIL_MYSQL_DIAL_TIMEOUT", defaultMySQLDialTimeout)

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	var b strings.Builder
	b.WriteString(dsn)
	if !hasMySQLQueryParam(lower, "timeout") {
		b.WriteString(sep)
		b.WriteString("timeout=")
		b.WriteString(dialTO.String())
		sep = "&"
	}
	if !hasMySQLQueryParam(lower, "readtimeout") {
		b.WriteString(sep)
		b.WriteString("readTimeout=")
		b.WriteString(readTO.String())
		sep = "&"
	}
	if !hasMySQLQueryParam(lower, "writetimeout") {
		b.WriteString(sep)
		b.WriteString("writeTimeout=")
		b.WriteString(writeTO.String())
	}
	return b.String()
}

func normalizeMySQLDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return dsn
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return legacyNormalizeMySQLDSN(dsn)
	}
	dial := getenvDuration("EASYMAIL_MYSQL_DIAL_TIMEOUT", defaultMySQLDialTimeout)
	readTO := getenvDuration("EASYMAIL_MYSQL_READ_TIMEOUT", defaultMySQLReadTimeout)
	writeTO := getenvDuration("EASYMAIL_MYSQL_WRITE_TIMEOUT", defaultMySQLWriteTimeout)

	if cfg.Timeout < dial {
		cfg.Timeout = dial
	}
	if cfg.ReadTimeout < readTO {
		cfg.ReadTimeout = readTO
	}
	if cfg.WriteTimeout < writeTO {
		cfg.WriteTimeout = writeTO
	}
	cfg.CheckConnLiveness = true
	if cfg.Params != nil {
		delete(cfg.Params, "timeout")
		delete(cfg.Params, "readTimeout")
		delete(cfg.Params, "writeTimeout")
	}
	return cfg.FormatDSN()
}

func getenvDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

// gormSlowThreshold 鑾峰彇GORM鎱㈡煡璇㈤槇鍊硷紝榛樿 2s
func gormSlowThreshold() time.Duration {
	return getenvDuration("EASYMAIL_GORM_SLOW_QUERY_LOG", 2000*time.Millisecond)
}

// initMySQL 鍒濆鍖朚ySQL杩炴帴
func initMySQL(dsn string) error {
	dsn = normalizeMySQLDSN(dsn)

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             gormSlowThreshold(),
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		NowFunc:     func() time.Time { return time.Now().Local() },
		PrepareStmt: true,
	})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(defaultMaxOpenConns)
	sqlDB.SetMaxIdleConns(defaultMaxIdleConns)
	sqlDB.SetConnMaxLifetime(defaultConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(context.Background(), defaultMySQLDialTimeout+5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("mysql ping: %w", err)
	}
	return nil
}

// GetDB returns the global database connection.
func GetDB() *gorm.DB {
	return db
}
