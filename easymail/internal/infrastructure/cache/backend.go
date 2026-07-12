package cache

import (
	"context"
	"time"
)

// CacheBackend is a pluggable cache backend.
// Implementations: MemoryBackend, RedisBackend, etc.
type CacheBackend interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Close() error
}

