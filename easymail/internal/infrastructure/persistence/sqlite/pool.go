package sqlite

import (
	"sync"

	"gorm.io/gorm"
)

// Pool caches *gorm.DB connections keyed by SQLite file path.
type Pool struct {
	cfg Config
	mu  sync.RWMutex
	dbs map[string]*gorm.DB
}

func NewPool(cfg Config) *Pool {
	return &Pool{cfg: cfg, dbs: make(map[string]*gorm.DB)}
}

// DB returns a cached *gorm.DB for the given path, opening and migrating it if needed.
func (p *Pool) DB(path string) (*gorm.DB, error) {
	// Fast path: read-lock to check cache.
	p.mu.RLock()
	db, ok := p.dbs[path]
	p.mu.RUnlock()
	if ok {
		return db, nil
	}

	// Slow path: open and migrate, then store.
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock.
	if db, ok := p.dbs[path]; ok {
		return db, nil
	}

	db, err := OpenFile(path, p.cfg)
	if err != nil {
		return nil, err
	}
	// Auto-migrate the message schema for per-mailbox SQLite.
	if err := db.AutoMigrate(&EmailPO{}, &FolderPO{}, &ContactPO{}, &ContactGroupPO{}); err != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}
	p.dbs[path] = db
	return db, nil
}

// CloseDB closes and removes a single database from the pool by path.
func (p *Pool) CloseDB(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	db, ok := p.dbs[path]
	if !ok {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		delete(p.dbs, path)
		return err
	}
	err = sqlDB.Close()
	delete(p.dbs, path)
	return err
}

// Close shuts down all cached connections.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var first error
	for path, db := range p.dbs {
		sqlDB, err := db.DB()
		if err != nil {
			first = err
			continue
		}
		if err := sqlDB.Close(); err != nil && first == nil {
			first = err
		}
		delete(p.dbs, path)
	}
	return first
}
