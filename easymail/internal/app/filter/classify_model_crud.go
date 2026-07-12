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

// Classify model persistence: CRUD, activation enrichment, training samples; class_labels sync on FastText train success only.
// FastText training jobs live in classify_model_train.go; validation/errors in classify_model_validate.go.

package filter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/filter/assets"
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

type classifyModelService struct {
	db             *gorm.DB
	fastTextExe    string
	modelRoot      string
	onnxRuntimeLib string
	trainRunning   sync.Map // model ID -> placeholder while a train job is in flight
}

func NewClassifyModelService(db *gorm.DB, fastTextExecutable, modelRoot, onnxRuntimeLib string) ClassifyModelService {
	return &classifyModelService{
		db:             db,
		fastTextExe:    strings.TrimSpace(fastTextExecutable),
		modelRoot:      strings.TrimSpace(modelRoot),
		onnxRuntimeLib: strings.TrimSpace(onnxRuntimeLib),
	}
}

func (s *classifyModelService) List(ctx context.Context, keyword, algorithm string, status *int, page, pageSize int) ([]classifier.Model, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	query := s.db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).Where("is_deleted = ?", false)

	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	if algorithm != "" {
		query = query.Where("algorithm = ?", algorithm)
	}

	if status != nil {
		enabled := *status == 1
		query = query.Where("enabled = ?", enabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []mysql.ClassifyModelPO
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	models := make([]classifier.Model, len(pos))
	for i := range pos {
		models[i] = *mysql.PoToClassifyModel(&pos[i])
	}

	s.enrichActivation(ctx, models)
	return models, total, nil
}

func (s *classifyModelService) GetByID(ctx context.Context, id int64) (*classifier.Model, error) {
	m, err := s.loadClassifyModelByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichActivationOne(ctx, m)
	return m, nil
}

func (s *classifyModelService) Create(ctx context.Context, name, algorithm, tokenizer string, languages classifier.Languages, savePath string, maxTextLength int, emailFields classifier.EmailFields, params string) error {

	if err := ValidateClassifyModelDisplayName(ctx, s.db, name, 0); err != nil {
		return err
	}

	model := classifier.Model{
		Name:          name,
		Algorithm:     classifier.Algorithm(algorithm),
		Tokenizer:     classifier.Tokenizer(tokenizer),
		Languages:     languages,
		SavePath:      savePath,
		MaxTextLength: maxTextLength,
		EmailFields:   emailFields,
		Enabled:       false,
		IsDeleted:     false,
		TrainStatus:   classifier.TrainStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if params != "" {
		var modelParams classifier.ModelParams
		if err := json.Unmarshal([]byte(params), &modelParams); err == nil {
			model.Params = modelParams
		}
	}

	po := mysql.ClassifyModelToPO(&model)
	if err := s.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	return nil
}

func (s *classifyModelService) Update(ctx context.Context, id int64, name, algorithm, tokenizer string, languages classifier.Languages, savePath string, maxTextLength int, emailFields classifier.EmailFields, enabled bool, params string) error {
	model, err := s.loadClassifyModelByID(ctx, id)
	if err != nil {
		return err
	}

	model.Name = name
	model.Algorithm = classifier.Algorithm(algorithm)
	model.Tokenizer = classifier.Tokenizer(tokenizer)
	model.Languages = languages
	model.SavePath = savePath
	model.MaxTextLength = maxTextLength
	model.EmailFields = emailFields
	model.Enabled = enabled
	model.UpdatedAt = time.Now()

	if params != "" {
		var modelParams classifier.ModelParams
		if err := json.Unmarshal([]byte(params), &modelParams); err == nil {
			model.Params = modelParams
		}
	}

	if model.Enabled && !classifyModelAssetsReady(model) {
		return ErrClassifyModelActivationNotReady
	}

	if err := ValidateClassifyModelDisplayName(ctx, s.db, name, model.ID); err != nil {
		return err
	}

	po := mysql.ClassifyModelToPO(model)
	return s.db.WithContext(ctx).Save(po).Error
}

func (s *classifyModelService) Delete(ctx context.Context, id int64) error {
	model, err := s.loadClassifyModelByID(ctx, id)
	if err != nil {
		return err
	}

	model.IsDeleted = true
	model.DeleteTime = time.Now()
	model.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("classify_model_id = ?", model.ID).Delete(&mysql.ModelSamplePO{}).Error; err != nil {
			return err
		}
		return tx.Model(&mysql.ClassifyModelPO{}).Where("id = ?", model.ID).Updates(map[string]interface{}{
			"is_deleted":  true,
			"delete_time": time.Now(),
			"updated_at":  time.Now(),
		}).Error
	}); err != nil {
		return err
	}

	savePath := strings.TrimSpace(model.SavePath)
	cache.InvalidateClassifyModelsCache()
	cache.InvalidateClassifyModelsCache()
	if err := assets.RemoveClassifyModelSavePath(savePath); err != nil {
		return fmt.Errorf("%w: %v", ErrClassifyModelRemoveFiles, err)
	}
	return nil
}

