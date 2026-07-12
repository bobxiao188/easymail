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

// CustomFeatureRepository implements rule.CustomFeatureRepository using MySQL.
type CustomFeatureRepository struct {
	db *gorm.DB
}

// NewCustomFeatureRepository creates a new MySQL-backed custom feature repository.
func NewCustomFeatureRepository(db *gorm.DB) *CustomFeatureRepository {
	return &CustomFeatureRepository{db: db}
}

// LoadEnabledCustomFeatures returns all enabled custom feature definitions ordered by id DESC.
func (r *CustomFeatureRepository) LoadEnabledCustomFeatures(ctx context.Context) ([]rule.CustomFeature, error) {
	var pos []CustomFeaturePO
	if err := r.db.WithContext(ctx).
		Model(&CustomFeaturePO{}).
		Where("enabled = ?", true).
		Order("id DESC").
		Find(&pos).Error; err != nil {
		return nil, err
	}
	out := make([]rule.CustomFeature, len(pos))
	for i := range pos {
		out[i] = *poToCustomFeature(&pos[i])
	}
	return out, nil
}
