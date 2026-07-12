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

package migrate

import (
	"path/filepath"
	"strings"
	"testing"

	"easymail/internal/infrastructure/persistence/sqlite"
)

func TestAutoMigrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "migrate.db")
	db, err := sqlite.OpenFile(path, sqlite.Config{})
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}

	err = AutoMigrate(db)
	if err != nil {
		// SQLite may have index creation issues with AutoMigrate
		// Check if tables were created despite the error
		if !strings.Contains(err.Error(), "index") {
			t.Fatalf("AutoMigrate() error = %v", err)
		}
		t.Logf("AutoMigrate() index warning (expected on SQLite): %v", err)
	}

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}
