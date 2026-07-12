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
	"io"
	"strings"
)

var ErrLineTooLong = errors.New("line too long")

// ReadLineLimited reads one line up to max bytes.
// If the line exceeds max, ErrLineTooLong is returned after draining
// the remainder so the next call starts at a valid position.
func ReadLineLimited(r *bufio.Reader, max int) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	if len(line) > max {
		for err == nil {
			var chunk string
			chunk, err = r.ReadString('\n')
			if strings.HasSuffix(chunk, "\n") {
				break
			}
		}
		return "", ErrLineTooLong
	}

	line = strings.TrimRight(line, "\r\n")
	return line, nil
}

// ParseMailFrom extracts the reverse-path from a MAIL FROM command.
func ParseMailFrom(line string) (addr string, ok bool) {
	line = strings.TrimSpace(line)
	upper := strings.ToUpper(line)
	if !strings.HasPrefix(upper, "MAIL FROM") {
		return "", false
	}
	rest := strings.TrimSpace(line[9:])
	if !strings.HasPrefix(rest, ":") {
		return "", false
	}
	rest = strings.TrimSpace(rest[1:])
	return extractAngleAddr(rest), true
}

// ParseRcptTo extracts the forward-path from a RCPT TO command.
func ParseRcptTo(line string) (addr string, ok bool) {
	line = strings.TrimSpace(line)
	upper := strings.ToUpper(line)
	if !strings.HasPrefix(upper, "RCPT TO") {
		return "", false
	}
	rest := strings.TrimSpace(line[7:])
	if !strings.HasPrefix(rest, ":") {
		return "", false
	}
	rest = strings.TrimSpace(rest[1:])
	return extractAngleAddr(rest), true
}

func extractAngleAddr(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>' {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}
