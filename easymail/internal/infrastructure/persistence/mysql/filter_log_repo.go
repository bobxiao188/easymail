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

	"easymail/internal/domain/filter/rule"

	"gorm.io/gorm"
)

// FilterLogRepository implements rule.FilterLogRepository using MySQL.
type FilterLogRepository struct {
	db *gorm.DB
}

func NewFilterLogRepository(db *gorm.DB) *FilterLogRepository {
	return &FilterLogRepository{db: db}
}

func (r *FilterLogRepository) Insert(ctx context.Context, row *rule.FilterLog) error {
	return InsertFilterLog(ctx, r.db, row)
}

func (r *FilterLogRepository) GetByID(ctx context.Context, id int64) (*rule.FilterLog, error) {
	return GetFilterLog(ctx, r.db, id)
}

func (r *FilterLogRepository) ListPaged(ctx context.Context, page, pageSize int, ip, sender, recipient string, createdFrom, createdTo *time.Time) (int64, []rule.FilterLog, error) {
	f := &FilterLogFilter{
		IP:        ip,
		Sender:    sender,
		Recipient: recipient,
	}
	if createdFrom != nil {
		f.CreatedAfter = createdFrom
	}
	if createdTo != nil {
		f.CreatedBefore = createdTo
	}
	return ListFilterLogs(ctx, r.db, page, pageSize, f)
}
