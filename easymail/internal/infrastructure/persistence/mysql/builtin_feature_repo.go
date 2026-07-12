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

	"easymail/internal/domain/filter/rule"

	"gorm.io/gorm"
)

// BuiltinFeatureRepository implements rule.BuiltinFeatureRepository using MySQL.
type BuiltinFeatureRepository struct {
	db *gorm.DB
}

func NewBuiltinFeatureRepository(db *gorm.DB) *BuiltinFeatureRepository {
	return &BuiltinFeatureRepository{db: db}
}

func (r *BuiltinFeatureRepository) List(ctx context.Context) ([]rule.BuiltinFeature, error) {
	return ListFeatureDefs(ctx, r.db)
}

func (r *BuiltinFeatureRepository) ListPaged(ctx context.Context, page, pageSize int) (int64, []rule.BuiltinFeature, error) {
	return ListFeatureDefsPaged(ctx, r.db, page, pageSize)
}
