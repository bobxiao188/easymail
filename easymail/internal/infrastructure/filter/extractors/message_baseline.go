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

package extractors

import (
	"bytes"
	"context"
	"fmt"
	"net/textproto"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/pkg/rfc2047"

	enmime "github.com/jhillyerd/enmime/v2"
)

// messageBaselineExtractor runs baseline structural features at Body stage (single enmime.ReadEnvelope).
type messageBaselineExtractor struct{}

func (messageBaselineExtractor) Key() string         { return "message_baseline" }
func (messageBaselineExtractor) Stage() filter.Stage { return filter.StageBody }

func (messageBaselineExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil {
		return nil, nil
	}
	_ = ctx

	out := make(map[string]float64)
	body := fc.BodyBytes
	headers := fc.Headers
	out["body_bytes"] = float64(len(body))
	out["rcpt_count"] = float64(len(fc.Rcpts))

	raw := BuildRawRFC822(headers, body)
	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		subj := headerGetTrim(headers, "Subject")
		subj = rfc2047.DecodeHeader(subj)
		out["subject_len"] = float64(len([]rune(subj)))
		if strings.TrimSpace(headerGetTrim(headers, "List-Unsubscribe")) != "" {
			out["has_list_unsubscribe"] = 1
		} else {
			out["has_list_unsubscribe"] = 0
		}
		out["mime_part_count"] = 0
		out["attachment_count"] = 0
		return out, nil
	}

	subj := strings.TrimSpace(env.GetHeader("Subject"))
	subj = rfc2047.DecodeHeader(subj)
	out["subject_len"] = float64(len([]rune(subj)))
	if strings.TrimSpace(env.GetHeader("List-Unsubscribe")) != "" {
		out["has_list_unsubscribe"] = 1
	} else {
		out["has_list_unsubscribe"] = 0
	}
	out["mime_part_count"] = float64(countMIMEParts(env.Root))
	out["attachment_count"] = float64(len(env.Attachments))
	return out, nil
}

func init() {
	rule.Register(messageBaselineExtractor{})
}

// ExtractMessageBaseline returns the same feature map as messageBaselineExtractor.Run using a minimal MilterContext.
func ExtractMessageBaseline(headers textproto.MIMEHeader, body []byte, rcptCount int) map[string]float64 {
	fc := filter.NewMilterContext()
	fc.Headers = headers
	fc.BodyBytes = body
	fc.Rcpts = make([]string, rcptCount)
	var e messageBaselineExtractor
	b, _ := e.Run(context.Background(), fc)
	return map[string]float64(b)
}

// BuildRawRFC822 concatenates MIME headers (CRLF-terminated block) and raw body for parsers such as enmime.ReadEnvelope.
func BuildRawRFC822(headers textproto.MIMEHeader, body []byte) []byte {
	return append(formatHeadersCRLF(headers), body...)
}

func headerGetTrim(h textproto.MIMEHeader, key string) string {
	if h == nil {
		return ""
	}
	return strings.TrimSpace(h.Get(key))
}

func formatHeadersCRLF(h textproto.MIMEHeader) []byte {
	if h == nil {
		return []byte("\r\n")
	}
	var b strings.Builder
	for k, vals := range h {
		can := textproto.CanonicalMIMEHeaderKey(k)
		for _, v := range vals {
			b.WriteString(can)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteString("\r\n")
		}
	}
	b.WriteString("\r\n")
	return []byte(b.String())
}

func countMIMEParts(p *enmime.Part) int {
	if p == nil {
		return 0
	}
	n := 1
	for c := p.FirstChild; c != nil; c = c.NextSibling {
		n += countMIMEParts(c)
	}
	return n
}

// SnapshotString converts features to a readable stable-ish string.
func SnapshotString(f map[string]float64) string {
	if len(f) == 0 {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{")
	first := true
	for k, v := range f {
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString(fmt.Sprintf("%q:%g", k, v))
	}
	b.WriteString("}")
	return b.String()
}
