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

package redisstore

import (
	"context"
	"fmt"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/infrastructure/filter/utcdate"

	"github.com/redis/go-redis/v9"
)

// FilterDayStatsPrefix is the Redis HASH prefix for intraday filter policy aggregates (UTC calendar day).
const FilterDayStatsPrefix = "easymail:filter:stats:day:"

// RecordIntradayFilterOutcome increments the UTC-day HASH for normalized policy outcome; TTL until end of day UTC plus 48h.
func RecordIntradayFilterOutcome(ctx context.Context, rdb *redis.Client, actionApplied string) {
	if rdb == nil {
		return
	}
	now := time.Now().UTC()
	day := utcdate.DateUTC(now).Format("2006-01-02")
	o := filter.NormalizeOutcome(actionApplied)
	field := fmt.Sprintf("count_%s", string(o))
	key := FilterDayStatsPrefix + day
	pipe := rdb.TxPipeline()
	pipe.HIncrBy(ctx, key, "total_count", 1)
	pipe.HIncrBy(ctx, key, field, 1)
	pipe.HSet(ctx, key, "updated_at", now.Format(time.RFC3339))
	exp := endOfUTCDay(now).Add(48 * time.Hour)
	pipe.ExpireAt(ctx, key, exp)
	_, _ = pipe.Exec(ctx)
}

func endOfUTCDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
}
