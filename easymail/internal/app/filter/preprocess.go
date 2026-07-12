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
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/infrastructure/filter/extractors"
	html2text "easymail/internal/pkg/html2text"

	enmime "github.com/jhillyerd/enmime/v2"
)

// ApplyBody parses fc.Headers + fc.BodyBytes with enmime, then fills TextBody, HTMLBody,
// URLList, AttachmentNames, and Attachments on fc. Uses feature.Html2Text when only HTML exists.
// Errors are swallowed (best-effort); returns nil so the milter pipeline is not blocked.
func ApplyBody(fc *filter.MilterContext) error {
	if fc == nil {
		return nil
	}
	raw := extractors.BuildRawRFC822(fc.Headers, fc.BodyBytes)
	if len(raw) == 0 {
		clearBodyFields(fc)
		return nil
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		clearBodyFields(fc)
		fc.URLList = extractors.ExtractURLsFromBody(fc.BodyBytes)
		fc.AttachmentNames = extractors.ExtractAttachmentNamesFromRFC822(fc.BodyBytes)
		return nil
	}

	fc.HTMLBody = env.HTML

	// parse the text body
	rowText := strings.TrimSpace(env.Text)
	if fc.HTMLBody == "" {
		fc.TextBody = rowText
	} else {
		h2t := html2text.NewHtml2Text(nil)
		textParts, linkURLs := h2t.Parse(fc.HTMLBody)
		var b strings.Builder
		for _, p := range textParts {
			p = strings.TrimSpace(p)
			if p != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(p)
			}
		}
		fc.TextBody = strings.TrimSpace(b.String())
		fc.URLList = mergeUniqueURLs(linkURLs, extractors.ExtractURLsFromBody([]byte(fc.TextBody)))
	}

	names := make([]string, 0, len(env.Attachments))
	// Convert enmime attachments to domain Attachment value objects.
	parts := make([]filter.Attachment, 0, len(env.Attachments))
	for _, a := range env.Attachments {
		if a == nil {
			continue
		}
		if n := strings.TrimSpace(a.FileName); n != "" {
			names = append(names, n)
		}
		parts = append(parts, filter.Attachment{FileName: a.FileName, ContentType: a.ContentType, Content: a.Content})
	}
	fc.AttachmentNames = names
	fc.Attachments = parts
	return nil
}

func clearBodyFields(fc *filter.MilterContext) {
	fc.TextBody = ""
	fc.HTMLBody = ""
	fc.URLList = nil
	fc.AttachmentNames = nil
	fc.Attachments = nil
}

func mergeUniqueURLs(base []string, extra []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(base)+len(extra))
	for _, u := range base {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	for _, u := range extra {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
