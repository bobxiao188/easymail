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
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// tokenize splits on whitespace; quoted strings and parenthesized lists are single tokens.
func tokenize(line string) ([]string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	var out []string
	for i := 0; i < len(line); {
		if line[i] == ' ' || line[i] == '\t' {
			i++
			continue
		}
		if line[i] == '(' {
			end, err := findBalanced(line, i, '(', ')')
			if err != nil {
				return nil, err
			}
			out = append(out, line[i:end+1])
			i = end + 1
			continue
		}
		if line[i] == '"' {
			j := i + 1
			var b strings.Builder
			for j < len(line) {
				if line[j] == '\\' && j+1 < len(line) {
					b.WriteByte(line[j+1])
					j += 2
					continue
				}
				if line[j] == '"' {
					out = append(out, b.String())
					i = j + 1
					break
				}
				b.WriteByte(line[j])
				j++
			}
			if j >= len(line) {
				return nil, errors.New("unclosed quote")
			}
			continue
		}
		j := i
		for j < len(line) && line[j] != ' ' && line[j] != '\t' {
			j++
		}
		out = append(out, line[i:j])
		i = j
	}
	return out, nil
}

func findBalanced(s string, start int, open, close byte) (int, error) {
	if start >= len(s) || s[start] != open {
		return 0, errors.New("expected open paren")
	}
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == open {
			depth++
		} else if s[i] == close {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, errors.New("unbalanced paren")
}

func parseSeqSet(s string) (SeqSet, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty sequence set")
	}
	var ss SeqSet
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		a, b, err := parseRangeToken(part)
		if err != nil {
			return nil, err
		}
		ss = append(ss, SeqRange{Start: a, Stop: b})
	}
	return ss, nil
}

func parseUIDSet(s string) (UIDSet, error) {
	ss, err := parseSeqSet(s)
	if err != nil {
		return nil, err
	}
	return UIDSet(ss), nil
}

func parseRangeToken(tok string) (uint32, uint32, error) {
	if !strings.Contains(tok, ":") {
		n, err := strconv.ParseUint(tok, 10, 32)
		if err != nil {
			return 0, 0, err
		}
		u := uint32(n)
		return u, u, nil
	}
	parts := strings.SplitN(tok, ":", 2)
	var start, stop uint32
	if parts[0] == "*" {
		start = 0
	} else {
		n, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return 0, 0, err
		}
		start = uint32(n)
	}
	if parts[1] == "*" {
		stop = 0
	} else {
		n, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return 0, 0, err
		}
		stop = uint32(n)
	}
	return start, stop, nil
}

// parseFetchList parses a parenthesized FETCH item list (UID FLAGS ENVELOPE BODY.PEEK[], ...).
func parseFetchList(listStr string) (FetchOptions, error) {
	listStr = strings.TrimSpace(listStr)
	if !strings.HasPrefix(listStr, "(") {
		return FetchOptions{}, fmt.Errorf("fetch list must start with (")
	}
	end, err := findBalanced(listStr, 0, '(', ')')
	if err != nil {
		return FetchOptions{}, err
	}
	inner := strings.TrimSpace(listStr[1:end])
	var opts FetchOptions
	if inner == "" {
		return opts, nil
	}
	// parseFetchItems is bracket-aware: BODY.PEEK[HEADER.FIELDS (From ...)]
	// stays as a single token, no reassembly needed.
	parts := parseFetchItems(inner)

	for _, p := range parts {
		p = strings.TrimSpace(p)
		up := strings.ToUpper(p)
		switch {
		case up == "UID":
			opts.UID = true
		case up == "FLAGS":
			opts.Flags = true
		case up == "ENVELOPE":
			opts.Envelope = true
		case up == "RFC822.SIZE":
			opts.RFC822Size = true
		case up == "INTERNALDATE":
			opts.InternalDate = true
		case strings.HasPrefix(up, "BODY.PEEK"):
			opts.BodyPeek = true
			opts.BodyItem = p
		case strings.HasPrefix(up, "BODY["), strings.HasPrefix(up, "BODY"):
			if strings.Contains(up, "PEEK") {
				opts.BodyPeek = true
			} else {
				opts.BodyNotPeek = true
			}
			opts.BodyItem = p
		case up == "BODY[]", strings.HasPrefix(up, "BODY[]"):
			opts.BodyNotPeek = true
		case up == "RFC822", strings.HasPrefix(up, "RFC822"):
			opts.BodyNotPeek = true
			opts.BodyItem = p
		case strings.HasPrefix(up, "FAST"):
			opts.Flags = true
			opts.InternalDate = true
			opts.RFC822Size = true
		case strings.HasPrefix(up, "ALL"):
			opts.Flags = true
			opts.InternalDate = true
			opts.RFC822Size = true
			opts.Envelope = true
		case strings.HasPrefix(up, "FULL"):
			opts.Flags = true
			opts.InternalDate = true
			opts.RFC822Size = true
			opts.Envelope = true
		}
	}
	return opts, nil
}

// parseFetchItems splits a parenthesized inner string into FETCH data items.
// Brackets [...] are treated as atomic so BODY.PEEK[HEADER.FIELDS (From To ...)]
// stays as a single token regardless of spaces inside brackets.
func parseFetchItems(s string) []string {
	var out []string
	i := 0
	for i < len(s) {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		// Parenthesized list — single token
		if s[i] == '(' {
			end, err := findBalanced(s, i, '(', ')')
			if err != nil {
				out = append(out, strings.TrimSpace(s[i:]))
				break
			}
			out = append(out, s[i:end+1])
			i = end + 1
			continue
		}
		// Bracket-aware: spaces inside [...] are part of the token
		j := i
		depth := 0
		for j < len(s) && (depth > 0 || (s[j] != ' ' && s[j] != '\t')) {
			if s[j] == '[' {
				depth++
			} else if s[j] == ']' {
				depth--
			}
			j++
		}
		out = append(out, s[i:j])
		i = j
	}
	return out
}

// parseStoreItem parses +FLAGS / -FLAGS / FLAGS (including .SILENT).
func parseStoreItem(item string) (StoreOp, bool) {
	silent := strings.HasSuffix(strings.ToUpper(item), ".SILENT")
	base := item
	if silent {
		base = strings.TrimSuffix(item, ".SILENT")
		base = strings.TrimSuffix(base, ".silent")
	}
	u := strings.ToUpper(strings.TrimSpace(base))
	switch {
	case strings.HasPrefix(u, "+FLAGS"):
		return StoreOpAdd, silent
	case strings.HasPrefix(u, "-FLAGS"):
		return StoreOpRemove, silent
	case strings.HasPrefix(u, "FLAGS"):
		return StoreOpReplace, silent
	default:
		return StoreOpReplace, silent
	}
}

// parseFlagList parses a flag list like (\Seen) or (\\Seen).
func parseFlagList(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "(") {
		return nil
	}
	end, err := findBalanced(s, 0, '(', ')')
	if err != nil {
		return nil
	}
	inner := strings.TrimSpace(s[1:end])
	if inner == "" {
		return nil
	}
	var flags []string
	for _, p := range strings.Fields(inner) {
		p = strings.TrimSpace(p)
		if p != "" {
			flags = append(flags, p)
		}
	}
	return flags
}
