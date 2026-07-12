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

// Package mailstats records delivery outcomes in Redis (HASH per day/month) for admin overview.
package mailstats

import (
	"context"
	"strings"
	"time"

	"easymail/pkg/constants"

	"github.com/redis/go-redis/v9"
)

const (
	MailStatsDailyPrefix   = "easymail:mail:daily:"
	MailStatsMonthlyPrefix = "easymail:mail:monthly:"
	TopSenderPrefix        = "easymail:topsender:"
)

// Recorder increments Redis HASH counters for mail delivery stats (LMTP / inbound path).
type Recorder struct {
	rdb *redis.Client
}

func NewRecorder(rdb *redis.Client) *Recorder {
	return &Recorder{rdb: rdb}
}

// RecordMail increments daily/monthly aggregates. action is accept|spam|reject|quarantine.
func (r *Recorder) RecordMail(ctx context.Context, action string, size int64) error {
	if r == nil || r.rdb == nil {
		return nil
	}
	now := time.Now()
	dailyKey := MailStatsDailyPrefix + now.Format("2006-01-02")
	monthlyKey := MailStatsMonthlyPrefix + now.Format("2006-01")

	pipe := r.rdb.TxPipeline()

	pipe.HIncrBy(ctx, dailyKey, "total_count", 1)
	pipe.HIncrBy(ctx, dailyKey, "total_size", size)

	pipe.HIncrBy(ctx, monthlyKey, "total_count", 1)
	pipe.HIncrBy(ctx, monthlyKey, "total_size", size)

	switch action {
	case "accept":
		pipe.HIncrBy(ctx, dailyKey, "normal_count", 1)
		pipe.HIncrBy(ctx, dailyKey, "normal_size", size)
		pipe.HIncrBy(ctx, monthlyKey, "normal_count", 1)
		pipe.HIncrBy(ctx, monthlyKey, "normal_size", size)
	case "spam":
		pipe.HIncrBy(ctx, dailyKey, "spam_count", 1)
		pipe.HIncrBy(ctx, dailyKey, "spam_size", size)
		pipe.HIncrBy(ctx, monthlyKey, "spam_count", 1)
		pipe.HIncrBy(ctx, monthlyKey, "spam_size", size)
	case "reject":
		pipe.HIncrBy(ctx, dailyKey, "reject_count", 1)
		pipe.HIncrBy(ctx, dailyKey, "reject_size", size)
		pipe.HIncrBy(ctx, monthlyKey, "reject_count", 1)
		pipe.HIncrBy(ctx, monthlyKey, "reject_size", size)
	case "quarantine":
		pipe.HIncrBy(ctx, dailyKey, "quarantine_count", 1)
		pipe.HIncrBy(ctx, dailyKey, "quarantine_size", size)
		pipe.HIncrBy(ctx, monthlyKey, "quarantine_count", 1)
		pipe.HIncrBy(ctx, monthlyKey, "quarantine_size", size)
	}

	pipe.HSet(ctx, dailyKey, "updated_at", now.Format(time.RFC3339))
	pipe.HSet(ctx, monthlyKey, "updated_at", now.Format(time.RFC3339))

	pipe.Expire(ctx, dailyKey, 30*24*time.Hour)
	pipe.Expire(ctx, monthlyKey, 12*30*24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

// RecordSender increments top-sender HASH for domain or address keys.
func (r *Recorder) RecordSender(ctx context.Context, sender string, senderType string) error {
	if r == nil || r.rdb == nil {
		return nil
	}
	sender = strings.ToLower(strings.TrimSpace(sender))
	if sender == "" {
		return nil
	}
	now := time.Now()
	key := TopSenderPrefix + senderType + ":" + sender

	pipe := r.rdb.TxPipeline()
	pipe.HIncrBy(ctx, key, "count", 1)
	pipe.HSet(ctx, key, "updated_at", now.Format(time.RFC3339))
	pipe.Expire(ctx, key, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	return err
}

// FolderKindToAction maps mailbox folder kind (after LMTP delivery) to mailstats action.
func FolderKindToAction(kind constants.FolderID) string {
	switch kind {
	case constants.Spam:
		return "spam"
	case constants.Quarantine:
		return "quarantine"
	default:
		return "accept"
	}
}
