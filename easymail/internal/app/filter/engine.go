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
	"context"
	"net/textproto"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/filter/extractors"
	"easymail/pkg/config"

	"gorm.io/gorm"
)

// RuleEngine loads filter rules and evaluates them per milter stage.
type RuleEngine struct {
	DB     *gorm.DB
	Config config.FilterConfig
}

// MatchResult is the first matching rule or the default policy.
type MatchResult struct {
	Action      string // accept | spam | quarantine | reject
	RuleID      int64  // 0 means default policy
	FeatureSnap string
	TraceJSON   string
}

func (e *RuleEngine) featureKeyStages(ctx context.Context) map[string]filter.Stage {
	if e == nil {
		return extractors.FeatureKeyStages(ctx, nil)
	}
	// extractors.FeatureKeyStages already caches internally via CachedFeatureKeyStages.
	// Do NOT wrap this call in another CachedFeatureKeyStages layer ? that would
	// reacquire the same sync.Mutex and cause a deadlock on cold cache.
	return extractors.FeatureKeyStages(ctx, e.DB)
}

// ruleEvalStage returns the pipeline phase for a rule: persisted Stage when set, otherwise derived from ConditionJSON.
func (e *RuleEngine) ruleEvalStage(r *rule.Rule, idx map[string]filter.Stage) filter.Stage {
	if r != nil && r.Stage != nil {
		s := filter.Stage(*r.Stage)
		if s >= filter.StageConnect && s <= filter.StageBody {
			return s
		}
	}
	if r == nil {
		return filter.StageConnect
	}
	return computeRuleStageFromIndex(r.ConditionJSON, idx)
}

// Evaluate matches rules in the body stage from extracted features (legacy helper).
func (e *RuleEngine) Evaluate(ctx context.Context, hdr textproto.MIMEHeader, body []byte, rcptCount int) (*MatchResult, error) {
	feat := extractors.ExtractMessageBaseline(hdr, body, rcptCount)
	snap := SnapshotJSON(feat)
	return e.EvaluateWithFeatures(ctx, feat, snap)
}

// EvaluateWithFeatures evaluates rules with an already-built feature map (body stage only).
func (e *RuleEngine) EvaluateWithFeatures(ctx context.Context, feat map[string]float64, snap string) (*MatchResult, error) {
	return e.EvaluateWithFeaturesAtStage(ctx, filter.StageBody, feat, snap)
}

// EvaluateWithFeaturesAtStage evaluates only rules whose dependency stage equals st (after feature extraction for that stage).
func (e *RuleEngine) EvaluateWithFeaturesAtStage(ctx context.Context, st filter.Stage, feat map[string]float64, snap string) (*MatchResult, error) {
	if snap == "" {
		snap = SnapshotJSON(feat)
	}

	if e == nil {
		return &MatchResult{Action: "accept", RuleID: 0, FeatureSnap: snap, TraceJSON: "{}"}, nil
	}

	if e.DB == nil || !e.Config.Enable {
		act := normalizeAction(e.Config.DefaultAction)
		return &MatchResult{Action: act, RuleID: 0, FeatureSnap: snap, TraceJSON: "{}"}, nil
	}

	idx := e.featureKeyStages(ctx)

	rules, err := cache.CachedFilterRules(ctx, e.DB)
	if err != nil {
		return nil, err
	}

	for i := range rules {
		r := &rules[i]
		if e.ruleEvalStage(r, idx) != st {
			continue
		}
		trace := make(map[string]bool)
		ok, err := EvalConditionJSON(r.ConditionJSON, feat, trace)
		if err != nil || !ok {
			continue
		}
		act := normalizeAction(string(r.Action))
		tj := "{}"
		if e.Config.LogConditionTrace {
			tj = TraceToJSON(trace)
		}
		return &MatchResult{Action: act, RuleID: r.ID, FeatureSnap: snap, TraceJSON: tj}, nil
	}

	act := normalizeAction(e.Config.DefaultAction)
	return &MatchResult{Action: act, RuleID: 0, FeatureSnap: snap, TraceJSON: "{}"}, nil
}

func normalizeAction(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "spam", "quarantine", "reject":
		return strings.ToLower(strings.TrimSpace(a))
	default:
		return "accept"
	}
}
