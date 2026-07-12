package persistence

import (
	"context"
	"fmt"
	"sync"
)

// RedisProvider is the abstract port for Redis connections.
type RedisProvider interface {
	// Ping checks connectivity.
	Ping(ctx context.Context) error

	// Close shuts down the connection.
	Close() error
}

// RedisFactory opens a RedisProvider from an address string.
type RedisFactory interface {
	Driver() string
	Open(ctx context.Context, addr string) (RedisProvider, error)
}

var (
	redisMu        sync.RWMutex
	redisFactories = map[string]RedisFactory{}
)

func RegisterRedis(f RedisFactory) {
	redisMu.Lock()
	redisFactories[f.Driver()] = f
	redisMu.Unlock()
}

func OpenRedis(ctx context.Context, driver, addr string) (RedisProvider, error) {
	redisMu.RLock()
	f, ok := redisFactories[driver]
	redisMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("redis driver %q not registered", driver)
	}
	return f.Open(ctx, addr)
}
