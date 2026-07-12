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
	"log"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/stats"
	"easymail/internal/infrastructure/filter/utcdate"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const rollupJobFilterLogs = "filter_logs_daily"

type rollupAggRow struct {
	Action string `gorm:"column:action"`
	Cnt    int64  `gorm:"column:cnt"`
}

// RollupFilterLogsForStatDate aggregates filter_logs for [statDate, statDate+1) UTC into filter_mail_stats_daily and is idempotent per day.
func RollupFilterLogsForStatDate(ctx context.Context, db *gorm.DB, statDate time.Time) error {
	if db == nil {
		return nil
	}
	statDate = utcdate.DateUTC(statDate)
	start := statDate
	end := statDate.Add(24 * time.Hour)

	var raw []rollupAggRow
	q := `
SELECT LOWER(TRIM(COALESCE(action_applied, ''))) AS action, COUNT(*) AS cnt
FROM filter_logs
WHERE created_at >= ? AND created_at < ?
GROUP BY LOWER(TRIM(COALESCE(action_applied, '')))
`
	if err := db.WithContext(ctx).Raw(q, start, end).Scan(&raw).Error; err != nil {
		return err
	}

	merged := map[string]int64{}
	for _, row := range raw {
		o := filter.NormalizeOutcome(row.Action)
		merged[string(o)] += row.Cnt
	}

	// Calendar day as YYYY-MM-DD for stable DATE matching (avoids TZ drift vs DELETE + INSERT).
	dayStr := statDate.Format("2006-01-02")

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(merged) == 0 {
			return tx.Where("stat_date = ?", dayStr).Delete(&FilterMailStatsDailyPO{}).Error
		}

		keys := make([]string, 0, len(merged))
		for action, cnt := range merged {
			if cnt == 0 {
				continue
			}
			keys = append(keys, action)
			row := FilterMailStatsDailyPO{
				StatDate:      statDate,
				ActionApplied: action,
				MailCount:     cnt,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "stat_date"},
					{Name: "action_applied"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"mail_count", "updated_at"}),
			}).Create(&row).Error; err != nil {
				return err
			}
		}

		// Remove outcome rows that no longer appear in this day's aggregate (stale after re-roll).
		if len(keys) == 0 {
			return nil
		}
		return tx.Where("stat_date = ? AND action_applied NOT IN ?", dayStr, keys).
			Delete(&FilterMailStatsDailyPO{}).Error
	})
}

func loadOrCreateWatermark(ctx context.Context, db *gorm.DB) (*stats.FilterStatsRollupWatermark, error) {
	var wm FilterStatsRollupWatermarkPO
	err := db.WithContext(ctx).Where("job_name = ?", rollupJobFilterLogs).First(&wm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		wm = FilterStatsRollupWatermarkPO{JobName: rollupJobFilterLogs}
		if err := db.WithContext(ctx).Create(&wm).Error; err != nil {
			return nil, err
		}
		return poToFilterStatsRollupWatermark(&wm), nil
	}
	if err != nil {
		return nil, err
	}
	return poToFilterStatsRollupWatermark(&wm), nil
}

// RunFilterLogsRollupCatchUp rolls each missing UTC calendar day from (last watermark + 1) through yesterday. Cold start rolls only yesterday.
func RunFilterLogsRollupCatchUp(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	wm, err := loadOrCreateWatermark(ctx, db)
	if err != nil {
		return err
	}
	yEnd := utcdate.YesterdayUTC(time.Now())
	var startDay time.Time
	if wm.LastCompletedStatDate == nil {
		startDay = yEnd
	} else {
		last := utcdate.DateUTC(*wm.LastCompletedStatDate)
		startDay = last.AddDate(0, 0, 1)
	}
	for d := startDay; !d.After(yEnd); d = d.AddDate(0, 0, 1) {
		if err := RollupFilterLogsForStatDate(ctx, db, d); err != nil {
			return err
		}
		completed := d
		if err := db.WithContext(ctx).Model(&FilterStatsRollupWatermarkPO{}).
			Where("job_name = ?", rollupJobFilterLogs).
			Update("last_completed_stat_date", completed).Error; err != nil {
			return err
		}
	}
	return nil
}

// StartFilterLogsRollupWorker periodically runs catch-up until ctx is cancelled.
func StartFilterLogsRollupWorker(ctx context.Context, db *gorm.DB, interval time.Duration) {
	if db == nil || interval <= 0 {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	run := func() {
		c, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := RunFilterLogsRollupCatchUp(c, db); err != nil {
			log.Printf("easymail: filter_logs rollup: %v", err)
		}
	}
	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}
