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

package tracer

import (
	"context"
	"time"
)

type Protocol string

const (
	ProtocolLMTP   Protocol = "lmtp"
	ProtocolMilter Protocol = "milter"
	ProtocolPolicy Protocol = "policy"
	ProtocolAuth   Protocol = "dovecot_auth"
)

type Event struct {
	Timestamp time.Time         `json:"ts"`
	SessionID string            `json:"sessionId"`
	Protocol  Protocol          `json:"protocol"`
	Stage     string            `json:"stage"`
	Remote    string            `json:"remote,omitempty"`
	Local     string            `json:"local,omitempty"`
	Duration  time.Duration     `json:"duration,omitempty"`
	Err       string            `json:"err,omitempty"`
	Fields    map[string]any    `json:"fields,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
}

type SessionMeta struct {
	Protocol Protocol
	Remote   string
	Local    string
	Tags     map[string]string
}

type Sink interface {
	Write(ctx context.Context, e Event) error
	Close() error
}

type Tracer interface {
	NewSession(ctx context.Context, meta SessionMeta) Session
	Close() error
}

type Session interface {
	ID() string
	Event(stage string, fields map[string]any)
	Error(stage string, err error, fields map[string]any)
	End(stage string, fields map[string]any)
}
