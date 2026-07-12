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

// Package easydns provides a thread-safe, high-performance DNS resolver
// with configurable servers, timeouts, and concurrent failover.
package easydns

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Config holds DNS resolver configuration.
type Config struct {
	// Servers is a list of DNS server addresses.
	// If empty, uses system default resolvers.
	// Examples: "8.8.8.8", "1.1.1.1:53", "[2001:4860:4860::8888]:53"
	Servers []string

	// Timeout is the maximum time to wait for a DNS response.
	// Default: 5 seconds.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts per query.
	// Default: 2 (tries up to 3 servers total).
	MaxRetries int

	// EnableRoundRobin enables round-robin server selection.
	// When false, uses first-successful server.
	EnableRoundRobin bool

	// PreferGo forces Go's DNS resolver instead of system resolver.
	// Default: true
	PreferGo bool

	// EnableHealthCheck enables periodic health checks for DNS servers.
	// Default: false
	EnableHealthCheck bool

	// HealthCheckInterval is the interval for health checks when enabled.
	// Default: 30 seconds.
	HealthCheckInterval time.Duration

	// ConnectionPoolSize is the size of the connection pool per server.
	// Default: 10.
	ConnectionPoolSize int
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout:             5 * time.Second,
		MaxRetries:          2,
		EnableRoundRobin:    true,
		PreferGo:            true,
		EnableHealthCheck:   false,
		HealthCheckInterval: 30 * time.Second,
		ConnectionPoolSize:  10,
	}
}

// Resolver is a thread-safe DNS resolver with configurable servers and timeouts.
type Resolver struct {
	config      Config
	dialFunc    func(ctx context.Context, network, address string) (net.Conn, error)
	serverIdx   atomic.Uint32
	mu          sync.RWMutex
	initialized bool
	healthCheck *healthChecker
	pool        *connPool
}

// Global default resolver instance.
var defaultResolver atomic.Pointer[Resolver]
var defaultMu sync.Once

func init() {
	defaultResolver.Store(NewResolver(DefaultConfig()))
}

// NewResolver creates a DNS resolver with the given configuration.
func NewResolver(cfg Config) *Resolver {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultConfig().Timeout
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = DefaultConfig().MaxRetries
	}
	if cfg.ConnectionPoolSize == 0 {
		cfg.ConnectionPoolSize = DefaultConfig().ConnectionPoolSize
	}

	r := &Resolver{
		config:      cfg,
		initialized: true,
	}

	r.initDialFunc()
	r.pool = newConnPool(cfg.ConnectionPoolSize)

	if cfg.EnableHealthCheck {
		r.healthCheck = newHealthChecker(cfg.HealthCheckInterval, r.checkServerHealth)
		r.healthCheck.start()
	}

	return r
}

// initDialFunc sets up the DNS dial function based on configuration.
func (r *Resolver) initDialFunc() {
	if len(r.config.Servers) == 0 {
		r.dialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: r.config.Timeout}
			return d.DialContext(ctx, network, address)
		}
		return
	}

	cleaned := r.cleanServers()

	if len(cleaned) == 1 {
		host, port, _ := net.SplitHostPort(cleaned[0])
		r.dialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return r.dialWithPool(ctx, network, net.JoinHostPort(host, port))
		}
	} else {
		r.dialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return r.dialWithFailover(ctx, network, cleaned)
		}
	}
}

// cleanServers normalizes server addresses by adding default port.
func (r *Resolver) cleanServers() []string {
	cleaned := make([]string, len(r.config.Servers))
	for i, s := range r.config.Servers {
		if _, _, err := net.SplitHostPort(s); err != nil {
			cleaned[i] = net.JoinHostPort(s, "53")
		} else {
			cleaned[i] = s
		}
	}
	return cleaned
}

// dialWithPool dials using connection pool for single server.
func (r *Resolver) dialWithPool(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := r.pool.get(address)
	if err != nil {
		d := net.Dialer{Timeout: r.config.Timeout}
		return d.DialContext(ctx, network, address)
	}
	return conn, nil
}

// dialWithFailover tries multiple DNS servers with configurable strategy.
func (r *Resolver) dialWithFailover(ctx context.Context, network string, servers []string) (net.Conn, error) {
	startIdx := int(r.serverIdx.Add(1)) % len(servers)

	for i := 0; i <= r.config.MaxRetries && i < len(servers); i++ {
		idx := (startIdx + i) % len(servers)
		server := servers[idx]

		if r.config.EnableRoundRobin {
			r.serverIdx.Add(1)
		}

		conn, err := r.pool.get(server)
		if err == nil {
			return conn, nil
		}

		d := net.Dialer{Timeout: r.config.Timeout}
		conn, err = d.DialContext(ctx, network, server)
		if err == nil {
			return conn, nil
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("all DNS servers unreachable")
}

// SetServers updates the list of DNS servers at runtime.
// Thread-safe: acquires write lock.
func (r *Resolver) SetServers(servers []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config.Servers = servers
	r.initDialFunc()
}

// GetServers returns the current list of DNS servers.
// Thread-safe: acquires read lock.
func (r *Resolver) GetServers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]string, len(r.config.Servers))
	copy(servers, r.config.Servers)
	return servers
}

