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
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/cache"

	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"
	"easymail/pkg/types"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateClassifyModelRequest binds JSON for creating a spam classify ML model definition.
type CreateClassifyModelRequest struct {
	Name          string          `json:"name" binding:"required"`
	Algorithm     string          `json:"algorithm" binding:"required"`
	Tokenizer     string          `json:"tokenizer" binding:"required"`
	Languages     []string        `json:"languages" binding:"required"`
	SavePath      string          `json:"savePath"`
	MaxTextLength int             `json:"maxTextLength" binding:"required"`
	EmailFields   []string        `json:"emailFields" binding:"required"`
	Params        json.RawMessage `json:"params"`
}

// PredictClassifyModelRequest binds JSON for admin try-predict.
type PredictClassifyModelRequest struct {
	Text          string   `json:"text"`
	LanguageCodes []string `json:"languageCodes"`
}

// UpdateClassifyModelRequest binds JSON for updating a classify model.
type UpdateClassifyModelRequest struct {
	Name          string          `json:"name" binding:"required"`
	Algorithm     string          `json:"algorithm" binding:"required"`
	Tokenizer     string          `json:"tokenizer" binding:"required"`
	Languages     []string        `json:"languages" binding:"required"`
	SavePath      string          `json:"savePath"`
	MaxTextLength int             `json:"maxTextLength" binding:"required"`
	EmailFields   []string        `json:"emailFields" binding:"required"`
	Enabled       bool            `json:"enabled"`
	Params        json.RawMessage `json:"params"`
}

// ListClassifyModelsHandler returns paginated classify model definitions.
func (h *Handler) ListClassifyModelsHandler(c *gin.Context) {
	var req types.PaginationRequest
	if err := req.FromGinContext(c); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	algorithm := c.Query("algorithm")
	statusStr := c.Query("status")
	var status *int
	if statusStr != "" {
		statusInt, err := strconv.Atoi(statusStr)
		if err == nil {
			status = &statusInt
		}
	}

	models, total, err := h.spamModelService.List(c.Request.Context(), req.Keyword, algorithm, status, req.Page, req.PageSize)
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}

	meta := types.NewPaginationMeta(req.Page, req.PageSize, total)
	response.SuccessWithPagination(c, models, meta)
}

// GetClassifyModelHandler returns one classify model by ID.
func (h *Handler) GetClassifyModelHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	model, err := h.spamModelService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundClassifyModel))
		return
	}

	response.Success(c, model)
}

// CreateClassifyModelHandler persists a new classify model.
func (h *Handler) CreateClassifyModelHandler(c *gin.Context) {
	if strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(512 << 20); err != nil {
			response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartParseFailed))
			return
		}
		algo := strings.TrimSpace(c.PostForm("algorithm"))
		if strings.EqualFold(algo, string(classifier.AlgorithmDistilBERT)) {
			name, tokenizer, params, maxLen, langs, emails, onnx, ok := bindDistilBERTMultipartCreate(c)
			if !ok {
				return
			}
			err := h.spamModelService.CreateDistilBERTWithONNXFile(c.Request.Context(), name, tokenizer, langs, maxLen, emails, params, onnx)
			if err != nil {
				response.Fail(c, messageSpamModelOp(c, err))
				return
			}
			cache.InvalidateClassifyModelsCache()
			response.Success(c, nil)
			return
		}
	}

	var req CreateClassifyModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	algo := strings.TrimSpace(req.Algorithm)
	if strings.EqualFold(algo, string(classifier.AlgorithmDistilBERT)) {
		response.Fail(c, appi18n.Message(c, appi18n.KeyErrClassifyModelDistilBERTUseMultipart))
		return
	}
	if !strings.EqualFold(algo, string(classifier.AlgorithmFastText)) && strings.TrimSpace(req.SavePath) == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelSavePathRequired))
		return
	}
	paramsStr := ""
	if len(req.Params) > 0 {
		paramsStr = string(req.Params)
	}

	err := h.spamModelService.Create(c.Request.Context(), req.Name, req.Algorithm, req.Tokenizer, req.Languages, req.SavePath, req.MaxTextLength, req.EmailFields, paramsStr)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	cache.InvalidateClassifyModelsCache()

	response.Success(c, nil)
}

// UpdateClassifyModelHandler updates an existing classify model.
func (h *Handler) UpdateClassifyModelHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	if strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(512 << 20); err != nil {
			response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartParseFailed))
			return
		}
		algo := strings.TrimSpace(c.PostForm("algorithm"))
		if strings.EqualFold(algo, string(classifier.AlgorithmDistilBERT)) {
			name, tokenizer, params, maxLen, langs, emails, enabled, onnx, ok := bindDistilBERTMultipartUpdate(c)
			if !ok {
				return
			}
			err := h.spamModelService.UpdateDistilBERTWithOptionalONNXFile(c.Request.Context(), id, name, tokenizer, langs, maxLen, emails, enabled, params, onnx)
			if err != nil {
				response.Fail(c, messageSpamModelOp(c, err))
				return
			}
			cache.InvalidateClassifyModelsCache()
			response.Success(c, nil)
			return
		}
	}

	var req UpdateClassifyModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	algo := strings.TrimSpace(req.Algorithm)
	if !strings.EqualFold(algo, string(classifier.AlgorithmFastText)) && strings.TrimSpace(req.SavePath) == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelSavePathRequired))
		return
	}

	paramsStr := ""
	if len(req.Params) > 0 {
		paramsStr = string(req.Params)
	}

	err = h.spamModelService.Update(c.Request.Context(), id, req.Name, req.Algorithm, req.Tokenizer, req.Languages, req.SavePath, req.MaxTextLength, req.EmailFields, req.Enabled, paramsStr)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	cache.InvalidateClassifyModelsCache()

	response.Success(c, nil)
}

