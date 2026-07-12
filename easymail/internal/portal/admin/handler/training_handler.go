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

package handler

import (
	"errors"
	"strconv"

	"easymail/internal/app/filter"
	"easymail/internal/domain/filter/classifier"

	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StartTrainingRequest binds the ad-hoc training launch payload.
type StartTrainingRequest struct {
	ModelName      string                 `json:"modelName" binding:"required"`
	Algorithm      string                 `json:"algorithm" binding:"required"`
	Params         StartTrainingParams    `json:"params"`
	SampleMappings []StartTrainingMapping `json:"sampleMappings" binding:"required"`
}

// StartTrainingParams binds FastText hyperparameters.
type StartTrainingParams struct {
	LearningRate float64 `json:"learningRate"`
	Epoch        int     `json:"epoch"`
	WordNgrams   int     `json:"wordNgrams"`
	Dim          int     `json:"dim"`
	Loss         string  `json:"loss"`
}

// StartTrainingSourceGroup binds one source group (category + tags + limit).
type StartTrainingSourceGroup struct {
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	LimitType string   `json:"limitType"`
	LimitN    int      `json:"limitN"`
}

// StartTrainingMapping binds one target class -> source groups entry.
type StartTrainingMapping struct {
	TargetClass string                    `json:"targetClass" binding:"required"`
	Sources     []StartTrainingSourceGroup `json:"sources" binding:"required"`
}

// StartTrainingHandler launches an ad-hoc FastText training job.
func (h *Handler) StartTrainingHandler(c *gin.Context) {
	var req StartTrainingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	lr := req.Params.LearningRate
	ep := req.Params.Epoch
	wng := req.Params.WordNgrams
	dim := req.Params.Dim
	params := classifier.ModelParams{
		LearningRate: &lr,
		Epoch:        &ep,
		WordNgrams:   &wng,
		Dim:          &dim,
		Loss:         req.Params.Loss,
	}

	mappings := make([]filter.TargetClassMapping, 0, len(req.SampleMappings))
	for _, m := range req.SampleMappings {
		groups := make([]filter.SourceGroup, 0, len(m.Sources))
		for _, g := range m.Sources {
			groups = append(groups, filter.SourceGroup{
				Category:  g.Category,
				Tags:      g.Tags,
				LimitType: g.LimitType,
				LimitN:    g.LimitN,
			})
		}
		mappings = append(mappings, filter.TargetClassMapping{
			TargetClass: m.TargetClass,
			Sources:     groups,
		})
	}

	task, err := h.trainingService.StartTraining(c.Request.Context(), filter.TrainingRequest{
		ModelName:      req.ModelName,
		Algorithm:      req.Algorithm,
		Params:         params,
		SampleMappings: mappings,
	})
	if err != nil {
		response.Fail(c, messageTrainingOp(c, err))
		return
	}
	response.Success(c, task)
}

// GetTrainingHandler returns one training task and its log.
func (h *Handler) GetTrainingHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	task, err := h.trainingService.GetTraining(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundRecord))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	response.Success(c, task)
}
