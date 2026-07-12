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
	"net/http"
	"strconv"
	"strings"

	filtersvc "easymail/internal/app/filter"

	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"
	"easymail/pkg/types"

	"github.com/gin-gonic/gin"
)

// ListModelSamplesHandler lists training samples for one classify model.
func (h *Handler) ListModelSamplesHandler(c *gin.Context) {
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	var req types.PaginationRequest
	if err := req.FromGinContext(c); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	labelFilter := strings.TrimSpace(c.Query("label"))
	rows, total, err := h.spamModelService.ListModelSamples(c.Request.Context(), mid, req.Keyword, labelFilter, req.Page, req.PageSize)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	meta := types.NewPaginationMeta(req.Page, req.PageSize, total)
	response.SuccessWithPagination(c, rows, meta)
}

// ListModelSampleLabelsHandler returns distinct sample labels for one classify model (sorted).
func (h *Handler) ListModelSampleLabelsHandler(c *gin.Context) {
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	labels, err := h.spamModelService.ListModelSampleLabels(c.Request.Context(), mid)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	response.Success(c, labels)
}

// ExportModelSamplesTrainTxtHandler returns train.txt lines: __label__<class>\\t<text> (UTF-8), compatible with scripts/train/distiBERT/train.py.
func (h *Handler) ExportModelSamplesTrainTxtHandler(c *gin.Context) {
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	data, err := h.spamModelService.ExportModelSamplesTrainTxt(c.Request.Context(), mid)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	c.Header("Content-Disposition", `attachment; filename="train.txt"`)
	c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
}

type createModelSamplesRequest struct {
	Text  string `json:"text"`
	Label string `json:"label"`
	Items []struct {
		Text  string `json:"text"`
		Label string `json:"label"`
	} `json:"items"`
}

// CreateModelSamplesHandler inserts one sample (text+label) or a batch (items).
func (h *Handler) CreateModelSamplesHandler(c *gin.Context) {
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	var req createModelSamplesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	var inputs []filtersvc.ModelSampleInput
	if len(req.Items) > 0 {
		for _, it := range req.Items {
			inputs = append(inputs, filtersvc.ModelSampleInput{Text: it.Text, Label: it.Label})
		}
	} else {
		inputs = append(inputs, filtersvc.ModelSampleInput{Text: req.Text, Label: req.Label})
	}
	err = h.spamModelService.CreateModelSamples(c.Request.Context(), mid, inputs)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	response.Success(c, nil)
}

type updateModelSampleRequest struct {
	Text  string `json:"text" binding:"required"`
	Label string `json:"label" binding:"required"`
}

// UpdateModelSampleHandler updates one sample row.
func (h *Handler) UpdateModelSampleHandler(c *gin.Context) {
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	sid, err := strconv.ParseInt(c.Param("sampleId"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	var req updateModelSampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	err = h.spamModelService.UpdateModelSample(c.Request.Context(), mid, sid, req.Text, req.Label)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	response.Success(c, nil)
}

// DeleteModelSampleHandler removes one sample row.
func (h *Handler) DeleteModelSampleHandler(c *gin.Context) {
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	sid, err := strconv.ParseInt(c.Param("sampleId"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	err = h.spamModelService.DeleteModelSample(c.Request.Context(), mid, sid)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	response.Success(c, nil)
}

