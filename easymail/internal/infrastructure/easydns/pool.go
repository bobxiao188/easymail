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

package easydns

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// connPool manages a pool of reusable DNS connections.
type connPool struct {
	mu      sync.RWMutex
	pool    map[string]*sync.Pool
	maxSize int
	timeout time.Duration
}

// newConnPool creates a new connection pool.
func newConnPool(maxSize int) *connPool {
	return &connPool{
		pool:    make(map[string]*sync.Pool),
		maxSize: maxSize,
		timeout: 30 * time.Second,
	}
}

// get retrieves a connection from the pool or creates a new one.
func (p *connPool) get(address string) (net.Conn, error) {
	p.mu.RLock()
	pool, exists := p.pool[address]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		if len(p.pool) >= p.maxSize {
			p.mu.Unlock()
			return nil, ErrPoolFull
		}
		pool = &sync.Pool{
			New: func() interface{} {
				d := net.Dialer{Timeout: p.timeout}
				conn, err := d.Dial("udp", address)
				if err != nil {
					return nil
				}
				return conn
			},
		}
		p.pool[address] = pool
		p.mu.Unlock()
	}

	conn := pool.Get()
	if conn == nil {
		return nil, ErrPoolEmpty
	}

	return conn.(net.Conn), nil
}

// close closes all connections in the pool.
func (p *connPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, pool := range p.pool {
		for {
			conn := pool.Get()
			if conn == nil {
				break
			}
			conn.(net.Conn).Close()
		}
		delete(p.pool, addr)
	}
}

// healthChecker performs periodic health checks for DNS servers.
type healthChecker struct {
	interval  time.Duration
	checkFunc func(string) bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
	isRunning atomic.Bool
}

// newHealthChecker creates a new health checker.
func newHealthChecker(interval time.Duration, checkFunc func(string) bool) *healthChecker {
	return &healthChecker{
		interval:  interval,
		checkFunc: checkFunc,
		stopChan:  make(chan struct{}),
	}
}

// start starts the health checker.
func (h *healthChecker) start() {
	if h.isRunning.Swap(true) {
		return
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.checkFunc("health")
			case <-h.stopChan:
				return
			}
		}
	}()
}

// stop stops the health checker.
func (h *healthChecker) stop() {
	if !h.isRunning.Swap(false) {
		return
	}

	close(h.stopChan)
	h.wg.Wait()
}

// checkServerHealth checks if a DNS server is healthy.
func (r *Resolver) checkServerHealth(server string) bool {
	if server == "health" {
		return r.IsHealthy()
	}

	conn, err := net.DialTimeout("udp", server, r.config.Timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Pool errors.
var (
	ErrPoolFull  = &PoolError{msg: "connection pool is full"}
	ErrPoolEmpty = &PoolError{msg: "connection pool is empty"}
)

// PoolError represents a pool-related error.
type PoolError struct {
	msg string
}

func (e *PoolError) Error() string {
	return e.msg
}
