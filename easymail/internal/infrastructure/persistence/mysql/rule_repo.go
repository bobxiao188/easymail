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

// RuleRepository implements rule.RuleRepository using MySQL.
type RuleRepository struct {
	db *gorm.DB
}

func NewRuleRepository(db *gorm.DB) *RuleRepository {
	return &RuleRepository{db: db}
}

func (r *RuleRepository) List(ctx context.Context) ([]rule.Rule, error) {
	return ListRules(ctx, r.db)
}

func (r *RuleRepository) ListPaged(ctx context.Context, page, pageSize int) (int64, []rule.Rule, error) {
	return ListRulesPaged(ctx, r.db, page, pageSize)
}

func (r *RuleRepository) GetByID(ctx context.Context, id int64) (*rule.Rule, error) {
	return GetRule(ctx, r.db, id)
}

func (r *RuleRepository) Create(ctx context.Context, rl *rule.Rule) error {
	return CreateRule(ctx, r.db, rl)
}

func (r *RuleRepository) Update(ctx context.Context, rl *rule.Rule) error {
	return SaveRule(ctx, r.db, rl)
}

func (r *RuleRepository) Delete(ctx context.Context, id int64) error {
	return DeleteRule(ctx, r.db, id)
}
