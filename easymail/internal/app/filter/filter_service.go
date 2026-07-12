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

package filter

import (
	"context"
	"net"
	"net/textproto"
	"time"
)

// ScanRequest carries all per-message context for a single scan.
type ScanRequest struct {
	// SMTP session metadata
	ConnectIP     net.IP
	ConnectHost   string
	ConnectFamily string
	ConnectPort   uint16
	HeloName      string
	MailFrom      string
	RcptTo        []string

	// Message content
	Headers   textproto.MIMEHeader
	Body      []byte
	BodyText  string
	BodyHTML  string

	// Runtime identifiers
	TraceID string
	QueueID string
	// StartedAt is when the SMTP transaction began (for time-bucketing and timeouts).
	StartedAt time.Time
}

// ScanResult is the outcome of a full scan pipeline.
type ScanResult struct {
	Action      string // accept | spam | quarantine | reject
	RuleID      int64  // 0 means default policy matched
	FeatureSnap string
	TraceJSON   string
	Duration    time.Duration
}

// FilterService is the primary anti-spam / anti-virus scanning API for mail delivery.
// It orchestrates:
//  1. Feature extraction (FeatureEngine in domain/filter/feature)
//  2. Rule evaluation (RuleEngine)
//  3. Antivirus scanning (AntivirusService)
type FilterService interface {
	// Scan runs the full email scanning pipeline and returns the result.
	Scan(ctx context.Context, req *ScanRequest) (*ScanResult, error)

	// ScanStage runs scanning only for a specific SMTP pipeline stage (connect, helo, etc.).
	// This allows milter-style incremental evaluation.
	ScanStage(ctx context.Context, stage string, req *ScanRequest) (*ScanResult, error)
}