func bindDistilBERTMultipartCreate(c *gin.Context) (name, tokenizer, params string, maxLen int, langs classifier.Languages, emails classifier.EmailFields, onnx *multipart.FileHeader, ok bool) {
	name = strings.TrimSpace(c.PostForm("name"))
	tokenizer = strings.TrimSpace(c.PostForm("tokenizer"))
	params = strings.TrimSpace(c.PostForm("params"))
	if name == "" || tokenizer == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartNameTokenizer))
		return "", "", "", 0, nil, nil, nil, false
	}
	maxStr := strings.TrimSpace(c.PostForm("maxTextLength"))
	var err error
	maxLen, err = strconv.Atoi(maxStr)
	if err != nil || maxLen < 10 || maxLen > 512 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelDistilBERTMaxTextLen))
		return "", "", "", 0, nil, nil, nil, false
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(c.PostForm("languages"))), &langs); err != nil || len(langs) == 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartLanguages))
		return "", "", "", 0, nil, nil, nil, false
	}
	var ef []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(c.PostForm("emailFields"))), &ef); err != nil || len(ef) == 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartEmailFields))
		return "", "", "", 0, nil, nil, nil, false
	}
	emails = ef
	fh, err := c.FormFile("onnx")
	if err != nil || fh == nil {
		response.Fail(c, appi18n.Message(c, appi18n.KeyErrClassifyModelOnnxRequired))
		return "", "", "", 0, nil, nil, nil, false
	}
	return name, tokenizer, params, maxLen, langs, emails, fh, true
}

func bindDistilBERTMultipartUpdate(c *gin.Context) (name, tokenizer, params string, maxLen int, langs classifier.Languages, emails classifier.EmailFields, enabled bool, onnx *multipart.FileHeader, ok bool) {
	name = strings.TrimSpace(c.PostForm("name"))
	tokenizer = strings.TrimSpace(c.PostForm("tokenizer"))
	params = strings.TrimSpace(c.PostForm("params"))
	if name == "" || tokenizer == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartNameTokenizer))
		return "", "", "", 0, nil, nil, false, nil, false
	}
	maxStr := strings.TrimSpace(c.PostForm("maxTextLength"))
	maxLen, err := strconv.Atoi(maxStr)
	if err != nil || maxLen < 10 || maxLen > 512 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelDistilBERTMaxTextLen))
		return "", "", "", 0, nil, nil, false, nil, false
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(c.PostForm("languages"))), &langs); err != nil || len(langs) == 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartLanguages))
		return "", "", "", 0, nil, nil, false, nil, false
	}
	var ef []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(c.PostForm("emailFields"))), &ef); err != nil || len(ef) == 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrClassifyModelMultipartEmailFields))
		return "", "", "", 0, nil, nil, false, nil, false
	}
	emails = ef
	enabled = strings.EqualFold(strings.TrimSpace(c.PostForm("enabled")), "true")
	if fh, err := c.FormFile("onnx"); err == nil && fh != nil {
		onnx = fh
	}
	return name, tokenizer, params, maxLen, langs, emails, enabled, onnx, true
}

// DeleteClassifyModelHandler removes a classify model.
func (h *Handler) DeleteClassifyModelHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err = h.spamModelService.Delete(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	cache.InvalidateClassifyModelsCache()

	response.Success(c, nil)
}

// PredictClassifyModelHandler runs one inference for a classify model (real predictor path).
func (h *Handler) PredictClassifyModelHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	var req PredictClassifyModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	out, err := h.spamModelService.Predict(c.Request.Context(), id, req.Text, req.LanguageCodes)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundClassifyModel))
			return
		}
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}

	response.Success(c, out)
}

// StartClassifyModelTrainHandler starts FastText supervised training in the background.
func (h *Handler) StartClassifyModelTrainHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err = h.spamModelService.StartFastTextTraining(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}
	cache.InvalidateClassifyModelsCache()

	response.Success(c, nil)
}

// ExportClassifyModelHandler exports a model as a zip archive (model.conf + binary).
func (h *Handler) ExportClassifyModelHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	data, filename, err := h.spamModelService.ExportModel(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/zip", data)
}

// ImportClassifyModelHandler imports a model from a zip archive upload.
func (h *Handler) ImportClassifyModelHandler(c *gin.Context) {
	zipFile, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	algorithm := c.PostForm("algorithm")

	if err := h.spamModelService.ImportModel(c.Request.Context(), zipFile, algorithm); err != nil {
		response.Fail(c, messageSpamModelOp(c, err))
		return
	}

	cache.InvalidateClassifyModelsCache()
	response.Success(c, nil)
}
