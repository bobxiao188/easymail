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

package rule

import (
	"fmt"
	"strings"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/classifier"
)

// BuiltinFeature is a built-in filter feature definition (read-only in admin; used for AST).
type BuiltinFeature struct {
	ID          int64        `json:"id"`
	FeatureKey  string       `json:"featureKey"`
	Label       string       `json:"label"`
	ValueType   string       `json:"valueType"`
	Stage       filter.Stage `json:"stage,omitempty"`
	Description string       `json:"description"`
	Unit        string       `json:"unit"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	// ModelSource indicates if this feature is generated from a classifier model
	ModelSource *ModelSource `json:"modelSource,omitempty"`
}

// ModelSource represents the classifier model that generated this builtin feature.
type ModelSource struct {
	ModelID       uint   `json:"modelId"`
	ModelName     string `json:"modelName"`
	FeatureOrigin string `json:"featureOrigin"` // "model_name" or "model_name_label"
}

// Rule is an antispam rule row (higher priority is evaluated first).
type Rule struct {
	ID            int64          `json:"id"`
	Name          string         `json:"name"`
	Enabled       bool           `json:"enabled"`
	Priority      int            `json:"priority"`
	Stage         *filter.Stage  `json:"stage,omitempty"`
	Action        filter.Outcome `json:"action"`
	ConditionJSON string         `json:"conditionJson"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	IsDeleted     bool           `json:"isDeleted"`
	CreatorId     int64          `json:"creatorId"`
}

// FilterLog is a best-effort milter session audit row.
type FilterLog struct {
	ID                  int64          `json:"id"`
	TraceID             string         `json:"traceId"`
	QueueID             string         `json:"queueId"`
	IP                  string         `json:"ip"`
	Sender              string         `json:"sender"`
	Recipient           string         `json:"recipient"`
	Subject             string         `json:"subject"`
	Stage               filter.Stage   `json:"stage,omitempty"`
	RuleID              *int64         `json:"ruleId,omitempty"`
	ActionApplied       filter.Outcome `json:"actionApplied"`
	FeatureSnapshotJSON string         `json:"featureSnapshotJson"`
	ConditionTraceJSON  string         `json:"conditionTraceJson"`
	DurationMs          int            `json:"durationMs"`
	CreatedAt           time.Time      `json:"createdAt"`
}

// CustomFeature is an admin-defined metadata feature.
type CustomFeature struct {
	ID          int64        `json:"id"`
	FeatureKey  string       `json:"featureKey"`
	Label       string       `json:"label"`
	Stage       filter.Stage `json:"stage,omitempty"`
	Type        string       `json:"type"`
	ValueType   string       `json:"valueType"`
	Enabled     bool         `json:"enabled"`
	SpecJSON    string       `json:"specJson"`
	Description string       `json:"description"`
	Unit        string       `json:"unit"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	IsDeleted   bool         `json:"isDeleted"`
	CreatorId   int64        `json:"creatorId"`
}

// FilterResult is the outcome of a scan pipeline stage.
type FilterResult struct {
	Action      filter.Outcome
	RuleID      int64
	FeatureSnap string
	TraceJSON   string
	ModelScores []classifier.ModelScore
}

// RuleFeatureEntry is one Body-stage feature key derived from a model (rules UI + stage index).
type RuleFeatureEntry struct {
	Key          string
	DisplayLabel string
	// ModelSource optional source info for UI display
	ModelSource *ModelSource `json:"modelSource,omitempty"`
}

// RuleFeatureEntries returns feature keys emitted when the model runs with persisted ClassLabels.
func RuleFeatureEntries(m classifier.Model) []RuleFeatureEntry {
	base := classifier.SanitizeFeatureKey(strings.TrimSpace(m.Name))
	if base == "" {
		return nil
	}
	displayName := strings.TrimSpace(m.Name)
	modelSource := &ModelSource{
		ModelID:       uint(m.ID),
		ModelName:     displayName,
		FeatureOrigin: "model_name",
	}
	if len(m.ClassLabels) == 0 {
		return []RuleFeatureEntry{{Key: base, DisplayLabel: displayName, ModelSource: modelSource}}
	}
	out := make([]RuleFeatureEntry, 0, len(m.ClassLabels))
	for _, raw := range m.ClassLabels {
		lab := strings.TrimSpace(raw)
		if lab == "" {
			continue
		}
		sub := classifier.SanitizeFeatureKey(lab)
		if sub == "" {
			continue
		}
		fk := classifier.SanitizeFeatureKey(base + "_" + sub)
		if fk == "" {
			continue
		}
		out = append(out, RuleFeatureEntry{
			Key:          fk,
			DisplayLabel: fmt.Sprintf("%s %s %s", displayName, "·", lab),
			ModelSource: &ModelSource{
				ModelID:       uint(m.ID),
				ModelName:     displayName,
				FeatureOrigin: "model_name_label",
			},
		})
	}
	if len(out) == 0 {
		return []RuleFeatureEntry{{Key: base, DisplayLabel: displayName, ModelSource: modelSource}}
	}
	return out
}
