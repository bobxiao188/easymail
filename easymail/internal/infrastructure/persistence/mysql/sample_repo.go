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
	"fmt"
	"strings"

	"easymail/internal/domain/filter/classifier"

	"gorm.io/gorm"
)

// SampleRepository implements classifier.SampleRepository using MySQL.
type SampleRepository struct {
	db *gorm.DB
}

func NewSampleRepository(db *gorm.DB) *SampleRepository {
	return &SampleRepository{db: db}
}

func (r *SampleRepository) List(ctx context.Context, modelID int64, keyword, labelFilter string, page, pageSize int) ([]classifier.Sample, int64, error) {
	q := r.db.WithContext(ctx).Model(&ModelSamplePO{}).Where("classify_model_id = ?", modelID)
	if s := strings.TrimSpace(keyword); s != "" {
		q = q.Where("text LIKE ?", "%"+s+"%")
	}
	if s := strings.TrimSpace(labelFilter); s != "" {
		q = q.Where("label = ?", s)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var pos []ModelSamplePO
	if err := q.Order("id ASC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	out := make([]classifier.Sample, len(pos))
	for i := range pos {
		out[i] = *PoToModelSample(&pos[i])
	}
	return out, total, nil
}

func (r *SampleRepository) ListLabels(ctx context.Context, modelID int64) ([]string, error) {
	var labels []string
	if err := r.db.WithContext(ctx).
		Model(&ModelSamplePO{}).
		Where("classify_model_id = ?", modelID).
		Select("DISTINCT label").
		Order("label ASC").
		Pluck("label", &labels).Error; err != nil {
		return nil, err
	}
	return labels, nil
}

func (r *SampleRepository) Create(ctx context.Context, samples []classifier.Sample) error {
	if len(samples) == 0 {
		return nil
	}
	pos := make([]ModelSamplePO, len(samples))
	for i := range samples {
		pos[i] = *ModelSampleToPO(&samples[i])
	}
	return r.db.WithContext(ctx).Create(&pos).Error
}

func (r *SampleRepository) Update(ctx context.Context, sample *classifier.Sample) error {
	po := ModelSampleToPO(sample)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *SampleRepository) Delete(ctx context.Context, modelID, sampleID int64) error {
	return r.db.WithContext(ctx).
		Where("classify_model_id = ? AND id = ?", modelID, sampleID).
		Delete(&ModelSamplePO{}).Error
}

func (r *SampleRepository) ExportTrainTxt(ctx context.Context, modelID int64) ([]byte, error) {
	var pos []ModelSamplePO
	if err := r.db.WithContext(ctx).
		Where("classify_model_id = ?", modelID).
		Order("id ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}
	var b strings.Builder
	for _, po := range pos {
		label := strings.TrimSpace(po.Label)
		text := strings.TrimSpace(po.Text)
		if label == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("__label__%s\t%s\n", label, text))
	}
	return []byte(b.String()), nil
}
