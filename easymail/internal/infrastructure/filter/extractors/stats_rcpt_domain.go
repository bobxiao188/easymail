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

// rcptDomainStatsExtractor tracks and exposes sender->rcpt_domain distribution.
type rcptDomainStatsExtractor struct{}

func (rcptDomainStatsExtractor) Key() string         { return "stats_rcpt_domain" }
func (rcptDomainStatsExtractor) Stage() filter.Stage { return filter.StageRcptTo }
func (rcptDomainStatsExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil {
		return nil, nil
	}
	sender := strings.ToLower(strings.TrimSpace(fc.MailFrom))
	if sender == "" || len(fc.Rcpts) == 0 {
		return nil, nil
	}
	rcpt := strings.ToLower(strings.TrimSpace(fc.Rcpts[len(fc.Rcpts)-1]))
	i := strings.LastIndex(rcpt, "@")
	if i < 0 || i >= len(rcpt)-1 {
		return filter.FeatureBatch{"rcpt_domain_present": 0}, nil
	}
	dom := strings.TrimSpace(rcpt[i+1:])
	if dom == "" {
		return filter.FeatureBatch{"rcpt_domain_present": 0}, nil
	}

	now := time.Now()
	day := redisstats.LocalDateYYYYMMDD(now)
	_, _ = redisstats.Incr(ctx, redisstats.SenderRcptDomainDayKey(day, sender, dom), redisstats.FilterDayCounterTTL)
	_, _ = redisstats.Incr(ctx, redisstats.SenderRcptDomainKey(redisstats.Window5m, sender, dom), redisstats.Window5m.TTL)

	cDay, _ := redisstats.GetInt(ctx, redisstats.SenderRcptDomainDayKey(day, sender, dom))
	c5, _ := redisstats.GetInt(ctx, redisstats.SenderRcptDomainKey(redisstats.Window5m, sender, dom))

	// outcome distribution for this (sender, rcpt_domain): local calendar day
	aDay, _ := redisstats.GetInt(ctx, redisstats.SenderRcptDomainOutcomeDayKey(day, sender, dom, filter.OutcomeAccept))
	rDay, _ := redisstats.GetInt(ctx, redisstats.SenderRcptDomainOutcomeDayKey(day, sender, dom, filter.OutcomeReject))
	sDay, _ := redisstats.GetInt(ctx, redisstats.SenderRcptDomainOutcomeDayKey(day, sender, dom, filter.OutcomeSpam))
	qDay, _ := redisstats.GetInt(ctx, redisstats.SenderRcptDomainOutcomeDayKey(day, sender, dom, filter.OutcomeQuarantine))
	tDay := aDay + rDay + sDay + qDay
	rejectRateDay := 0.0
	if tDay > 0 {
		rejectRateDay = float64(rDay) / float64(tDay)
	}

	return filter.FeatureBatch{
		"rcpt_domain_present":               1,
		"sender_rcpt_domain_count_1d":       float64(cDay),
		"sender_rcpt_domain_count_5m":       float64(c5),
		"sender_rcpt_domain_accept_1d":      float64(aDay),
		"sender_rcpt_domain_reject_1d":      float64(rDay),
		"sender_rcpt_domain_spam_1d":        float64(sDay),
		"sender_rcpt_domain_quarantine_1d":  float64(qDay),
		"sender_rcpt_domain_total_1d":       float64(tDay),
		"sender_rcpt_domain_reject_rate_1d": rejectRateDay,
	}, nil
}

func init() {
	rule.Register(rcptDomainStatsExtractor{})
}
