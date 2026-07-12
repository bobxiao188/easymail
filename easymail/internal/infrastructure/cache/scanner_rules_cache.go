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

package cache

import (
	"context"
	"sync"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

const filterRulesCacheTTL = 60 * time.Second

var (
	filterRulesMu       sync.RWMutex
	filterRulesCached   []rule.Rule
	filterRulesLoadedAt time.Time
)

// InvalidateFilterRulesCache clears cached enabled filter rules (call after admin rule CUD).
func InvalidateFilterRulesCache() {
	filterRulesMu.Lock()
	filterRulesCached = nil
	filterRulesLoadedAt = time.Time{}
	filterRulesMu.Unlock()
}

// CachedFilterRules returns enabled filter rules ordered by priority DESC with TTL caching.
func CachedFilterRules(ctx context.Context, db *gorm.DB) ([]rule.Rule, error) {
	if db == nil {
		return nil, nil
	}
	now := time.Now()
	filterRulesMu.RLock()
	if !filterRulesLoadedAt.IsZero() && now.Sub(filterRulesLoadedAt) < filterRulesCacheTTL {
		out := filterRulesCached
		filterRulesMu.RUnlock()
		return out, nil
	}
	filterRulesMu.RUnlock()

	filterRulesMu.Lock()
	defer filterRulesMu.Unlock()
	if !filterRulesLoadedAt.IsZero() && time.Since(filterRulesLoadedAt) < filterRulesCacheTTL && filterRulesCached != nil {
		return filterRulesCached, nil
	}
	var pos []mysql.RulePO
	if err := db.WithContext(ctx).Model(&mysql.RulePO{}).Where("enabled = ?", true).Order("priority DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	list := make([]rule.Rule, len(pos))
	for i := range pos {
		p := &pos[i]
		list[i] = rule.Rule{
			ID:            p.ID,
			Name:          p.Name,
			Enabled:       p.Enabled,
			Priority:      p.Priority,
			Action:        filter.Outcome(p.Action),
			ConditionJSON: p.ConditionJSON,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
			IsDeleted:     p.IsDeleted,
			CreatorId:     p.CreatorId,
		}
	}
	filterRulesCached = list
	filterRulesLoadedAt = time.Now()
	return list, nil
}
