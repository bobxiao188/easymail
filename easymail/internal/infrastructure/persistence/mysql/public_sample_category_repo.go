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
	"strings"

	"gorm.io/gorm"
)

// PublicSampleCategoryRepository implements CRUD operations for sample categories.
type PublicSampleCategoryRepository struct {
	db *gorm.DB
}

func NewPublicSampleCategoryRepository(db *gorm.DB) *PublicSampleCategoryRepository {
	return &PublicSampleCategoryRepository{db: db}
}

var (
	ErrCategoryNotFound   = errors.New("category not found")
	ErrCategoryNameExists = errors.New("category name already exists")
	ErrCategoryHasSamples = errors.New("category has samples, cannot delete")
)

// CategoryRow is one row from public_sample_categories table.
type CategoryRow struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SampleCount int64  `json:"sampleCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func (r *PublicSampleCategoryRepository) List(ctx context.Context, keyword string, page, pageSize int) ([]CategoryRow, int64, error) {
	q := r.db.WithContext(ctx).Model(&PublicSampleCategoryPO{})
	if s := strings.TrimSpace(keyword); s != "" {
		q = q.Where("name LIKE ? OR description LIKE ?", "%"+s+"%", "%"+s+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var pos []PublicSampleCategoryPO
	if err := q.Order("id ASC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]CategoryRow, len(pos))
	for i := range pos {
		rows[i] = CategoryRow{
			ID:          pos[i].ID,
			Name:        pos[i].Name,
			Description: pos[i].Description,
			SampleCount: pos[i].SampleCount,
			CreatedAt:   pos[i].CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   pos[i].UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return rows, total, nil
}

func (r *PublicSampleCategoryRepository) GetByID(ctx context.Context, id uint) (*CategoryRow, error) {
	var po PublicSampleCategoryPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &CategoryRow{
		ID:          po.ID,
		Name:        po.Name,
		Description: po.Description,
		SampleCount: po.SampleCount,
		CreatedAt:   po.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   po.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (r *PublicSampleCategoryRepository) Create(ctx context.Context, name, description string) (*CategoryRow, error) {
	// Check if name already exists
	var count int64
	if err := r.db.WithContext(ctx).Model(&PublicSampleCategoryPO{}).Where("name = ?", strings.ToLower(strings.TrimSpace(name))).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrCategoryNameExists
	}

	po := &PublicSampleCategoryPO{
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	return &CategoryRow{
		ID:          po.ID,
		Name:        po.Name,
		Description: po.Description,
		CreatedAt:   po.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   po.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (r *PublicSampleCategoryRepository) Update(ctx context.Context, id uint, name, description string) (*CategoryRow, error) {
	var po PublicSampleCategoryPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if name != "" {
		trimmedName := strings.TrimSpace(name)
		// Check if new name conflicts with existing category (excluding self)
		var count int64
		if err := r.db.WithContext(ctx).Model(&PublicSampleCategoryPO{}).
			Where("name = ? AND id != ?", strings.ToLower(trimmedName), id).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrCategoryNameExists
		}
		updates["name"] = trimmedName
	}
	if description != "" {
		updates["description"] = strings.TrimSpace(description)
	}

	if err := r.db.WithContext(ctx).Model(&PublicSampleCategoryPO{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload to get updated values
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return nil, err
	}
	return &CategoryRow{
		ID:          po.ID,
		Name:        po.Name,
		Description: po.Description,
		SampleCount: po.SampleCount,
		CreatedAt:   po.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   po.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (r *PublicSampleCategoryRepository) Delete(ctx context.Context, id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Check if category exists
		var po PublicSampleCategoryPO
		if err := tx.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCategoryNotFound
			}
			return err
		}

		// Delete all samples in this category first (cascade delete)
		if err := tx.WithContext(ctx).Where("category_id = ?", id).Delete(&PublicSamplePO{}).Error; err != nil {
			return err
		}

		// Delete the category
		return tx.WithContext(ctx).Where("id = ?", id).Delete(&PublicSampleCategoryPO{}).Error
	})
}

// UpdateSampleCount updates the sample count for a category.
func (r *PublicSampleCategoryRepository) UpdateSampleCount(ctx context.Context, id uint, count int64) error {
	return r.db.WithContext(ctx).Model(&PublicSampleCategoryPO{}).Where("id = ?", id).Update("sample_count", count).Error
}

// SyncSampleCounts recalculates and updates sample counts for all categories.
func (r *PublicSampleCategoryRepository) SyncSampleCounts(ctx context.Context) error {
	type result struct {
		CategoryID uint
		Count      int64
	}
	var results []result
	if err := r.db.WithContext(ctx).
		Model(&PublicSamplePO{}).
		Select("category_id, COUNT(*) as count").
		Group("category_id").
		Find(&results).Error; err != nil {
		return err
	}

	// Build map of category_id -> count
	countMap := make(map[uint]int64, len(results))
	for _, r := range results {
		countMap[r.CategoryID] = r.Count
	}

	// Update all categories
	var categories []PublicSampleCategoryPO
	if err := r.db.WithContext(ctx).Find(&categories).Error; err != nil {
		return err
	}

	for _, cat := range categories {
		count, exists := countMap[cat.ID]
		if !exists {
			count = 0
		}
		if cat.SampleCount != count {
			if err := r.db.WithContext(ctx).Model(&PublicSampleCategoryPO{}).Where("id = ?", cat.ID).Update("sample_count", count).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// GetByName returns a category by name (case-insensitive).
func (r *PublicSampleCategoryRepository) GetByName(ctx context.Context, name string) (*PublicSampleCategoryPO, error) {
	var po PublicSampleCategoryPO
	if err := r.db.WithContext(ctx).Where("LOWER(name) = ?", strings.ToLower(name)).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &po, nil
}
