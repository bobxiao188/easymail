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
	"time"

	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/persistence/mysql"
	redisstore "easymail/internal/infrastructure/filter/persistence/redis"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// StatsService is the application fa莽ade for async filter statistics (intraday Redis + daily rollups).
type StatsService struct{}

func (StatsService) RecordIntraday(ctx context.Context, rdb *redis.Client, actionApplied string) {
	redisstore.RecordIntradayFilterOutcome(ctx, rdb, actionApplied)
}

// RollupFilterLogsForStatDate delegates to mysqlstore (scheduled jobs / admin).
func (StatsService) RollupFilterLogsForStatDate(ctx context.Context, db *gorm.DB, statDate time.Time) error {
	return mysql.RollupFilterLogsForStatDate(ctx, db, statDate)
}

func (StatsService) InsertLog(ctx context.Context, db *gorm.DB, row *rule.FilterLog) error {
	return mysql.InsertFilterLog(ctx, db, row)
}
