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

	"easymail/internal/domain/filter/classifier"

	"gorm.io/gorm"
)

// ModelConfigRepository implements classifier.ModelConfigRepository using MySQL.
type ModelConfigRepository struct {
	db *gorm.DB
}

// NewModelConfigRepository creates a new MySQL-backed model config repository.
func NewModelConfigRepository(db *gorm.DB) *ModelConfigRepository {
	return &ModelConfigRepository{db: db}
}

// AllModels returns every row in classify_models, ordered by id ASC.
func (r *ModelConfigRepository) AllModels(ctx context.Context) ([]classifier.Model, error) {
	var pos []ClassifyModelPO
	if err := r.db.WithContext(ctx).Model(&ClassifyModelPO{}).Order("id ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	out := make([]classifier.Model, len(pos))
	for i := range pos {
		out[i] = *PoToClassifyModel(&pos[i])
	}
	return out, nil
}
