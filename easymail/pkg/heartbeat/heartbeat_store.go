package heartbeat

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// HeartbeatStore is the storage abstraction for heartbeat status.
// Implementations: RedisHeartbeatStore, MemoryHeartbeatStore.
type HeartbeatStore interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Close() error
}

// ---------------------------------------------------------------------------
// RedisHeartbeatStore — backed by go-redis
// ---------------------------------------------------------------------------

// RedisHeartbeatStore wraps *redis.Client to satisfy HeartbeatStore.
type RedisHeartbeatStore struct {
	rdb *redis.Client
}

// NewRedisHeartbeatStore creates a HeartbeatStore backed by Redis.
func NewRedisHeartbeatStore(rdb *redis.Client) *RedisHeartbeatStore {
	return &RedisHeartbeatStore{rdb: rdb}
}

func (s *RedisHeartbeatStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return s.rdb.Set(ctx, key, value, ttl).Err()
}

func (s *RedisHeartbeatStore) Get(ctx context.Context, key string) (string, error) {
	return s.rdb.Get(ctx, key).Result()
}

func (s *RedisHeartbeatStore) Close() error {
	return s.rdb.Close()
}

// ---------------------------------------------------------------------------
// MemoryHeartbeatStore — in-process map with TTL
// ---------------------------------------------------------------------------

type memHeartbeatItem struct {
	value     string
	expiresAt time.Time
}

// MemoryHeartbeatStore stores heartbeat status in local memory.
// Data is per-process and lost on restart.
type MemoryHeartbeatStore struct {
	mu   sync.RWMutex
	data map[string]memHeartbeatItem
	done chan struct{}
}

// NewMemoryHeartbeatStore creates an in-memory heartbeat store with periodic
// expired-key cleanup. Call Close() to stop the cleanup goroutine.
func NewMemoryHeartbeatStore() *MemoryHeartbeatStore {
	s := &MemoryHeartbeatStore{
		data: make(map[string]memHeartbeatItem),
		done: make(chan struct{}),
	}
	go s.evictLoop()
	return s
}

func (s *MemoryHeartbeatStore) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.evictExpired()
		case <-s.done:
			return
		}
	}
}

func (s *MemoryHeartbeatStore) evictExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.data {
		if now.After(v.expiresAt) {
			delete(s.data, k)
		}
	}
}

func (s *MemoryHeartbeatStore) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
	s.mu.Lock()
	s.data[key] = memHeartbeatItem{
		value:     value.(string),
		expiresAt: time.Now().Add(ttl),
	}
	s.mu.Unlock()
	return nil
}

func (s *MemoryHeartbeatStore) Get(_ context.Context, key string) (string, error) {
	s.mu.RLock()
	item, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return "", redis.Nil
	}
	if time.Now().After(item.expiresAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return "", redis.Nil
	}
	return item.value, nil
}

func (s *MemoryHeartbeatStore) Close() error {
	close(s.done)
	s.mu.Lock()
	s.data = make(map[string]memHeartbeatItem)
	s.mu.Unlock()
	return nil
}