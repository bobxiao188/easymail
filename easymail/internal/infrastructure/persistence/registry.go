package persistence

import (
	"context"
	"fmt"
	"sync"
)

// Factory opens a DBProvider from a connection string.
type Factory interface {
	// Driver returns the driver name (e.g. "mysql", "sqlite3").
	Driver() string

	// Open establishes a connection and returns a DBProvider.
	Open(ctx context.Context, dsn string) (DBProvider, error)
}

var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

// Register registers a database driver factory.
// Call from init() or from the assembly point before Open().
func Register(f Factory) {
	mu.Lock()
	factories[f.Driver()] = f
	mu.Unlock()
}

// Open opens a database connection using a registered driver.
func Open(ctx context.Context, driver, dsn string) (DBProvider, error) {
	mu.RLock()
	f, ok := factories[driver]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("database driver %q not registered; import the driver package or register it first", driver)
	}
	return f.Open(ctx, dsn)
}
