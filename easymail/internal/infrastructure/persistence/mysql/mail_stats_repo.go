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
	"time"

	"easymail/internal/domain/filter/stats"

	"gorm.io/gorm"
)

// MailStatsRepository implements stats.MailStatsRepository using MySQL.
type MailStatsRepository struct {
	db *gorm.DB
}

func NewMailStatsRepository(db *gorm.DB) *MailStatsRepository {
	return &MailStatsRepository{db: db}
}

func (r *MailStatsRepository) RollupForDate(ctx context.Context, statDate time.Time) error {
	return RollupFilterLogsForStatDate(ctx, r.db, statDate)
}

func (r *MailStatsRepository) ListByDateRange(ctx context.Context, from, to time.Time) ([]stats.FilterMailStatsDaily, error) {
	var pos []FilterMailStatsDailyPO
	if err := r.db.WithContext(ctx).
		Where("stat_date >= ? AND stat_date <= ?", from, to).
		Order("stat_date ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}
	out := make([]stats.FilterMailStatsDaily, len(pos))
	for i := range pos {
		out[i] = *poToFilterMailStatsDaily(&pos[i])
	}
	return out, nil
}
