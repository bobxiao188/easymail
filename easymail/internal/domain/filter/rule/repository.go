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
	"context"
	"time"

	"easymail/internal/domain/filter"
)

// FeatureExtractor produces numeric features from scan session state at a pipeline Stage.
// Implementations live in package extractors; registration is side-effect imported from feature.
type FeatureExtractor interface {
	Key() string
	Stage() filter.Stage
	Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error)
}

// CustomFeatureRepository loads enabled custom-feature definitions.
type CustomFeatureRepository interface {
	LoadEnabledCustomFeatures(ctx context.Context) ([]CustomFeature, error)
}

// ScanStatsRepository persists filter outcomes and rollups.
type ScanStatsRepository interface {
	InsertScanLog(ctx context.Context, row *FilterLog) error
	RecordIntradayOutcome(ctx context.Context, actionApplied filter.Outcome) error
}

// RuleRepository is the CRUD port for antispam rules.
type RuleRepository interface {
	List(ctx context.Context) ([]Rule, error)
	ListPaged(ctx context.Context, page, pageSize int) (total int64, rows []Rule, err error)
	GetByID(ctx context.Context, id int64) (*Rule, error)
	Create(ctx context.Context, r *Rule) error
	Update(ctx context.Context, r *Rule) error
	Delete(ctx context.Context, id int64) error
}

// FilterLogRepository is the read/write port for delivery audit logs.
type FilterLogRepository interface {
	Insert(ctx context.Context, row *FilterLog) error
	GetByID(ctx context.Context, id int64) (*FilterLog, error)
	ListPaged(ctx context.Context, page, pageSize int, ip, sender, recipient string, createdFrom, createdTo *time.Time) (total int64, rows []FilterLog, err error)
}

// BuiltinFeatureRepository is the read-only port for built-in feature definitions.
type BuiltinFeatureRepository interface {
	List(ctx context.Context) ([]BuiltinFeature, error)
	ListPaged(ctx context.Context, page, pageSize int) (total int64, rows []BuiltinFeature, err error)
}
