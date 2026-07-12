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
)

const featureKeyStagesCacheTTL = 30 * time.Second

var (
	featureKeyStagesMu       sync.Mutex
	featureKeyStagesCached   map[string]int
	featureKeyStagesLoadedAt time.Time
)

// InvalidateFeatureKeyStagesCache clears cached feature-key 鈫?pipeline stage ordinals.
func InvalidateFeatureKeyStagesCache() {
	featureKeyStagesMu.Lock()
	featureKeyStagesCached = nil
	featureKeyStagesLoadedAt = time.Time{}
	featureKeyStagesMu.Unlock()
}

// CachedFeatureKeyStages returns map[string]int stage ordinals (same numeric values as feature.Stage).
func CachedFeatureKeyStages(ctx context.Context, compute func(context.Context) map[string]int) map[string]int {
	featureKeyStagesMu.Lock()
	defer featureKeyStagesMu.Unlock()
	if featureKeyStagesCached != nil && time.Since(featureKeyStagesLoadedAt) < featureKeyStagesCacheTTL {
		return featureKeyStagesCached
	}
	m := compute(ctx)
	if m == nil {
		m = make(map[string]int)
	}
	featureKeyStagesCached = m
	featureKeyStagesLoadedAt = time.Now()
	return featureKeyStagesCached
}

