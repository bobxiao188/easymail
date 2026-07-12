package sqlite

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// OpenFile opens a SQLite file at the given path and applies PRAGMA settings.
func OpenFile(path string, cfg Config) (*gorm.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path is empty")
	}
	// Ensure parent directory exists
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.maxOpen())
	// WAL mode allows concurrent readers; use a reasonable idle pool.
	sqlDB.SetMaxIdleConns(cfg.maxIdle())
	// Ensure idle connections are recycled periodically.
	// This is particularly important for SQLite to avoid stale WAL readers.

	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}
	if cfg.WAL {
		if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("pragma journal_mode: %w", err)
		}
	}
	if _, err := sqlDB.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", cfg.busyTimeout())); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	return db, nil
}
