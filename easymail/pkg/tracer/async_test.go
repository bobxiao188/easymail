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
	"sync/atomic"
	"testing"
	"time"
)

type countingSink struct {
	n atomic.Int64
}

func (s *countingSink) Write(ctx context.Context, e Event) error {
	s.n.Add(1)
	return nil
}
func (s *countingSink) Close() error { return nil }

func TestAsyncTracerWritesEvents(t *testing.T) {
	sink := &countingSink{}
	tr := NewAsync(Config{
		Enabled:   true,
		QueueSize: 16,
	}, sink)
	defer tr.Close()

	sess := tr.NewSession(context.Background(), SessionMeta{Protocol: ProtocolLMTP, Remote: "r", Local: "l"})
	sess.Event("a", map[string]any{"k": "v"})
	sess.End("end", nil)

	// best-effort: allow worker to drain
	time.Sleep(50 * time.Millisecond)
	if sink.n.Load() == 0 {
		t.Fatalf("expected events written")
	}
}

func TestAsyncSessionEventLimit(t *testing.T) {
	sink := &countingSink{}
	tr := NewAsync(Config{
		Enabled:          true,
		QueueSize:        128,
		MaxEventsPerSess: 3,
	}, sink)
	defer tr.Close()

	sess := tr.NewSession(context.Background(), SessionMeta{Protocol: ProtocolLMTP})
	for i := 0; i < 10; i++ {
		sess.Event("x", nil)
	}
	sess.End("end", nil)
	time.Sleep(50 * time.Millisecond)
	if sink.n.Load() > 10 {
		t.Fatalf("unexpected count: %d", sink.n.Load())
	}
}
