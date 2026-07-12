/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
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
	"easymail/internal/domain/shared"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"
	"easymail/pkg/types"

	"github.com/gin-gonic/gin"
)

// CreateDomainRequest binds JSON for creating a mail domain.
type CreateDomainRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateDomainRequest binds JSON for updating a mail domain.
type UpdateDomainRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      *bool  `json:"active"`
	IsDeleted   *bool  `json:"isDeleted"`
}

// DKIMSettingsRequest binds JSON for updating DKIM settings of a domain.
type DKIMSettingsRequest struct {
	Enabled      bool   `json:"enabled"`
	Selector     string `json:"selector"`
	PrivateKey   string `json:"privateKey"`
}

// ListDomainsRequest binds query parameters for domain listing.
type ListDomainsRequest struct {
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
	Keyword        string `form:"keyword"`
	IncludeDeleted bool   `form:"include_deleted"`
}

// ListDomainsHandler returns paginated domains.
func (h *Handler) ListDomainsHandler(c *gin.Context) {
	var req ListDomainsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	// Validate pagination parameters
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	mailDomainList, total, err := h.mailDomainService.List(c.Request.Context(), req.Keyword, req.Page, req.PageSize, req.IncludeDeleted)
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}

	meta := types.NewPaginationMeta(req.Page, req.PageSize, total)
	response.SuccessWithPagination(c, mailDomainList, meta)
}

// GetDomainHandler returns a single domain by ID.
func (h *Handler) GetDomainHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	mailDomain, err := h.mailDomainService.GetByID(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundDomain))
		return
	}

	response.Success(c, mailDomain)
}

// CreateDomainHandler creates a new domain.
func (h *Handler) CreateDomainHandler(c *gin.Context) {
	var req CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	_, err := h.mailDomainService.Create(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		h.log.Errorf("create domain failed: %v", err)
		response.Fail(c, messageMailDomainOp(c, err))
		return
	}

	response.Success(c, nil)
}

// UpdateDomainHandler updates domain metadata and active flag.
func (h *Handler) UpdateDomainHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	var req UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	// Use pointer values for partial update
	name := req.Name
	description := req.Description
	active := req.Active
	isDeleted := req.IsDeleted

	err := h.mailDomainService.UpdateWithFields(c.Request.Context(), shared.GlobalID(idStr), &name, &description, active, isDeleted)
	if err != nil {
		h.log.Errorf("update domain failed: %v", err)
		response.Fail(c, messageMailDomainOp(c, err))
		return
	}

	response.Success(c, nil)
}

// DeleteDomainHandler soft-deletes a domain.
func (h *Handler) DeleteDomainHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err := h.mailDomainService.Delete(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		h.log.Errorf("delete domain failed: %v", err)
		response.Fail(c, messageMailDomainOp(c, err))
		return
	}

	response.Success(c, nil)
}

// ToggleDomainActiveHandler toggles the domain active flag.
func (h *Handler) ToggleDomainActiveHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err := h.mailDomainService.ToggleActive(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		h.log.Errorf("toggle domain active failed: %v", err)
		response.Fail(c, messageMailDomainOp(c, err))
		return
	}

	response.Success(c, nil)
}

// UpdateDKIMSettingsHandler updates DKIM signing settings for a domain.
func (h *Handler) UpdateDKIMSettingsHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	var req DKIMSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	err := h.mailDomainService.UpdateDKIMSettings(c.Request.Context(), shared.GlobalID(idStr), req.Enabled, req.Selector, req.PrivateKey)
	if err != nil {
		h.log.Errorf("update DKIM settings failed: %v", err)
		response.Fail(c, messageMailDomainOp(c, err))
		return
	}

	response.Success(c, nil)
}

// PurgeDomainHandler permanently deletes a soft-deleted domain and all its accounts.
func (h *Handler) PurgeDomainHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err := h.mailDomainService.PurgeDomain(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		h.log.Errorf("purge domain failed: %v", err)
		response.Fail(c, messageMailDomainOp(c, err))
		return
	}

	response.Success(c, nil)
}
