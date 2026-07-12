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

	"easymail/internal/infrastructure/persistence"
	_ "easymail/internal/infrastructure/persistence/mysql"
	_ "easymail/internal/infrastructure/persistence/postgres"
	_ "easymail/internal/infrastructure/persistence/sqlite3"

	"gorm.io/gorm"
)

// initDB opens a database connection through the registered persistence factory.
// Supported drivers: mysql, postgres, sqlite3 (default: mysql).
// The underlying *gorm.DB is stored in the package-level db variable.
func initDB(ctx context.Context, driver, dsn string) error {
	prov, err := persistence.Open(ctx, driver, dsn)
	if err != nil {
		return fmt.Errorf("open %s: %w", driver, err)
	}

	raw, err := prov.DB(ctx)
	if err != nil {
		return fmt.Errorf("get db from provider: %w", err)
	}

	gdb, ok := raw.(*gorm.DB)
	if !ok {
		return fmt.Errorf("persistence provider returned non-GORM connection")
	}

	db = gdb
	return nil
}
