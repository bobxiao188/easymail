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

package htmlsanitize

import (
	"bytes"
	"html"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	xhtml "golang.org/x/net/html"
)

func SanitizeEmailHTML(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	p := bluemonday.UGCPolicy()
	sanitized := p.Sanitize(s)
	if strings.TrimSpace(sanitized) == "" {
		return "", nil
	}
	return normalizeFragment(sanitized)
}

func PlainTextToSafeHTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return `<p class="wm-plain wm-empty">(鏃犳鏂?</p>`
	}
	escaped := html.EscapeString(s)
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\n")
	parts := strings.Split(escaped, "\n")
	var b strings.Builder
	b.WriteString(`<div class="wm-plain">`)
	for i, line := range parts {
		if i > 0 {
			b.WriteString("<br>")
		}
		b.WriteString(line)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func normalizeFragment(s string) (string, error) {
	nodes, err := xhtml.ParseFragment(strings.NewReader(s), nil)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	for _, n := range nodes {
		if err := xhtml.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(buf.String()), nil
}
