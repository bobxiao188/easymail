package cache

import (
	"context"
	"sync"
	"time"
)

type memItem struct {
	data      []byte
	expiresAt time.Time
}

type MemoryBackend struct {
	mu    sync.RWMutex
	data  map[string]memItem
	done  chan struct{}
	close sync.Once
}

// NewMemoryBackend creates an in-memory cache with periodic expired-key cleanup.
// The cleanup runs every 5 minutes. Call Close() to stop the goroutine.
func NewMemoryBackend() *MemoryBackend {
	m := &MemoryBackend{
		data: make(map[string]memItem),
		done: make(chan struct{}),
	}
	go m.evictLoop()
	return m
}

func (m *MemoryBackend) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.evictExpired()
		case <-m.done:
			return
		}
	}
}

func (m *MemoryBackend) evictExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, v := range m.data {
		if now.After(v.expiresAt) {
			delete(m.data, k)
		}
	}
}

func (m *MemoryBackend) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	item, ok := m.data[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if time.Now().After(item.expiresAt) {
		m.mu.Lock()
		delete(m.data, key)
		m.mu.Unlock()
		return nil, false, nil
	}
	return item.data, true, nil
}

func (m *MemoryBackend) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	m.data[key] = memItem{data: value, expiresAt: time.Now().Add(ttl)}
	m.mu.Unlock()
	return nil
}

func (m *MemoryBackend) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	for _, k := range keys {
		delete(m.data, k)
	}
	m.mu.Unlock()
	return nil
}

func (m *MemoryBackend) Close() error {
	m.close.Do(func() {
		close(m.done)
	})
	m.mu.Lock()
	m.data = make(map[string]memItem)
	m.mu.Unlock()
	return nil
}
