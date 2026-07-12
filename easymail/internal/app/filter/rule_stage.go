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

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/filter/extractors"
	"gorm.io/gorm"
)

// computeRuleStageFromIndex returns the pipeline stage at which all features referenced by the condition are available.
func computeRuleStageFromIndex(condJSON string, idx map[string]filter.Stage) filter.Stage {
	keys, err := rule.CollectConditionFeatureKeys(condJSON)
	if err != nil || len(keys) == 0 {
		return filter.StageConnect
	}
	mx := filter.StageConnect
	for _, k := range keys {
		st, ok := idx[k]
		if !ok {
			st = filter.StageBody
		}
		mx = filter.MaxStage(mx, st)
	}
	return mx
}

// ComputeRuleStageFromCondition resolves the evaluation stage for a condition (max of referenced feature key stages).
func ComputeRuleStageFromCondition(ctx context.Context, db *gorm.DB, conditionJSON string) (filter.Stage, error) {
	idx := extractors.FeatureKeyStages(ctx, db)
	return computeRuleStageFromIndex(conditionJSON, idx), nil
}

// SetRuleStageFromCondition sets r.Stage from ConditionJSON using the current feature registry and DB custom features.
// Call this when creating or updating a rule so the milter engine can read Stage without recomputing.
func SetRuleStageFromCondition(ctx context.Context, db *gorm.DB, r *rule.Rule) error {
	if r == nil {
		return nil
	}
	st, err := ComputeRuleStageFromCondition(ctx, db, r.ConditionJSON)
	if err != nil {
		return err
	}
	v := st
	r.Stage = &v
	return nil
}

