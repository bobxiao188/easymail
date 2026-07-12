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

package milter

import (
	"time"

	service "easymail/internal/app/filter"
	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/feature"
	"easymail/internal/infrastructure/filter/extractors"
	"easymail/internal/protocol/milter"
	"easymail/pkg/config"
	"easymail/pkg/logger/easylog"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const maxMilterBodyAccum = 50 << 20 // 50 MB

func coalesceStageMatchResult(res *service.MatchResult) *service.MatchResult {
	if res != nil {
		return res
	}
	return &service.MatchResult{Action: "accept", RuleID: 0, FeatureSnap: "{}", TraceJSON: "{}"}
}

type MilterHandler struct {
	bodyAccum   []byte
	eng         *service.RuleEngine
	db          *gorm.DB
	Log         *easylog.Logger
	fe          feature.FeatureEngine
	av          *service.AntivirusService
	fc          *filter.MilterContext
	Config      config.FilterConfig
	RedisClient *redis.Client
}

func NewMilterHandler(eng *service.RuleEngine, av *service.AntivirusService, db *gorm.DB, log *easylog.Logger, cfg config.FilterConfig, redisClient *redis.Client) *MilterHandler {
	return &MilterHandler{eng: eng, av: av, db: db, Log: log, fe: &extractors.FeatureEngine{}, Config: cfg, RedisClient: redisClient}
}

func NewMilterHandlerFactory(db *gorm.DB, cfg config.FilterConfig, log *easylog.Logger, redisClient *redis.Client) func() milter.Milter {
	eng := &service.RuleEngine{DB: db, Config: cfg}
	av := service.NewAntivirusServiceFromConfig(cfg)
	return func() milter.Milter {
		return NewMilterHandler(eng, av, db, log, cfg, redisClient)
	}
}

func (h *MilterHandler) infof(format string, args ...interface{}) {
	if h.Log != nil {
		h.Log.Infof(format, args...)
	}
}

func (h *MilterHandler) warnf(format string, args ...interface{}) {
	if h.Log != nil {
		h.Log.Warnf(format, args...)
	}
}

func (h *MilterHandler) stageTimeout(defaultTimeout time.Duration) time.Duration {
	if h.Config.StageTimeoutMs <= 0 {
		return defaultTimeout
	}
	return time.Duration(h.Config.StageTimeoutMs) * time.Millisecond
}
