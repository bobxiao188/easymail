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

package mysql

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mysql_test.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	})
	return db
}

func TestNewStaticDB(t *testing.T) {
	db := newTestDB(t)
	provider := NewStaticDB(db)
	if provider == nil {
		t.Fatal("NewStaticDB() returned nil")
	}

	gotDB, err := provider.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if gotDB != db {
		t.Error("DB() returned different instance")
	}
}

func TestGormDB(t *testing.T) {
	db := newTestDB(t)
	provider := NewStaticDB(db)

	gotDB, err := GormDB(context.Background(), provider)
	if err != nil {
		t.Fatalf("GormDB() error = %v", err)
	}
	if gotDB == nil {
		t.Fatal("GormDB() returned nil")
	}
}

func TestGormDB_NilProvider(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil provider")
		}
	}()
	// GormDB doesn't check for nil provider, will panic
	GormDB(context.Background(), nil)
}

func TestStaticDBProvider(t *testing.T) {
	db := newTestDB(t)
	provider := NewStaticDB(db)

	var _ DBProvider = provider

	s := provider.(*staticDB)
	if s.db != db {
		t.Error("staticDB.db not set correctly")
	}
}
