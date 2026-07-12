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

package cache

import (
	"sync"
	"time"
)

// MemoryTTL holds a value refreshed on TTL expiry (generic in-process cache).
type MemoryTTL[T any] struct {
	mu       sync.Mutex
	loadedAt time.Time
	valid    bool
	val      T
	err      error
}

// Get returns the cached value if valid for ttl; otherwise runs refresh and stores the result.
func (m *MemoryTTL[T]) Get(now time.Time, ttl time.Duration, refresh func() (T, error)) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.valid && ttl > 0 && now.Sub(m.loadedAt) < ttl {
		return m.val, m.err
	}
	v, err := refresh()
	m.val = v
	m.err = err
	m.loadedAt = now
	m.valid = true
	return v, err
}

// Invalidate drops the cached entry so the next Get refreshes.
func (m *MemoryTTL[T]) Invalidate() {
	m.mu.Lock()
	m.valid = false
	m.mu.Unlock()
}

