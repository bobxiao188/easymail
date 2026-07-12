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

	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/persistence/mysql"
	"easymail/pkg/database"

	"gorm.io/gorm"
)

const classifyModelsCacheTTL = 60 * time.Second

var (
	classifyModelsMu       sync.RWMutex
	classifyModelsCached   []classifier.Model
	classifyModelsLoadedAt time.Time
)

// InvalidateClassifyModelsCache clears the in-memory classify_models snapshot (call after admin CUD).
func InvalidateClassifyModelsCache() {
	classifyModelsMu.Lock()
	classifyModelsCached = nil
	classifyModelsLoadedAt = time.Time{}
	classifyModelsMu.Unlock()
	InvalidateFeatureKeyStagesCache()
}

// CachedClassifyModels returns every row in classify_models (enabled or not, deleted or not), ordered by id ASC, with TTL caching.
// Call InvalidateClassifyModelsCache after mutations so readers see fresh data without waiting for TTL.
func CachedClassifyModels(ctx context.Context, db *gorm.DB) ([]classifier.Model, error) {
	if db == nil {
		db = database.GetDB()
	}
	if db == nil {
		return nil, nil
	}
	now := time.Now()
	classifyModelsMu.RLock()
	if !classifyModelsLoadedAt.IsZero() && now.Sub(classifyModelsLoadedAt) < classifyModelsCacheTTL {
		out := classifyModelsCached
		classifyModelsMu.RUnlock()
		return cloneClassifyModelSlice(out), nil
	}
	classifyModelsMu.RUnlock()

	classifyModelsMu.Lock()
	defer classifyModelsMu.Unlock()
	if !classifyModelsLoadedAt.IsZero() && time.Since(classifyModelsLoadedAt) < classifyModelsCacheTTL && classifyModelsCached != nil {
		return cloneClassifyModelSlice(classifyModelsCached), nil
	}
	var pos []mysql.ClassifyModelPO
	if err := db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).Order("id ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	list := make([]classifier.Model, len(pos))
	for i := range pos {
		list[i] = *mysql.PoToClassifyModel(&pos[i])
	}
	classifyModelsCached = list
	classifyModelsLoadedAt = time.Now()
	return cloneClassifyModelSlice(list), nil
}

func cloneClassifyModelSlice(src []classifier.Model) []classifier.Model {
	if len(src) == 0 {
		return []classifier.Model{}
	}
	out := make([]classifier.Model, len(src))
	copy(out, src)
	return out
}



