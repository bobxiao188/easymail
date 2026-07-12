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
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

// Attachment is a domain value object for email attachment metadata.
type Attachment struct {
	FileName    string
	ContentType string
	Content     []byte
}

// FeatureBatch is extracted numeric features for rule evaluation (bool as 0/1).
type FeatureBatch map[string]float64

// Stage is an ordered milter / SMTP pipeline phase (distinct from Go's context.Context).
type Stage int

const (
	StageUnknown Stage = -1
	StageConnect Stage = iota
	StageHelo
	StageMailFrom
	StageRcptTo
	StageHeaders
	StageBody
)

// MaxStage returns the later pipeline phase.
func MaxStage(a, b Stage) Stage {
	if a > b {
		return a
	}
	return b
}

// String returns the stable wire/API name (logs, JSON, admin UI).
func (s Stage) String() string {
	switch s {
	case StageConnect:
		return "connect"
	case StageHelo:
		return "helo"
	case StageMailFrom:
		return "mail_from"
	case StageRcptTo:
		return "rcpt_to"
	case StageHeaders:
		return "headers"
	case StageBody:
		return "body"
	case StageUnknown:
		return "unknown"
	default:
		return fmt.Sprintf("stage(%d)", int(s))
	}
}

// IsValid reports whether s is one of the defined SMTP pipeline stages.
func (s Stage) IsValid() bool {
	return s >= StageConnect && s <= StageBody
}

// ParseStage maps a wire string to Stage (trim, lower). Unknown input yields StageUnknown.
func ParseStage(s string) Stage {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "connect":
		return StageConnect
	case "helo", "ehlo":
		return StageHelo
	case "mail_from", "mailfrom":
		return StageMailFrom
	case "rcpt_to", "rcpt":
		return StageRcptTo
	case "headers", "header":
		return StageHeaders
	case "body", "data":
		return StageBody
	default:
		return StageUnknown
	}
}

// MilterContext holds per-message filter state across milter stages (distinct from Go's context.Context).
type MilterContext struct {
	mu sync.Mutex

	Stage Stage

	MailFrom  string
	Rcpts     []string
	Headers   textproto.MIMEHeader
	BodyBytes []byte

	TraceID         string
	QueueID         string
	Sender          string
	SenderName      string
	Subject         string
	TextBody        string
	HTMLBody        string
	URLList         []string
	AttachmentNames []string
	Attachments     []Attachment

	RDNSNames []string

	ConnectHost   string
	ConnectFamily string
	ConnectPort   uint16
	ConnectIP     net.IP
	HeloName      string

	StartedAt time.Time

	Features FeatureBatch

	LastAction string
	LastRuleID int64

	ModelOutputs []ModelOutput
}

// ModelOutput is one ML classification result merged into features.
type ModelOutput struct {
	ModelKey         string
	Name             string
	Label            string
	Prob             float64
	ProbClass1       float64
	Err              string
	MultiLabelScores map[string]float64
}

func NewMilterContext() *MilterContext {
	return &MilterContext{
		StartedAt: time.Now(),
		Features:  make(FeatureBatch),
	}
}

func (fc *MilterContext) Set(key string, val float64) {
	fc.mu.Lock()
	fc.Features[key] = val
	fc.mu.Unlock()
}

func (fc *MilterContext) Merge(m map[string]float64) {
	if len(m) == 0 {
		return
	}
	fc.mu.Lock()
	for k, v := range m {
		fc.Features[k] = v
	}
	fc.mu.Unlock()
}

func (fc *MilterContext) Snapshot() map[string]float64 {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	out := make(map[string]float64, len(fc.Features))
	for k, v := range fc.Features {
		out[k] = v
	}
	return out
}

func (fc *MilterContext) AppendModelOutput(o ModelOutput) {
	fc.mu.Lock()
	fc.ModelOutputs = append(fc.ModelOutputs, o)
	fc.mu.Unlock()
}
