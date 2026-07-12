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

	"gorm.io/gorm"
)

// PublicSampleRepository implements CRUD operations for global training samples.
type PublicSampleRepository struct {
	db *gorm.DB
}

func NewPublicSampleRepository(db *gorm.DB) *PublicSampleRepository {
	return &PublicSampleRepository{db: db}
}

// PublicSampleRow is one row from public_samples table with category name.
type PublicSampleRow struct {
	ID         uint   `json:"id"`
	CategoryID uint   `json:"categoryId"`
	Category   string `json:"category"`
	Tag        string `json:"tag"`
	Text       string `json:"text"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

func (r *PublicSampleRepository) List(ctx context.Context, categoryID uint, tag, keyword string, page, pageSize int) ([]PublicSampleRow, int64, error) {
	q := r.db.WithContext(ctx).
		Table("public_samples s").
		Select("s.id, s.category_id, c.name as category, s.tag, s.text, s.created_at, s.updated_at").
		Joins("LEFT JOIN public_sample_categories c ON s.category_id = c.id")
	if categoryID > 0 {
		q = q.Where("s.category_id = ?", categoryID)
	}
	if s := strings.TrimSpace(tag); s != "" {
		q = q.Where("s.tag = ?", s)
	}
	if s := strings.TrimSpace(keyword); s != "" {
		q = q.Where("s.text LIKE ?", "%"+s+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	type rawRow struct {
		ID         uint   `json:"id"`
		CategoryID uint   `json:"category_id"`
		Category   string `json:"category"`
		Tag        string `json:"tag"`
		Text       string `json:"text"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	}
	var rawRows []rawRow
	if err := q.Order("s.id ASC").Offset(offset).Limit(pageSize).Find(&rawRows).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]PublicSampleRow, len(rawRows))
	for i := range rawRows {
		rows[i] = PublicSampleRow{
			ID:         rawRows[i].ID,
			CategoryID: rawRows[i].CategoryID,
			Category:   rawRows[i].Category,
			Tag:        rawRows[i].Tag,
			Text:       rawRows[i].Text,
			CreatedAt:  rawRows[i].CreatedAt,
			UpdatedAt:  rawRows[i].UpdatedAt,
		}
	}
	return rows, total, nil
}

func (r *PublicSampleRepository) ListTags(ctx context.Context, categoryID uint) ([]string, error) {
	var tags []string
	q := r.db.WithContext(ctx).Model(&PublicSamplePO{}).Select("DISTINCT tag").Order("tag ASC")
	if categoryID > 0 {
		q = q.Where("category_id = ?", categoryID)
	}
	if err := q.Pluck("tag", &tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *PublicSampleRepository) Create(ctx context.Context, categoryID uint, tag, text string) error {
	po := &PublicSamplePO{
		CategoryID: categoryID,
		Tag:        tag,
		Text:       text,
	}
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *PublicSampleRepository) CreateBatch(ctx context.Context, items []PublicSamplePO) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

func (r *PublicSampleRepository) Update(ctx context.Context, id uint, categoryID uint, tag, text string) error {
	updates := map[string]interface{}{}
	if categoryID > 0 {
		updates["category_id"] = categoryID
	}
	if tag != "" {
		updates["tag"] = tag
	}
	if text != "" {
		updates["text"] = text
	}
	return r.db.WithContext(ctx).Model(&PublicSamplePO{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PublicSampleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&PublicSamplePO{}).Error
}

func (r *PublicSampleRepository) DeleteBatch(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&PublicSamplePO{}).Error
}

func (r *PublicSampleRepository) UpdateBatch(ctx context.Context, ids []uint, categoryID uint, tag string) error {
	if len(ids) == 0 {
		return nil
	}
	updates := map[string]interface{}{}
	if categoryID > 0 {
		updates["category_id"] = categoryID
	}
	if tag != "" {
		updates["tag"] = tag
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&PublicSamplePO{}).Where("id IN ?", ids).Updates(updates).Error
}

// DescribeSamples returns sample counts grouped by category and tag.
func (r *PublicSampleRepository) DescribeSamples(ctx context.Context, categoryID uint) ([]struct {
	CategoryID uint   `json:"categoryId"`
	Category   string `json:"category"`
	Tag        string `json:"tag"`
	Count      int64  `json:"count"`
}, error) {
	type rawRow struct {
		CategoryID uint   `json:"category_id"`
		Category   string `json:"name"`
		Tag        string `json:"tag"`
		Count      int64  `json:"count"`
	}
	var rawRows []rawRow
	query := r.db.WithContext(ctx).
		Table("public_samples s").
		Select("s.category_id, c.name as name, s.tag, COUNT(*) as count").
		Joins("LEFT JOIN public_sample_categories c ON s.category_id = c.id")
	if categoryID > 0 {
		query = query.Where("s.category_id = ?", categoryID)
	}
	query = query.
		Group("s.category_id, c.name, s.tag").
		Order("c.name ASC, s.tag ASC")
	if err := query.Find(&rawRows).Error; err != nil {
		return nil, err
	}
	rows := make([]struct {
		CategoryID uint   `json:"categoryId"`
		Category   string `json:"category"`
		Tag        string `json:"tag"`
		Count      int64  `json:"count"`
	}, len(rawRows))
	for i := range rawRows {
		rows[i] = struct {
			CategoryID uint   `json:"categoryId"`
			Category   string `json:"category"`
			Tag        string `json:"tag"`
			Count      int64  `json:"count"`
		}{
			CategoryID: rawRows[i].CategoryID,
			Category:   rawRows[i].Category,
			Tag:        rawRows[i].Tag,
			Count:      rawRows[i].Count,
		}
	}
	return rows, nil
}
