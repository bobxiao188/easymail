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

package dbsink

import (
	"context"
	"encoding/json"
	"fmt"

	"easymail/pkg/database"
	"easymail/pkg/tracer"

	"gorm.io/gorm"
)

type Config struct {
	Enabled bool
}

type Sink struct {
	db *gorm.DB
}

func New(cfg Config) (*Sink, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if err := db.AutoMigrate(&SessionEvent{}); err != nil {
		return nil, err
	}
	return &Sink{db: db}, nil
}

func (s *Sink) Write(ctx context.Context, e tracer.Event) error {
	if s == nil || s.db == nil {
		return nil
	}
	fields, _ := json.Marshal(e.Fields)
	tags, _ := json.Marshal(e.Tags)
	row := &SessionEvent{
		TS:        e.Timestamp,
		SessionID: e.SessionID,
		Protocol:  string(e.Protocol),
		Stage:     e.Stage,
		Remote:    e.Remote,
		Local:     e.Local,
		Duration:  int64(e.Duration),
		Err:       e.Err,
		Fields:    string(fields),
		Tags:      string(tags),
	}
	return s.db.WithContext(ctx).Create(row).Error
}

func (s *Sink) Close() error { return nil }
