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

	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

const customFeaturesCacheTTL = 60 * time.Second

var (
	customFeaturesMu       sync.RWMutex
	customFeaturesCached   []rule.CustomFeature
	customFeaturesLoadedAt time.Time
)

// InvalidateCustomFeaturesCache clears cached enabled custom features (call after admin CUD).
func InvalidateCustomFeaturesCache() {
	customFeaturesMu.Lock()
	customFeaturesCached = nil
	customFeaturesLoadedAt = time.Time{}
	customFeaturesMu.Unlock()
	InvalidateFeatureKeyStagesCache()
}

// CachedCustomFeatures returns enabled custom feature definitions with TTL caching.
func CachedCustomFeatures(ctx context.Context, db *gorm.DB) ([]rule.CustomFeature, error) {
	if db == nil {
		return nil, nil
	}
	now := time.Now()
	customFeaturesMu.RLock()
	if !customFeaturesLoadedAt.IsZero() && now.Sub(customFeaturesLoadedAt) < customFeaturesCacheTTL {
		out := customFeaturesCached
		customFeaturesMu.RUnlock()
		return out, nil
	}
	customFeaturesMu.RUnlock()

	customFeaturesMu.Lock()
	defer customFeaturesMu.Unlock()
	if !customFeaturesLoadedAt.IsZero() && time.Since(customFeaturesLoadedAt) < customFeaturesCacheTTL && customFeaturesCached != nil {
		return customFeaturesCached, nil
	}

	repo := mysql.NewCustomFeatureRepository(db)
	list, err := repo.LoadEnabledCustomFeatures(ctx)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []rule.CustomFeature{}
	}
	customFeaturesCached = list
	customFeaturesLoadedAt = time.Now()
	return list, nil
}
