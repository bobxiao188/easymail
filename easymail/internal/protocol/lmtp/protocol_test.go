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

package lmtp

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

// readLineLimited reads a single line from the reader, ensuring it does not exceed max bytes.
// It returns the line content (without delimiter) and an error if the line is too long or read fails.
func readLineLimited(r *bufio.Reader, max int) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != bufio.ErrBufferFull {
		// Allow EOF if we have some data, otherwise return error
		if len(line) == 0 {
			return "", err
		}
	}

	// Remove trailing \r\n or \n
	line = strings.TrimRight(line, "\r\n")

	// Check length limit (excluding delimiter)
	if len(line) > max {
		return "", errors.New("line too long")
	}

	return line, nil
}

func TestExtractAngleAddr(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"angle bracket", "<user@example.com>", "user@example.com"},
		{"no angle", "user@example.com", "user@example.com"},
		// Outer and inner FWS around the path is stripped (canonical mailbox path).
		{"with spaces", "  < user@test.com >  ", "user@test.com"},
		{"empty brackets", "<>", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAngleAddr(tt.s)
			if got != tt.want {
				t.Errorf("extractAngleAddr(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseMailFrom_Extra(t *testing.T) {
	tests := []struct {
		name string
		line string
		addr string
		ok   bool
	}{
		{"valid", "MAIL FROM:<user@test.com>", "user@test.com", true},
		{"lowercase", "mail from:<user@test.com>", "user@test.com", true},
		{"with spaces", "MAIL FROM: <user@test.com>", "user@test.com", true},
		{"no colon", "MAIL FROM user@test.com", "", false},
		{"wrong cmd", "RCPT TO:<user@test.com>", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, ok := ParseMailFrom(tt.line)
			if ok != tt.ok || addr != tt.addr {
				t.Errorf("ParseMailFrom(%q) = (%q,%v), want (%q,%v)",
					tt.line, addr, ok, tt.addr, tt.ok)
			}
		})
	}
}

func TestParseRcptTo_Extra(t *testing.T) {
	tests := []struct {
		name string
		line string
		addr string
		ok   bool
	}{
		{"valid", "RCPT TO:<user@test.com>", "user@test.com", true},
		{"lowercase", "rcpt to:<user@test.com>", "user@test.com", true},
		{"no colon", "RCPT TO user", "", false},
		{"wrong cmd", "MAIL FROM:<user@test.com>", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, ok := ParseRcptTo(tt.line)
			if ok != tt.ok || addr != tt.addr {
				t.Errorf("ParseRcptTo(%q) = (%q,%v), want (%q,%v)",
					tt.line, addr, ok, tt.addr, tt.ok)
			}
		})
	}
}

func TestReadLineLimited(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		max     int
		want    string
		wantErr bool
	}{
		{"normal", "hello\n", 100, "hello", false},
		{"with CRLF", "hello\r\n", 100, "hello", false},
		{"exceeds limit", "hello world\n", 5, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadLineLimited(r, tt.max)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readLineLimited() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("readLineLimited() = %q, want %q", got, tt.want)
			}
		})
	}
}
