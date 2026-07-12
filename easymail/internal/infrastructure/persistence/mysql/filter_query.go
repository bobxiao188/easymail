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
	"time"

	"easymail/internal/domain/filter/rule"

	"gorm.io/gorm"
)

// ListFeatureDefs returns built-in feature definitions (read-only).
func ListFeatureDefs(ctx context.Context, db *gorm.DB) ([]rule.BuiltinFeature, error) {
	var pos []BuiltinFeaturePO
	if err := db.WithContext(ctx).Model(&BuiltinFeaturePO{}).Order("feature_key").Find(&pos).Error; err != nil {
		return nil, err
	}
	rows := make([]rule.BuiltinFeature, len(pos))
	for i := range pos {
		rows[i] = *poToBuiltinFeature(&pos[i])
	}
	return rows, nil
}

// ListFeatureDefsPaged returns a page of built-in feature definitions (read-only).
func ListFeatureDefsPaged(ctx context.Context, db *gorm.DB, page, pageSize int) (total int64, rows []rule.BuiltinFeature, err error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	q := db.WithContext(ctx).Model(&BuiltinFeaturePO{})
	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset := (page - 1) * pageSize
	var pos []BuiltinFeaturePO
	if err = q.Order("feature_key").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return 0, nil, err
	}
	rows = make([]rule.BuiltinFeature, len(pos))
	for i := range pos {
		rows[i] = *poToBuiltinFeature(&pos[i])
	}
	return total, rows, nil
}

// ListCustomFeatureDefs lists custom feature definitions (admin).
func ListCustomFeatureDefs(ctx context.Context, db *gorm.DB) ([]rule.CustomFeature, error) {
	var pos []CustomFeaturePO
	if err := db.WithContext(ctx).Model(&CustomFeaturePO{}).Order("id DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	rows := make([]rule.CustomFeature, len(pos))
	for i := range pos {
		rows[i] = *poToCustomFeature(&pos[i])
	}
	return rows, nil
}

// ListCustomFeatureDefsPaged returns a page of custom feature definitions (admin).
func ListCustomFeatureDefsPaged(ctx context.Context, db *gorm.DB, page, pageSize int) (total int64, rows []rule.CustomFeature, err error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	q := db.WithContext(ctx).Model(&CustomFeaturePO{})
	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset := (page - 1) * pageSize
	var pos []CustomFeaturePO
	if err = q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return 0, nil, err
	}
	rows = make([]rule.CustomFeature, len(pos))
	for i := range pos {
		rows[i] = *poToCustomFeature(&pos[i])
	}
	return total, rows, nil
}

func GetCustomFeatureDef(ctx context.Context, db *gorm.DB, id int64) (*rule.CustomFeature, error) {
	var po CustomFeaturePO
	if err := db.WithContext(ctx).Model(&CustomFeaturePO{}).First(&po, id).Error; err != nil {
		return nil, err
	}
	return poToCustomFeature(&po), nil
}

func CreateCustomFeatureDef(ctx context.Context, db *gorm.DB, f *rule.CustomFeature) error {
	po := customFeatureToPO(f)
	return db.WithContext(ctx).Create(po).Error
}

func SaveCustomFeatureDef(ctx context.Context, db *gorm.DB, f *rule.CustomFeature) error {
	po := customFeatureToPO(f)
	return db.WithContext(ctx).Save(po).Error
}

func DeleteCustomFeatureDef(ctx context.Context, db *gorm.DB, id int64) error {
	return db.WithContext(ctx).Delete(&CustomFeaturePO{}, id).Error
}

