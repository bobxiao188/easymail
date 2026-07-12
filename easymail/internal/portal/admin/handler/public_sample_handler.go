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
	"fmt"
	"strconv"

	"easymail/internal/infrastructure/persistence/mysql"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"
	"easymail/pkg/types"

	"github.com/gin-gonic/gin"
)

// ListSamplesHandler lists global training samples with pagination.
func (h *Handler) ListSamplesHandler(c *gin.Context) {
	var req types.PaginationRequest
	if err := req.FromGinContext(c); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	tag := c.Query("tag")
	rows, total, err := h.publicSampleService.ListSamples(c.Request.Context(), uint(categoryID), tag, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	meta := types.NewPaginationMeta(req.Page, req.PageSize, total)
	response.SuccessWithPagination(c, rows, meta)
}

// ListCategoriesHandler returns managed sample categories with pagination.
func (h *Handler) ListCategoriesHandler(c *gin.Context) {
	var req types.PaginationRequest
	if err := req.FromGinContext(c); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	categories, total, err := h.publicSampleCategoryService.ListCategories(c.Request.Context(), req.Keyword, req.Page, req.PageSize)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	meta := types.NewPaginationMeta(req.Page, req.PageSize, total)
	response.SuccessWithPagination(c, categories, meta)
}

// ListTagsHandler returns distinct sample tags, optionally filtered by category.
func (h *Handler) ListTagsHandler(c *gin.Context) {
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	tags, err := h.publicSampleService.ListTags(c.Request.Context(), uint(categoryID))
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, tags)
}

type createSampleRequest struct {
	CategoryID uint   `json:"categoryId" binding:"required"`
	Tag        string `json:"tag" binding:"required"`
	Text       string `json:"text" binding:"required"`
}

// CreateSampleHandler inserts one sample or a batch.
func (h *Handler) CreateSampleHandler(c *gin.Context) {
	var req gin.H
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	// Check for batch mode
	if itemsRaw, ok := req["items"]; ok {
		itemsList, ok := itemsRaw.([]interface{})
		if !ok || len(itemsList) == 0 {
			response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
			return
		}
		poList := make([]mysql.PublicSamplePO, 0, len(itemsList))
		for _, item := range itemsList {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
				return
			}
			categoryID := uint(toInt64(itemMap["categoryId"]))
			poList = append(poList, mysql.PublicSamplePO{
				CategoryID: categoryID,
				Tag:        toString(itemMap["tag"]),
				Text:       toString(itemMap["text"]),
			})
		}
		err := h.publicSampleService.CreateSamplesBatch(c.Request.Context(), poList)
		if err != nil {
			response.Fail(c, messagePublicSampleOp(c, err))
			return
		}
		response.Success(c, nil)
		return
	}

	// Single sample mode
	categoryID := uint(toInt64(req["categoryId"]))
	tag, _ := req["tag"].(string)
	text, _ := req["text"].(string)
	if categoryID == 0 || tag == "" || text == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	err := h.publicSampleService.CreateSample(c.Request.Context(), categoryID, tag, text)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, nil)
}

type updateSampleRequest struct {
	CategoryID uint   `json:"categoryId"`
	Tag        string `json:"tag"`
	Text       string `json:"text"`
}

// UpdateSampleHandler updates one sample row.
func (h *Handler) UpdateSampleHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	var req updateSampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	err = h.publicSampleService.UpdateSample(c.Request.Context(), uint(id), req.CategoryID, req.Tag, req.Text)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, nil)
}

// ========== Category Management Handlers ==========

// ListCategoriesHandler is already defined above for samples.
// These are additional category management handlers.

type createCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateCategoryHandler creates a new sample category.
func (h *Handler) CreateCategoryHandler(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	row, err := h.publicSampleCategoryService.CreateCategory(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, row)
}

type updateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateCategoryHandler updates a sample category.
func (h *Handler) UpdateCategoryHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	row, err := h.publicSampleCategoryService.UpdateCategory(c.Request.Context(), uint(id), req.Name, req.Description)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, row)
}

// DeleteCategoryHandler deletes a sample category.
func (h *Handler) DeleteCategoryHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	err = h.publicSampleCategoryService.DeleteCategory(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, nil)
}

// GetCategoryHandler gets a single category by ID.
func (h *Handler) GetCategoryHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	row, err := h.publicSampleCategoryService.GetCategory(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, row)
}

// Helper function to convert interface{} to int64
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case string:
		var n int64
		_, _ = fmt.Sscanf(val, "%d", &n)
		return n
	}
	return 0
}

// DeleteSampleHandler removes one sample row.
func (h *Handler) DeleteSampleHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	err = h.publicSampleService.DeleteSample(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, nil)
}

// DescribeSamplesHandler returns sample counts grouped by category and tag.
func (h *Handler) DescribeSamplesHandler(c *gin.Context) {
	categoryID, _ := strconv.ParseUint(c.Query("categoryId"), 10, 64)
	rows, err := h.publicSampleService.DescribeSamples(c.Request.Context(), uint(categoryID))
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, rows)
}

type batchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

// BatchDeleteSamplesHandler removes multiple sample rows.
func (h *Handler) BatchDeleteSamplesHandler(c *gin.Context) {
	var req batchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if len(req.IDs) == 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	err := h.publicSampleService.DeleteSamplesBatch(c.Request.Context(), req.IDs)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, nil)
}

type batchUpdateRequest struct {
	IDs        []uint `json:"ids" binding:"required"`
	CategoryID uint   `json:"categoryId" binding:"required"`
	Tag        string `json:"tag" binding:"required"`
}

// BatchUpdateSamplesHandler updates multiple sample rows with new category and tag.
func (h *Handler) BatchUpdateSamplesHandler(c *gin.Context) {
	var req batchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if len(req.IDs) == 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	err := h.publicSampleService.UpdateSamplesBatch(c.Request.Context(), req.IDs, req.CategoryID, req.Tag)
	if err != nil {
		response.Fail(c, messagePublicSampleOp(c, err))
		return
	}
	response.Success(c, nil)
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return v.(string)
}
