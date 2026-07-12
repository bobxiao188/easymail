/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * License: AGPLv3
 */

package antivirus

import "time"

// Config holds connection parameters for the ClamAV daemon (clamd).
type Config struct {
	// Addr is the clamd endpoint in host:port form (e.g. "127.0.0.1:3310").
	Addr string

	// Timeout for network read/write operations.
	Timeout time.Duration

	// ConnectTimeout for the initial TCP handshake.
	ConnectTimeout time.Duration

	// Enable toggles the ClamAV integration.
	Enable bool

	// MaxScanSize limits the maximum bytes sent to clamd per call (0 = use clamd default).
	MaxScanSize int

	// ScanEmailAttachments enables scanning of email attachments.
	ScanEmailAttachments bool

	// ScanEmailBody enables scanning of the raw email body.
	ScanEmailBody bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Addr:                 "127.0.0.1:3310",
		Timeout:              30 * time.Minute,
		ConnectTimeout:       5 * time.Second,
		Enable:               false,
		MaxScanSize:          20 * 1024 * 1024,
		ScanEmailAttachments: true,
		ScanEmailBody:        false,
	}
}
