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
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	redisfilter "easymail/internal/infrastructure/filter/stats/redis"
)

// connectStatsExtractor records and reads basic IP rate features.
type connectStatsExtractor struct{}

func (connectStatsExtractor) Key() string         { return "stats_connect" }
func (connectStatsExtractor) Stage() filter.Stage { return filter.StageConnect }
func (connectStatsExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil || fc.ConnectIP == nil {
		return nil, nil
	}
	ip := fc.ConnectIP.String()
	now := time.Now()
	day := redisfilter.LocalDateYYYYMMDD(now)

	// Best-effort: increment connection counters. If Redis is down, just skip features.
	_, _ = redisfilter.Incr(ctx, redisfilter.IPConnectDayKey(day, ip), redisfilter.FilterDayCounterTTL)
	_, _ = redisfilter.Incr(ctx, redisfilter.IPConnectKey(redisfilter.Window5m, ip), redisfilter.Window5m.TTL)

	nDay, errDay := redisfilter.GetInt(ctx, redisfilter.IPConnectDayKey(day, ip))
	n5, err5 := redisfilter.GetInt(ctx, redisfilter.IPConnectKey(redisfilter.Window5m, ip))
	if errDay != nil && err5 != nil {
		return nil, nil
	}

	// outcome counters (best-effort read): _1d local calendar day; _5m sliding window
	aDay, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeDayKey(day, ip, filter.OutcomeAccept))
	a5, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeKey(redisfilter.Window5m, ip, filter.OutcomeAccept))
	rDay, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeDayKey(day, ip, filter.OutcomeReject))
	r5, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeKey(redisfilter.Window5m, ip, filter.OutcomeReject))
	sDay, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeDayKey(day, ip, filter.OutcomeSpam))
	s5, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeKey(redisfilter.Window5m, ip, filter.OutcomeSpam))
	qDay, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeDayKey(day, ip, filter.OutcomeQuarantine))
	q5, _ := redisfilter.GetInt(ctx, redisfilter.IPOutcomeKey(redisfilter.Window5m, ip, filter.OutcomeQuarantine))
	tDay := aDay + rDay + sDay + qDay
	t5 := a5 + r5 + s5 + q5
	failRateDay := 0.0
	if tDay > 0 {
		failRateDay = float64(rDay) / float64(tDay)
	}
	failRate5 := 0.0
	if t5 > 0 {
		failRate5 = float64(r5) / float64(t5)
	}
	secDay := redisfilter.SecondsSinceLocalMidnight(now)
	out := filter.FeatureBatch{
		"ip_connect_count_1d":      float64(nDay),
		"ip_connect_count_5m":      float64(n5),
		"ip_connect_rate_1d":       float64(nDay) / float64(secDay),
		"ip_connect_rate_5m":       float64(n5) / float64((5 * time.Minute).Seconds()),
		"ip_outcome_accept_1d":     float64(aDay),
		"ip_outcome_reject_1d":     float64(rDay),
		"ip_outcome_spam_1d":       float64(sDay),
		"ip_outcome_quarantine_1d": float64(qDay),
		"ip_outcome_total_1d":      float64(tDay),
		"ip_reject_rate_1d":        failRateDay,
		"ip_reject_rate_5m":        failRate5,
	}
	return out, nil
}

func init() {
	rule.Register(connectStatsExtractor{})
}