func (s *classifyModelService) loadClassifyModelByID(ctx context.Context, id int64) (*classifier.Model, error) {
	var po mysql.ClassifyModelPO
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, false).First(&po).Error; err != nil {
		return nil, err
	}
	return mysql.PoToClassifyModel(&po), nil
}

// enrichActivation sets ActivationReady and turns off enabled when assets are missing (persisted).
func (s *classifyModelService) enrichActivation(ctx context.Context, list []classifier.Model) {
	var fixed bool
	for i := range list {
		list[i].ActivationReady = classifyModelAssetsReady(&list[i])
		if list[i].Enabled && !list[i].ActivationReady {
			res := s.db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).
				Where("id = ? AND is_deleted = ?", list[i].ID, false).
				Update("enabled", false)
			if res.Error == nil && res.RowsAffected > 0 {
				list[i].Enabled = false
				fixed = true
			}
		}
	}
	if fixed {
		cache.InvalidateClassifyModelsCache()
	}
}

func (s *classifyModelService) enrichActivationOne(ctx context.Context, m *classifier.Model) {
	m.ActivationReady = classifyModelAssetsReady(m)
	if m.Enabled && !m.ActivationReady {
		res := s.db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).
			Where("id = ? AND is_deleted = ?", m.ID, false).
			Update("enabled", false)
		if res.Error == nil && res.RowsAffected > 0 {
			m.Enabled = false
			cache.InvalidateClassifyModelsCache()
		}
	}
}

type distinctLabelRow struct {
	Label string `gorm:"column:label"`
}

// recomputeAndPersistClassLabels sets ClassifyModel.class_labels from distinct sample labels (ordered).
// Called after FastText training succeeds (not on every sample edit).
// When invalidate is true, clears the classify-models cache so milter / admin see the update immediately.
func (s *classifyModelService) recomputeAndPersistClassLabels(ctx context.Context, db *gorm.DB, modelID uint, invalidate bool) error {
	var rows []distinctLabelRow
	if err := db.WithContext(ctx).Model(&mysql.ModelSamplePO{}).
		Select("label").
		Where("classify_model_id = ?", modelID).
		Group("label").
		Order("label ASC").
		Find(&rows).Error; err != nil {
		return err
	}
	labels := make(classifier.ClassLabels, 0, len(rows))
	for _, r := range rows {
		t := strings.TrimSpace(r.Label)
		if t != "" {
			labels = append(labels, t)
		}
	}
	labelsJSON, _ := json.Marshal(labels)
	if err := db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).
		Where("id = ? AND is_deleted = ?", modelID, false).
		Update("class_labels", string(labelsJSON)).Error; err != nil {
		return err
	}
	if invalidate {
		cache.InvalidateClassifyModelsCache()
	}
	return nil
}

const (
	maxModelSampleTextRunes = 256 * 1024
	maxModelSampleBatch     = 200
)

func modelSampleRowsForModel(modelID uint, items []ModelSampleInput) ([]mysql.ModelSamplePO, error) {
	rows := make([]mysql.ModelSamplePO, 0, len(items))
	for _, it := range items {
		t, l, err := normalizeModelSample(it.Text, it.Label)
		if err != nil {
			return nil, err
		}
		rows = append(rows, mysql.ModelSamplePO{
			ClassifyModelID: modelID,
			Text:            t,
			Label:           l,
		})
	}
	return rows, nil
}

