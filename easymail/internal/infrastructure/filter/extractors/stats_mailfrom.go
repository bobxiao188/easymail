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
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	redisstats "easymail/internal/infrastructure/filter/stats/redis"
)

// mailFromStatsExtractor records and reads basic sender rate features.
type mailFromStatsExtractor struct{}

func (mailFromStatsExtractor) Key() string         { return "stats_mailfrom" }
func (mailFromStatsExtractor) Stage() filter.Stage { return filter.StageMailFrom }
func (mailFromStatsExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil {
		return nil, nil
	}
	s := strings.ToLower(strings.TrimSpace(fc.MailFrom))
	if s == "" {
		return nil, nil
	}

	now := time.Now()
	day := redisstats.LocalDateYYYYMMDD(now)
	_, _ = redisstats.Incr(ctx, redisstats.SenderMailFromDayKey(day, s), redisstats.FilterDayCounterTTL)
	_, _ = redisstats.Incr(ctx, redisstats.SenderMailFromKey(redisstats.Window5m, s), redisstats.Window5m.TTL)

	nDay, errDay := redisstats.GetInt(ctx, redisstats.SenderMailFromDayKey(day, s))
	n5, err5 := redisstats.GetInt(ctx, redisstats.SenderMailFromKey(redisstats.Window5m, s))
	if errDay != nil && err5 != nil {
		return nil, nil
	}

	// outcome counters (best-effort read): local calendar day
	aDay, _ := redisstats.GetInt(ctx, redisstats.SenderOutcomeDayKey(day, s, filter.OutcomeAccept))
	rDay, _ := redisstats.GetInt(ctx, redisstats.SenderOutcomeDayKey(day, s, filter.OutcomeReject))
	spDay, _ := redisstats.GetInt(ctx, redisstats.SenderOutcomeDayKey(day, s, filter.OutcomeSpam))
	qDay, _ := redisstats.GetInt(ctx, redisstats.SenderOutcomeDayKey(day, s, filter.OutcomeQuarantine))
	tDay := aDay + rDay + spDay + qDay
	rejectRateDay := 0.0
	if tDay > 0 {
		rejectRateDay = float64(rDay) / float64(tDay)
	}

	return filter.FeatureBatch{
		"sender_mailfrom_count_1d":     float64(nDay),
		"sender_mailfrom_count_5m":     float64(n5),
		"sender_outcome_accept_1d":     float64(aDay),
		"sender_outcome_reject_1d":     float64(rDay),
		"sender_outcome_spam_1d":       float64(spDay),
		"sender_outcome_quarantine_1d": float64(qDay),
		"sender_outcome_total_1d":      float64(tDay),
		"sender_reject_rate_1d":        rejectRateDay,
	}, nil
}

func init() {
	rule.Register(mailFromStatsExtractor{})
}
