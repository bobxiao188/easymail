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

package mysql

import (
	"context"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	redisstore "easymail/internal/infrastructure/filter/persistence/redis"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ScanStatsRepository implements rule.ScanStatsRepository using MySQL for persistence and Redis for intraday counters.
type ScanStatsRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewScanStatsRepository creates a new scan stats repository.
func NewScanStatsRepository(db *gorm.DB, rdb *redis.Client) *ScanStatsRepository {
	return &ScanStatsRepository{db: db, rdb: rdb}
}

// InsertScanLog persists a filter log row (best-effort for async writes).
func (r *ScanStatsRepository) InsertScanLog(ctx context.Context, row *rule.FilterLog) error {
	return InsertFilterLog(ctx, r.db, row)
}

// RecordIntradayOutcome increments the intraday outcome counter for the normalized action.
func (r *ScanStatsRepository) RecordIntradayOutcome(ctx context.Context, actionApplied filter.Outcome) error {
	redisstore.RecordIntradayFilterOutcome(ctx, r.rdb, string(actionApplied))
	return nil
}
