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

package imap

import (
	"crypto/tls"
	"net"
	"sync"
)

// Logger adapts easylog-style Printf for the IMAP server.
type Logger interface {
	Printf(format string, args ...interface{})
}

// ServerOptions configures the built-in IMAP server.
type ServerOptions struct {
	NewMailSession func() *MailSession
	Logger         Logger
	// MaxConcurrent limits simultaneous IMAP connections; 0 means unlimited.
	MaxConcurrent int
	// TLSConfig enables direct TLS (IMAPS on port 993) when non-nil.
	// When nil, the server uses plain TCP and STARTTLS can upgrade per-connection.
	TLSConfig *tls.Config
	// StartTLSConfig enables STARTTLS upgrade; when non-nil, STARTTLS is advertised in CAPABILITY.
	// For direct TLS (port 993), set TLSConfig. For STARTTLS (port 143), set StartTLSConfig.
	StartTLSConfig *tls.Config
	// Debug enables verbose protocol-level logging (incoming commands, outgoing responses).
	Debug bool
}

// Server implements IMAP4rev2 (RFC 9051) with IDLE (RFC 2177) and STARTTLS support.
type Server struct {
	opts     ServerOptions
	mu       sync.Mutex
	closed   bool
	sem      chan struct{}
}

// NewServer returns a server instance; call Serve with a listener.
func NewServer(opts *ServerOptions) *Server {
	s := &Server{opts: *opts}
	if opts.MaxConcurrent > 0 {
		s.sem = make(chan struct{}, opts.MaxConcurrent)
	}
	return s
}

// Serve accepts connections on ln and runs one MailSession per connection (blocks).
func (s *Server) Serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return nil
			}
			return err
		}
		// If direct TLS is configured, wrap the raw conn immediately.
		if s.opts.TLSConfig != nil {
			c = tls.Server(c, s.opts.TLSConfig)
		}
		ms := s.opts.NewMailSession()
		if s.sem != nil {
			s.sem <- struct{}{}
		}
		go func(conn net.Conn, session *MailSession) {
			if s.sem != nil {
				defer func() { <-s.sem }()
			}
			handleConn(conn, session, s.opts.Logger, s.opts.StartTLSConfig != nil, s.opts.Debug)
		}(c, ms)
	}
}

// Close marks the server closed; closing the listener ends Accept.
func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}

