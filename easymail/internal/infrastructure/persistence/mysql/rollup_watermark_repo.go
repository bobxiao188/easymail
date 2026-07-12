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
	"errors"

	"easymail/internal/domain/filter/stats"

	"gorm.io/gorm"
)

// RollupWatermarkRepository implements stats.RollupWatermarkRepository using MySQL.
type RollupWatermarkRepository struct {
	db *gorm.DB
}

func NewRollupWatermarkRepository(db *gorm.DB) *RollupWatermarkRepository {
	return &RollupWatermarkRepository{db: db}
}

func (r *RollupWatermarkRepository) Load(ctx context.Context) (*stats.FilterStatsRollupWatermark, error) {
	var po FilterStatsRollupWatermarkPO
	err := r.db.WithContext(ctx).Where("job_name = ?", rollupJobFilterLogs).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &stats.FilterStatsRollupWatermark{JobName: rollupJobFilterLogs}, nil
	}
	if err != nil {
		return nil, err
	}
	return poToFilterStatsRollupWatermark(&po), nil
}

func (r *RollupWatermarkRepository) Save(ctx context.Context, wm *stats.FilterStatsRollupWatermark) error {
	po := filterStatsRollupWatermarkToPO(wm)
	return r.db.WithContext(ctx).Where("job_name = ?", rollupJobFilterLogs).Save(po).Error
}
