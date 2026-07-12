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

	"easymail/internal/portal/webmail/middleware"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

type createLabelReq struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color" binding:"required"`
}

type updateLabelReq struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color" binding:"required"`
}

type setEmailLabelsReq struct {
	LabelIDs []int64 `json:"labelIds" binding:"required"`
}

// ListLabels returns all labels for the current user.
func (h *Handler) ListLabels(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	labels, err := h.mailService.ListLabels(c.Request.Context(), aid)
	if err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}
	response.Success(c, labels)
}

// CreateLabel creates a new label.
func (h *Handler) CreateLabel(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	var req createLabelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyLabelInvalidRequest))
		return
	}
	label, err := h.mailService.CreateLabel(c.Request.Context(), aid, req.Name, req.Color)
	if err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}
	response.Success(c, label)
}

// UpdateLabel updates an existing label.
func (h *Handler) UpdateLabel(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyLabelInvalidLabelID))
		return
	}
	var req updateLabelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyLabelInvalidRequest))
		return
	}
	if err := h.mailService.UpdateLabel(c.Request.Context(), aid, id, req.Name, req.Color); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": -1, "message": err.Error()})
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// DeleteLabel removes a label.
func (h *Handler) DeleteLabel(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyLabelInvalidLabelID))
		return
	}
	if err := h.mailService.DeleteLabel(c.Request.Context(), aid, id); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": -1, "message": err.Error()})
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// SetEmailLabels sets labels for a specific email.
func (h *Handler) SetEmailLabels(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyLabelInvalidEmailID))
		return
	}
	var req setEmailLabelsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyLabelInvalidRequest))
		return
	}
	if err := h.mailService.SetEmailLabels(c.Request.Context(), aid, mid, req.LabelIDs); err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}
	labels, _ := h.mailService.GetEmailLabels(c.Request.Context(), aid, mid)
	response.Success(c, labels)
}

// GetEmailLabels returns labels for a specific email.
func (h *Handler) GetEmailLabels(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyLabelInvalidEmailID))
		return
	}
	labels, err := h.mailService.GetEmailLabels(c.Request.Context(), aid, mid)
	if err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}
	response.Success(c, labels)
}
