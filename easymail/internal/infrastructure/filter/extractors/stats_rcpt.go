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
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	redisstats "easymail/internal/infrastructure/filter/stats/redis"
)

// rcptStatsExtractor records sender rcpt frequency and exposes per-message rcpt count.
type rcptStatsExtractor struct{}

func (rcptStatsExtractor) Key() string         { return "stats_rcpt" }
func (rcptStatsExtractor) Stage() filter.Stage { return filter.StageRcptTo }
func (rcptStatsExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil {
		return nil, nil
	}
	sender := strings.ToLower(strings.TrimSpace(fc.MailFrom))
	if sender != "" {
		_, _ = redisstats.Incr(ctx, redisstats.SenderRcptToKey(redisstats.Window1m, sender), redisstats.Window1m.TTL)
		_, _ = redisstats.Incr(ctx, redisstats.SenderRcptToKey(redisstats.Window5m, sender), redisstats.Window5m.TTL)
	}

	// We always expose current message rcpt count for rules.
	rcptCount := 0
	if fc.Rcpts != nil {
		rcptCount = len(fc.Rcpts)
	}
	return filter.FeatureBatch{
		"rcpt_count":          float64(rcptCount),
		"sender_rcpt_count":   float64(rcptCount),
		"sender_has_mailfrom": boolToFloat(sender != ""),
	}, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func init() {
	rule.Register(rcptStatsExtractor{})
}
