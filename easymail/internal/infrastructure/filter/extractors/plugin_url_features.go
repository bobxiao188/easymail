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
	"context"
	"regexp"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
)

type urlFeaturePlugin struct{}

func (urlFeaturePlugin) Key() string { return "url_basic" }

func (urlFeaturePlugin) Stage() filter.Stage { return filter.StageBody }

var urlRe = regexp.MustCompile(`https?://[^\s<>"']+`)

// shortURLRe: URL shape only (no fixed host/TLD strings):
//
//	A) compact host (1-4 labels + letter TLD) + slug that has a digit or ASCII uppercase; or
//	B) very short two-label host + alphanumeric lowercase slug (typical t.xx/xxxxxx short codes).
//
// (?-i) keeps slug case meaningful where needed; (?i) allows scheme/host casing.
var shortURLRe = regexp.MustCompile(`(?i)^https?://[a-z0-9-]{1,4}\.[a-z0-9-]{1,3}/[a-z0-9-]{1,256}`)

func (urlFeaturePlugin) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	_ = ctx
	if fc == nil {
		return nil, nil
	}
	if len(fc.URLList) > 0 {
		return urlBasicAccumulate(fc.URLList), nil
	}
	if len(fc.HTMLBody) == 0 {
		return urlBasicZeros(), nil
	}
	s := fc.HTMLBody
	matches := urlRe.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return urlBasicZeros(), nil
	}
	urls := make([]string, 0, len(matches))
	for _, idx := range matches {
		urls = append(urls, s[idx[0]:idx[1]])
	}
	return urlBasicAccumulate(urls), nil
}

func urlBasicZeros() filter.FeatureBatch {
	return filter.FeatureBatch{
		"body_url_count":       0,
		"body_url_has_http":    0,
		"body_url_has_https":   0,
		"body_url_short_count": 0,
		"body_url_has_short":   0,
	}
}

func urlBasicAccumulate(urls []string) filter.FeatureBatch {
	hasHTTP := 0.0
	hasHTTPS := 0.0
	shortCnt := 0
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "http://") {
			hasHTTP = 1
		}
		if strings.HasPrefix(raw, "https://") {
			hasHTTPS = 1
		}
		if isShortURLByRegexp(raw) {
			shortCnt++
		}
	}
	hasShort := 0.0
	if shortCnt > 0 {
		hasShort = 1
	}
	return filter.FeatureBatch{
		"body_url_count":       float64(len(urls)),
		"body_url_has_http":    hasHTTP,
		"body_url_has_https":   hasHTTPS,
		"body_url_short_count": float64(shortCnt),
		"body_url_has_short":   hasShort,
	}
}

func isShortURLByRegexp(raw string) bool {
	u := strings.TrimRight(strings.TrimSpace(raw), `.,;:!?)*]>'"`)
	return shortURLRe.MatchString(u)
}

func init() {
	rule.RegisterContentPlugin(urlFeaturePlugin{})
}