// ListRules returns all filter rules (admin list).
func ListRules(ctx context.Context, db *gorm.DB) ([]rule.Rule, error) {
	var pos []RulePO
	if err := db.WithContext(ctx).Model(&RulePO{}).Order("priority DESC, id DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	rows := make([]rule.Rule, len(pos))
	for i := range pos {
		rows[i] = *poToRule(&pos[i])
	}
	return rows, nil
}

// ListRulesPaged returns a page of filter rules (admin list).
func ListRulesPaged(ctx context.Context, db *gorm.DB, page, pageSize int) (total int64, rows []rule.Rule, err error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	q := db.WithContext(ctx).Model(&RulePO{})
	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset := (page - 1) * pageSize
	var pos []RulePO
	if err = q.Order("priority DESC, id DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return 0, nil, err
	}
	rows = make([]rule.Rule, len(pos))
	for i := range pos {
		rows[i] = *poToRule(&pos[i])
	}
	return total, rows, nil
}

// GetRule .
func GetRule(ctx context.Context, db *gorm.DB, id int64) (*rule.Rule, error) {
	var po RulePO
	if err := db.WithContext(ctx).Model(&RulePO{}).First(&po, id).Error; err != nil {
		return nil, err
	}
	return poToRule(&po), nil
}

// CreateRule .
func CreateRule(ctx context.Context, db *gorm.DB, r *rule.Rule) error {
	po := ruleToPO(r)
	return db.WithContext(ctx).Create(po).Error
}

// SaveRule updates the full rule row.
func SaveRule(ctx context.Context, db *gorm.DB, r *rule.Rule) error {
	po := ruleToPO(r)
	return db.WithContext(ctx).Save(po).Error
}

// DeleteRule permanently deletes a rule by id.
func DeleteRule(ctx context.Context, db *gorm.DB, id int64) error {
	return db.WithContext(ctx).Delete(&RulePO{}, id).Error
}

// FilterLogFilter optional AND filters for filter_logs list (substring match on text fields).
type FilterLogFilter struct {
	IP        string
	Sender    string
	Recipient string
	// CreatedAfter / CreatedBefore are inclusive bounds on created_at when non-nil.
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

func applyFilterLogFilter(q *gorm.DB, f *FilterLogFilter) *gorm.DB {
	if f == nil {
		return q
	}
	if s := strings.TrimSpace(f.IP); s != "" {
		q = q.Where("INSTR(COALESCE(ip,''), ?) > 0", s)
	}
	if s := strings.TrimSpace(f.Sender); s != "" {
		q = q.Where("INSTR(COALESCE(sender,''), ?) > 0", s)
	}
	if s := strings.TrimSpace(f.Recipient); s != "" {
		q = q.Where("INSTR(COALESCE(recipient,''), ?) > 0", s)
	}
	if f.CreatedAfter != nil {
		q = q.Where("created_at >= ?", *f.CreatedAfter)
	}
	if f.CreatedBefore != nil {
		q = q.Where("created_at <= ?", *f.CreatedBefore)
	}
	return q
}

// ListFilterLogs returns a page of delivery log rows.
func ListFilterLogs(ctx context.Context, db *gorm.DB, page, pageSize int, filter *FilterLogFilter) (total int64, rows []rule.FilterLog, err error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	q := db.WithContext(ctx).Model(&FilterLogPO{})
	q = applyFilterLogFilter(q, filter)
	if err = q.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset := (page - 1) * pageSize
	q = db.WithContext(ctx).Model(&FilterLogPO{})
	q = applyFilterLogFilter(q, filter)
	var pos []FilterLogPO
	if err = q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return 0, nil, err
	}
	rows = make([]rule.FilterLog, len(pos))
	for i := range pos {
		rows[i] = *poToFilterLog(&pos[i])
	}
	return total, rows, nil
}

// GetFilterLog .
func GetFilterLog(ctx context.Context, db *gorm.DB, id int64) (*rule.FilterLog, error) {
	var po FilterLogPO
	if err := db.WithContext(ctx).Model(&FilterLogPO{}).First(&po, id).Error; err != nil {
		return nil, err
	}
	return poToFilterLog(&po), nil
}

// InsertFilterLog persists a log row; callers may ignore errors for async fire-and-forget writes.
func InsertFilterLog(ctx context.Context, db *gorm.DB, row *rule.FilterLog) error {
	if db == nil || row == nil {
		return nil
	}
	po := filterLogToPO(row)
	return db.WithContext(ctx).Create(po).Error
}
