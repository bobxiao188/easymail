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
	"bytes"
	"io"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"

	"easymail/pkg/constants"
	"easymail/pkg/config"
)

// headerSectionBytes returns the RFC822 header block (before first blank line) for loose parsing when ReadMessage fails.
func headerSectionBytes(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	if i := bytes.Index(raw, []byte("\r\n\r\n")); i >= 0 {
		return raw[:i]
	}
	if i := bytes.Index(raw, []byte("\n\n")); i >= 0 {
		return raw[:i]
	}
	return raw
}

// parseHeadersLoose parses headers and properly unfolds continuation lines (RFC 5322 folding).
// Used as fallback for LF-only bodies or when net/mail.ReadMessage fails.
func parseHeadersLoose(section []byte) textproto.MIMEHeader {
	h := make(textproto.MIMEHeader)
	var curKey string
	var curVal strings.Builder
	flush := func() {
		if curKey != "" {
			h.Add(curKey, strings.TrimSpace(curVal.String()))
			curKey = ""
			curVal.Reset()
		}
	}
	for _, line := range bytes.Split(section, []byte{'\n'}) {
		line = bytes.TrimRight(line, "\r")
		if len(line) == 0 {
			flush()
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			// Continuation line: unfold by appending trimmed content with a single space separator.
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) > 0 {
				if curVal.Len() > 0 {
					curVal.WriteByte(' ')
				}
				curVal.Write(trimmed)
			}
			continue
		}
		// New header line
		flush()
		i := bytes.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		curKey = string(bytes.TrimSpace(line[:i]))
		v := string(bytes.TrimSpace(line[i+1:]))
		curVal.WriteString(v)
	}
	flush()
	return h
}

// LMTPRouteOptions selects system folder kind from MIME headers on LMTP delivery.
type LMTPRouteOptions struct {
	Config config.FilterConfig
}

// FolderKindForInbound chooses delivery folder kind from raw RFC822 bytes and config (default inbox).
func FolderKindForInbound(body []byte, o *LMTPRouteOptions) constants.FolderID {
	if o == nil {
		return constants.Inbox
	}
	if len(body) == 0 {
		return constants.Inbox
	}
	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err == nil {
		return folderKindFromHeaders(textproto.MIMEHeader(msg.Header), o)
	}
	// If strict parse fails, still parse header section so milter-written X-Easymail-* headers are not lost (e.g. LF-only, bad first line).
	sect := headerSectionBytes(body)
	if len(sect) == 0 {
		return constants.Inbox
	}
	return folderKindFromHeaders(parseHeadersLoose(sect), o)
}

func folderKindFromHeaders(h textproto.MIMEHeader, o *LMTPRouteOptions) constants.FolderID {
	def := defaultKind(o)
	if o != nil && o.Config.SkipForComposeDelivery {
		if strings.TrimSpace(h.Get(HeaderInternalCompose)) == "1" {
			return constants.Inbox
		}
	}
	act := strings.ToLower(strings.TrimSpace(h.Get(HeaderFilterAction)))
	switch act {
	case "spam":
		return constants.Spam
	case "quarantine":
		return constants.Quarantine
	case "reject":
		// Reject should be enforced in milter; if mail still reaches LMTP, fall back to default routing.
		return def
	case "accept", "":
		if act == "" {
			return def
		}
		return constants.Inbox
	default:
		return def
	}
}

func defaultKind(o *LMTPRouteOptions) constants.FolderID {
	if o == nil {
		return constants.Inbox
	}
	switch strings.ToLower(strings.TrimSpace(o.Config.DefaultAction)) {
	case "spam":
		return constants.Spam
	case "quarantine":
		return constants.Quarantine
	case "reject":
		// Default reject must not drop LMTP acceptance; LMTP only stores and routes.
		return constants.Quarantine
	default:
		return constants.Inbox
	}
}

// ParseFilterHeadersFromBody is for debugging; returns action, ruleID, traceID (empty if absent).
func ParseFilterHeadersFromBody(body []byte) (action, ruleID, traceID string) {
	msg, err := mail.ReadMessage(bytes.NewReader(body))
	var h textproto.MIMEHeader
	if err == nil {
		h = textproto.MIMEHeader(msg.Header)
	} else {
		sect := headerSectionBytes(body)
		if len(sect) == 0 {
			return "", "", ""
		}
		h = parseHeadersLoose(sect)
	}
	action = strings.TrimSpace(h.Get(HeaderFilterAction))
	ruleID = strings.TrimSpace(h.Get(HeaderFilterRuleID))
	traceID = strings.TrimSpace(h.Get(HeaderFilterTraceID))
	return action, ruleID, traceID
}

// ReadMessageBody returns body bytes after headers from raw message (best effort).
func ReadMessageBody(raw []byte) []byte {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	b, _ := io.ReadAll(msg.Body)
	return b
}

// ParseRuleID parses X-Easymail-Filter-Rule-Id; 0 if empty/invalid.
func ParseRuleID(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