// GetTimeout returns the current DNS timeout.
// Thread-safe: acquires read lock.
func (r *Resolver) GetTimeout() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.config.Timeout
}

// SetTimeout updates the timeout at runtime.
// Thread-safe: acquires write lock.
func (r *Resolver) SetTimeout(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config.Timeout = timeout
	r.initDialFunc()
}

// CreateDefaultResolver creates a resolver using system DNS configuration.
func CreateDefaultResolver() *Resolver {
	return NewResolver(DefaultConfig())
}

// SetDefault replaces the global default resolver.
// Thread-safe: uses atomic pointer.
func SetDefault(r *Resolver) {
	defaultResolver.Store(r)
}

// GetDefault returns the current global default resolver.
// Thread-safe: uses atomic pointer.
func GetDefault() *Resolver {
	return defaultResolver.Load()
}

// ensureDefault ensures the default resolver is initialized.
func ensureDefault() *Resolver {
	defaultMu.Do(func() {
		if defaultResolver.Load() == nil {
			defaultResolver.Store(NewResolver(DefaultConfig()))
		}
	})
	return defaultResolver.Load()
}

// LookupTXT performs a TXT record lookup with context and timeout support.
// Thread-safe: uses the resolver's dial function.
func (r *Resolver) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	if r == nil {
		r = ensureDefault()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	resolver := &net.Resolver{
		PreferGo: r.config.PreferGo,
		Dial:     r.dialFunc,
	}

	return resolver.LookupTXT(ctx, domain)
}

// LookupAddr performs a reverse DNS lookup.
func (r *Resolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	if r == nil {
		r = ensureDefault()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	resolver := &net.Resolver{
		PreferGo: r.config.PreferGo,
		Dial:     r.dialFunc,
	}

	return resolver.LookupAddr(ctx, addr)
}

// LookupIPAddr performs an IP address lookup.
func (r *Resolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	if r == nil {
		r = ensureDefault()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	resolver := &net.Resolver{
		PreferGo: r.config.PreferGo,
		Dial:     r.dialFunc,
	}

	return resolver.LookupIPAddr(ctx, host)
}

// LookupMX performs an MX record lookup.
func (r *Resolver) LookupMX(ctx context.Context, domain string) ([]*net.MX, error) {
	if r == nil {
		r = ensureDefault()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	resolver := &net.Resolver{
		PreferGo: r.config.PreferGo,
		Dial:     r.dialFunc,
	}

	return resolver.LookupMX(ctx, domain)
}

// LookupNS performs an NS record lookup.
func (r *Resolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	if r == nil {
		r = ensureDefault()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	resolver := &net.Resolver{
		PreferGo: r.config.PreferGo,
		Dial:     r.dialFunc,
	}

	return resolver.LookupNS(ctx, domain)
}

// LookupSRV performs an SRV record lookup.
func (r *Resolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	if r == nil {
		r = ensureDefault()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	resolver := &net.Resolver{
		PreferGo: r.config.PreferGo,
		Dial:     r.dialFunc,
	}

	return resolver.LookupSRV(ctx, service, proto, name)
}

// LookupCNAME performs a CNAME lookup.
func (r *Resolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	if r == nil {
		r = ensureDefault()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	resolver := &net.Resolver{
		PreferGo: r.config.PreferGo,
		Dial:     r.dialFunc,
	}

	return resolver.LookupCNAME(ctx, host)
}

// String returns a human-readable representation of the resolver.
func (r *Resolver) String() string {
	if r == nil {
		return "easydns.Resolver(nil)"
	}

	servers := r.GetServers()
	if len(servers) == 0 {
		return "easydns.Resolver(system-default)"
	}
	if len(servers) == 1 {
		return fmt.Sprintf("easydns.Resolver(%s)", servers[0])
	}
	return fmt.Sprintf("easydns.Resolver(%v)", servers)
}

// IsHealthy checks if the resolver can reach at least one DNS server.
func (r *Resolver) IsHealthy() bool {
	if r == nil {
		return false
	}

	servers := r.GetServers()
	if len(servers) == 0 {
		return true
	}

	for _, server := range servers {
		conn, err := net.DialTimeout("udp", server, r.config.Timeout)
		if err == nil {
			conn.Close()
			return true
		}
	}

	return false
}