func normalizeModelSample(text, label string) (string, string, error) {
	t := strings.TrimSpace(text)
	l := strings.TrimSpace(label)
	if t == "" || l == "" {
		return "", "", ErrModelSampleInvalid
	}
	if len([]rune(t)) > maxModelSampleTextRunes {
		return "", "", ErrModelSampleInvalid
	}
	return t, l, nil
}

func (s *classifyModelService) ListModelSamples(ctx context.Context, classifyModelID int64, keyword, labelFilter string, page, pageSize int) ([]classifier.Sample, int64, error) {
	if _, err := s.GetByID(ctx, classifyModelID); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	q := s.db.WithContext(ctx).Model(&mysql.ModelSamplePO{}).Where("classify_model_id = ?", uint(classifyModelID))
	if lf := strings.TrimSpace(labelFilter); lf != "" {
		q = q.Where("label = ?", lf)
	}
	kw := strings.TrimSpace(keyword)
	if kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(text LIKE ? OR label LIKE ?)", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var samplePos []mysql.ModelSamplePO
	if err := q.Order("id ASC").Offset(offset).Limit(pageSize).Find(&samplePos).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]classifier.Sample, len(samplePos))
	for i := range samplePos {
		rows[i] = *mysql.PoToModelSample(&samplePos[i])
	}
	return rows, total, nil
}

func (s *classifyModelService) CreateModelSamples(ctx context.Context, classifyModelID int64, items []ModelSampleInput) error {
	if len(items) == 0 {
		return ErrModelSampleInvalid
	}
	if len(items) > maxModelSampleBatch {
		return ErrModelSampleBatchTooLarge
	}
	if _, err := s.GetByID(ctx, classifyModelID); err != nil {
		return err
	}
	rows, err := modelSampleRowsForModel(uint(classifyModelID), items)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return err
	}
	return nil
}

func (s *classifyModelService) UpdateModelSample(ctx context.Context, classifyModelID, sampleID int64, text, label string) error {
	if _, err := s.GetByID(ctx, classifyModelID); err != nil {
		return err
	}
	t, l, err := normalizeModelSample(text, label)
	if err != nil {
		return err
	}
	var po mysql.ModelSamplePO
	res := s.db.WithContext(ctx).Where("id = ? AND classify_model_id = ?", uint(sampleID), uint(classifyModelID)).First(&po)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return ErrModelSampleNotFound
		}
		return res.Error
	}
	po.Text = t
	po.Label = l
	if err := s.db.WithContext(ctx).Save(&po).Error; err != nil {
		return err
	}
	return nil
}

func (s *classifyModelService) DeleteModelSample(ctx context.Context, classifyModelID, sampleID int64) error {
	if _, err := s.GetByID(ctx, classifyModelID); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Model(&mysql.ModelSamplePO{}).Where("id = ? AND classify_model_id = ?", uint(sampleID), uint(classifyModelID)).Delete(&mysql.ModelSamplePO{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrModelSampleNotFound
	}
	return nil
}

func sanitizeTrainExportLabel(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return strings.TrimSpace(s)
}

func sanitizeTrainExportText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func (s *classifyModelService) ListModelSampleLabels(ctx context.Context, classifyModelID int64) ([]string, error) {
	if _, err := s.GetByID(ctx, classifyModelID); err != nil {
		return nil, err
	}
	var labels []string
	err := s.db.WithContext(ctx).Model(&mysql.ModelSamplePO{}).
		Where("classify_model_id = ?", uint(classifyModelID)).
		Distinct("label").
		Order("label ASC").
		Pluck("label", &labels).Error
	if err != nil {
		return nil, err
	}
	return labels, nil
}

func (s *classifyModelService) ExportModelSamplesTrainTxt(ctx context.Context, classifyModelID int64) ([]byte, error) {
	if _, err := s.GetByID(ctx, classifyModelID); err != nil {
		return nil, err
	}
	var pos []mysql.ModelSamplePO
	if err := s.db.WithContext(ctx).Model(&mysql.ModelSamplePO{}).Where("classify_model_id = ?", uint(classifyModelID)).Order("id ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	var b strings.Builder
	for _, po := range pos {
		s := mysql.PoToModelSample(&po)
		lab := sanitizeTrainExportLabel(s.Label)
		body := sanitizeTrainExportText(s.Text)
		if lab == "" || body == "" {
			continue
		}
		b.WriteString("__label__")
		b.WriteString(lab)
		b.WriteByte('\t')
		b.WriteString(body)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}
