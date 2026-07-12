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

package milter

import (
	"context"
	"strings"
	"time"

	service "easymail/internal/app/filter"
	"easymail/internal/domain/filter"
	featstats "easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	redisstore "easymail/internal/infrastructure/filter/persistence/redis"
	redisstats "easymail/internal/infrastructure/filter/stats/redis"
	"easymail/internal/infrastructure/persistence/mysql"
	"easymail/internal/protocol/milter"
)

func (h *MilterHandler) finishMilterStage(ctx context.Context, st filter.Stage, extractTimeout time.Duration) (milter.Response, error) {
	// Apply stage timeout to the entire processing (Extract + evalAndTrack)
	// so that a cold cache or slow DB cannot block the milter response indefinitely.
	ctx, cancel := context.WithTimeout(ctx, extractTimeout)
	defer cancel()

	t0 := time.Now()
	var res *service.MatchResult
	var evalErr error
	defer func() {
		h.asyncStageFilterLog(coalesceStageMatchResult(res), time.Since(t0))
		h.infof("milter_trace stage=%s total elapsed_ms=%d", st, time.Since(t0).Milliseconds())
	}()
	h.infof("milter_trace stage=%s extract_start timeout_ms=%d", st, extractTimeout.Milliseconds())
	h.fe.Extract(ctx, st, h.fc, extractTimeout)
	h.infof("milter_trace stage=%s extract_end elapsed_ms=%d", st, time.Since(t0).Milliseconds())
	res, evalErr = h.evalAndTrack(ctx, st)
	h.infof("milter_trace stage=%s eval_end elapsed_ms=%d", st, time.Since(t0).Milliseconds())
	return h.milterStageResponse(ctx, res, evalErr)
}

func (h *MilterHandler) evalAndTrack(ctx context.Context, st filter.Stage) (*service.MatchResult, error) {
	if h.eng == nil || h.fc == nil {
		return &service.MatchResult{Action: "accept", RuleID: 0, FeatureSnap: "{}", TraceJSON: "{}"}, nil
	}
	feat := h.fc.Snapshot()
	snap := service.SnapshotString(feat)
	res, err := h.eng.EvaluateWithFeaturesAtStage(ctx, st, feat, snap)
	if err != nil || res == nil {
		return &service.MatchResult{Action: "accept", RuleID: 0, FeatureSnap: snap, TraceJSON: "{}"}, err
	}
	h.fc.LastAction = res.Action
	h.fc.LastRuleID = res.RuleID
	return res, nil
}

func (h *MilterHandler) milterStageResponse(ctx context.Context, res *service.MatchResult, err error) (milter.Response, error) {
	if err != nil {
		return milter.RespContinue, nil
	}
	if res == nil {
		return milter.RespContinue, nil
	}
	if strings.ToLower(strings.TrimSpace(res.Action)) == "reject" {
		o := featstats.NormalizeOutcome(res.Action)
		if h.fc != nil && h.fc.ConnectIP != nil {
			redisstats.RecordIPOutcome(ctx, h.fc.ConnectIP.String(), o)
		}
		if h.fc != nil {
			redisstats.RecordSenderOutcome(ctx, h.fc.MailFrom, o)
		}
		return milter.RespReject, nil
	}
	return milter.RespContinue, nil
}

// stripInboundEasymailPolicyHeaders removes policy headers from the MTA queue file (SMFIF_CHGHDRS),
// including legacy spellings, so clients cannot forge filter decisions before our Body-stage headers.
func (h *MilterHandler) stripInboundEasymailPolicyHeaders(m *milter.Modifier) {
	if m == nil {
		return
	}
	names := []string{
		service.HeaderFilterAction,
		service.HeaderFilterRuleID,
		service.HeaderFilterTraceID,
		"X-EasyMail-Filter-Action",
		"X-EasyMail-Filter-Rule-Id",
		"X-EasyMail-Filter-Trace-Id",
	}
	for _, n := range names {
		if err := m.RemoveHeader(n); err != nil {
			h.warnf("milter RemoveHeader %q: %v", n, err)
		}
	}
}

// asyncStageFilterLog persists one row per milter stage (best-effort). Session fields come from h.fc; only the match result and timing are passed.
func (h *MilterHandler) asyncStageFilterLog(res *service.MatchResult, d time.Duration) {
	if h.db == nil || h.eng == nil || h.fc == nil || res == nil {
		return
	}
	st := h.fc.Stage
	recipient := ""
	if len(h.fc.Rcpts) > 0 {
		recipient = h.fc.Rcpts[0]
	}
	subject := strings.TrimSpace(h.fc.Subject)
	if subject == "" && h.fc.Headers != nil {
		subject = h.fc.Headers.Get("Subject")
	}
	if len(subject) > 120 {
		subject = subject[:120] + "..."
	}
	row := &rule.FilterLog{
		TraceID:       h.fc.TraceID,
		QueueID:       h.fc.QueueID,
		IP:            h.fc.ConnectIP.String(),
		Recipient:     recipient,
		Sender:        h.fc.Sender,
		Subject:       subject,
		Stage:         st,
		RuleID:        &res.RuleID,
		ActionApplied: filter.Outcome(res.Action),
		DurationMs:    int(d.Milliseconds()),
	}
	row.FeatureSnapshotJSON = service.SnapshotJSON(h.fc.Snapshot())
	if h.eng.Config.LogConditionTrace {
		row.ConditionTraceJSON = res.TraceJSON
	}
	bodyStage := st == filter.StageBody
	go func(r *rule.FilterLog, intraday bool) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mysql.InsertFilterLog(ctx, h.db, r); err != nil {
			return
		}
		if intraday && h.RedisClient != nil {
			redisstore.RecordIntradayFilterOutcome(ctx, h.RedisClient, string(r.ActionApplied))
		}
	}(row, bodyStage)
}
