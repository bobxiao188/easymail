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
	"strings"

	"easymail/internal/domain/filter/classifier"

	"gorm.io/gorm"
)

// ModelRepository implements classifier.ModelRepository using MySQL.
type ModelRepository struct {
	db *gorm.DB
}

func NewModelRepository(db *gorm.DB) *ModelRepository {
	return &ModelRepository{db: db}
}

func (r *ModelRepository) GetByID(ctx context.Context, id int64) (*classifier.Model, error) {
	var po ClassifyModelPO
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
		return nil, err
	}
	return PoToClassifyModel(&po), nil
}

func (r *ModelRepository) List(ctx context.Context, keyword, algorithm string, status *int, page, pageSize int) ([]classifier.Model, int64, error) {
	q := r.db.WithContext(ctx).Model(&ClassifyModelPO{})
	if s := strings.TrimSpace(keyword); s != "" {
		q = q.Where("name LIKE ?", "%"+s+"%")
	}
	if s := strings.TrimSpace(algorithm); s != "" {
		q = q.Where("algorithm = ?", s)
	}
	if status != nil {
		enabled := *status == 1
		q = q.Where("enabled = ?", enabled)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var pos []ClassifyModelPO
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	out := make([]classifier.Model, len(pos))
	for i := range pos {
		out[i] = *PoToClassifyModel(&pos[i])
	}
	return out, total, nil
}

func (r *ModelRepository) Create(ctx context.Context, m *classifier.Model) error {
	po := ClassifyModelToPO(m)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *ModelRepository) Update(ctx context.Context, m *classifier.Model) error {
	po := ClassifyModelToPO(m)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *ModelRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ClassifyModelPO{}, id).Error
}
