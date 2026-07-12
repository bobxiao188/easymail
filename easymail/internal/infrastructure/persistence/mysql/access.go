package mysql

import (
	"context"
	"errors"

	"easymail/internal/infrastructure/persistence"
	"gorm.io/gorm"
)

// ErrDatabaseNotReady is returned when the database is not initialized.
var ErrDatabaseNotReady = errors.New("database not initialized")

// DBProvider provides a *gorm.DB connection.
type DBProvider interface {
	DB() (*gorm.DB, error)
}

type staticDB struct {
	db *gorm.DB
}

// NewStaticDB returns a DBProvider backed by a GORM connection.
func NewStaticDB(db *gorm.DB) DBProvider {
	return &staticDB{db: db}
}

func (s *staticDB) DB() (*gorm.DB, error) {
	if s.db == nil {
		return nil, ErrDatabaseNotReady
	}
	return s.db, nil
}

// staticDBPersistenceAdapter wraps staticDB to satisfy persistence.DBProvider.
type staticDBPersistenceAdapter struct {
	inner *staticDB
}

func (a *staticDBPersistenceAdapter) DB(ctx context.Context) (any, error) {
	g, err := a.inner.DB()
	if err != nil {
		return nil, err
	}
	return g.WithContext(ctx), nil
}

func (a *staticDBPersistenceAdapter) Ping(ctx context.Context) error {
	g, err := a.inner.DB()
	if err != nil {
		return err
	}
	sqlDB, err := g.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (a *staticDBPersistenceAdapter) Close() error {
	g, err := a.inner.DB()
	if err != nil {
		return err
	}
	sqlDB, err := g.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// NewPersistenceDBProvider wraps a DBProvider to satisfy persistence.DBProvider.
func NewPersistenceDBProvider(p DBProvider) persistence.DBProvider {
	if a, ok := p.(*staticDB); ok {
		return &staticDBPersistenceAdapter{inner: a}
	}
	// Fallback: wrap using reflection-like approach
	return &genericDBAdapter{inner: p}
}

type genericDBAdapter struct {
	inner DBProvider
}

func (a *genericDBAdapter) DB(ctx context.Context) (any, error) {
	g, err := a.inner.DB()
	if err != nil {
		return nil, err
	}
	return g.WithContext(ctx), nil
}

func (a *genericDBAdapter) Ping(ctx context.Context) error {
	g, err := a.inner.DB()
	if err != nil {
		return err
	}
	sqlDB, err := g.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (a *genericDBAdapter) Close() error {
	g, err := a.inner.DB()
	if err != nil {
		return err
	}
	sqlDB, err := g.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// gormDB extracts *gorm.DB with context binding.
func gormDB(ctx context.Context, p DBProvider) (*gorm.DB, error) {
	d, err := p.DB()
	if err != nil {
		return nil, err
	}
	return d.WithContext(ctx), nil
}

// GormDB returns *gorm.DB with context binding from mysql.DBProvider.
func GormDB(ctx context.Context, p DBProvider) (*gorm.DB, error) {
	return gormDB(ctx, p)
}

// GormDBFromProvider extracts a *gorm.DB with context binding from an abstract persistence.DBProvider.
func GormDBFromProvider(ctx context.Context, p persistence.DBProvider) (*gorm.DB, error) {
	conn, err := p.DB(ctx)
	if err != nil {
		return nil, err
	}
	gdb, ok := conn.(*gorm.DB)
	if !ok {
		return nil, errors.New("DBProvider is not backed by GORM")
	}
	return gdb, nil
}
