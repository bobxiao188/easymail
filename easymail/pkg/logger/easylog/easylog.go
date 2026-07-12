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

package easylog

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger 灏佽 logrus锛屾敮鎸佹寜妯″潡瀛楁锛坢odule锛変笌鐙珛鏂囦欢杈撳嚭
type Logger struct {
	l *logrus.Logger
	e *logrus.Entry
}

func (e *Logger) out() *logrus.Entry {
	if e == nil {
		return nil
	}
	if e.e != nil {
		return e.e
	}
	if e.l != nil {
		return logrus.NewEntry(e.l)
	}
	return nil
}

// New 鍒涘缓鏃ュ織锛沠ilePath 闈炵┖鏃跺悓鏃跺啓鍏ヨ鏂囦欢stderr
func New(filePath string) (*Logger, error) {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	l.SetLevel(logrus.InfoLevel)
	var writers []io.Writer
	writers = append(writers, os.Stderr)
	if filePath != "" {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		writers = append(writers, f)
	}
	l.SetOutput(io.MultiWriter(writers...))
	return &Logger{l: l}, nil
}

// WithModule adds a module field to every log entry.
func (e *Logger) WithModule(module string) *Logger {
	if e == nil || e.l == nil {
		return e
	}
	name := strings.TrimSpace(module)
	if name == "" {
		return e
	}
	var ent *logrus.Entry
	if e.e != nil {
		ent = e.e.WithField("module", name)
	} else {
		ent = e.l.WithField("module", name)
	}
	return &Logger{l: e.l, e: ent}
}

// SetLevelString sets log level from config.
func (e *Logger) SetLevelString(level string) *Logger {
	if e == nil || e.l == nil {
		return e
	}
	e.l.SetLevel(parseLevel(level))
	return e
}

func parseLevel(level string) logrus.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return logrus.DebugLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "trace":
		return logrus.TraceLevel
	default:
		return logrus.InfoLevel
	}
}

func (e *Logger) Debugf(format string, args ...interface{}) {
	if o := e.out(); o != nil {
		o.Debugf(format, args...)
	}
}

func (e *Logger) Infof(format string, args ...interface{}) {
	if o := e.out(); o != nil {
		o.Infof(format, args...)
	}
}

func (e *Logger) Info(args ...interface{}) {
	if o := e.out(); o != nil {
		o.Info(args...)
	}
}

func (e *Logger) Error(args ...interface{}) {
	if o := e.out(); o != nil {
		o.Error(args...)
	}
}

func (e *Logger) Errorf(format string, args ...interface{}) {
	if o := e.out(); o != nil {
		o.Errorf(format, args...)
	}
}

func (e *Logger) Warn(args ...interface{}) {
	if o := e.out(); o != nil {
		o.Warn(args...)
	}
}

func (e *Logger) Warnf(format string, args ...interface{}) {
	if o := e.out(); o != nil {
		o.Warnf(format, args...)
	}
}

// NewDiscardLogger returns a Logger that discards all output (no-op).
func NewDiscardLogger() *Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return &Logger{l: l}
}
