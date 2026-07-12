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

package filejsonl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"easymail/pkg/tracer"
)

type Config struct {
	Path        string
	FlushEvery  time.Duration
	RotateDaily bool
}

type Sink struct {
	cfg Config

	mu     sync.Mutex
	file   *os.File
	writer *bufio.Writer
	dayKey string
}

func New(cfg Config) (*Sink, error) {
	if cfg.Path == "" {
		cfg.Path = "session_trace.jsonl"
	}
	if cfg.FlushEvery <= 0 {
		cfg.FlushEvery = 500 * time.Millisecond
	}
	s := &Sink{cfg: cfg}
	if err := s.openLocked(time.Now()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Sink) Write(ctx context.Context, e tracer.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if s.cfg.RotateDaily {
		key := now.Format("20060102")
		if key != s.dayKey {
			_ = s.closeLocked()
			if err := s.openLocked(now); err != nil {
				return err
			}
		}
	}

	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := s.writer.Write(b); err != nil {
		return err
	}
	if err := s.writer.WriteByte('\n'); err != nil {
		return err
	}
	return s.writer.Flush()
}

func (s *Sink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeLocked()
}

func (s *Sink) openLocked(now time.Time) error {
	path := s.cfg.Path
	if s.cfg.RotateDaily {
		ext := filepath.Ext(path)
		base := path[:len(path)-len(ext)]
		path = fmt.Sprintf("%s_%s%s", base, now.Format("20060102"), ext)
		s.dayKey = now.Format("20060102")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	s.file = fh
	s.writer = bufio.NewWriterSize(fh, 64*1024)
	return nil
}

func (s *Sink) closeLocked() error {
	if s.writer != nil {
		_ = s.writer.Flush()
	}
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		s.writer = nil
		return err
	}
	return nil
}
