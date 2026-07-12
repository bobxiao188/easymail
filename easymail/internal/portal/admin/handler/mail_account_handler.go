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
	"easymail/internal/domain/messaging/storagepath"
	"easymail/internal/domain/shared"
	appi18n "easymail/pkg/i18n"
	"math/rand"
	"strconv"

	"easymail/pkg/response"
	"easymail/pkg/types"

	"github.com/gin-gonic/gin"
)

// AccountListRequest binds query parameters for paginated account listing.
type AccountListRequest struct {
	types.PaginationRequest
	DomainID string `form:"domainId" json:"domainId"`
	Keyword  string `form:"keyword" json:"keyword"`
}

// CreateAccountRequest binds JSON for creating a mailbox account.
type CreateAccountRequest struct {
	Username     string `json:"username" binding:"required"`
	DomainID     string `json:"domainId" binding:"required"`
	Password     string `json:"password" binding:"required,min=6"`
	StorageQuota int64  `json:"storageQuota"`
}

// UpdateAccountRequest binds JSON for updating an account.
type UpdateAccountRequest struct {
	Username     string `json:"username"`
	Active       *bool  `json:"active"`
	StorageQuota int64  `json:"storageQuota"`
}

// SetPasswordRequest binds JSON for admin password reset.
type SetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// ListAccountsHandler returns paginated accounts for a domain filter.
func (h *Handler) ListAccountsHandler(c *gin.Context) {
	var req AccountListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	// Validate pagination parameters to prevent divide by zero
	req.Validate()

	// Support status filter (-1 = all, 0 = inactive, 1 = active)
	status := c.DefaultQuery("status", "-1")

	// convert status to int
	statusInt, err := strconv.Atoi(status)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	domainID := shared.GlobalID(req.DomainID)
	accounts, total, err := h.mailAccountService.List(c.Request.Context(), domainID, req.Keyword, statusInt, req.Page, req.PageSize)
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}

	meta := types.NewPaginationMeta(req.Page, req.PageSize, total)
	response.SuccessWithPagination(c, accounts, meta)
}

// GetAccountHandler returns a single account by ID.
func (h *Handler) GetAccountHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	mailAccount, err := h.mailAccountService.GetByID(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundAccount))
		return
	}

	response.Success(c, mailAccount)
}

// CreateAccountHandler creates a new mailbox account under a domain.
func (h *Handler) CreateAccountHandler(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	domainGID := shared.GlobalID(req.DomainID)
	mailDomain, err := h.mailDomainService.GetByID(c.Request.Context(), domainGID)
	if err != nil || mailDomain == nil {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundDomain))
		return
	}
	if mailDomain.IsDeleted {
		response.Fail(c, appi18n.Message(c, appi18n.KeyErrDomainDeleted))
		return
	}

	// Check if username already taken within domain
	email := req.Username + "@" + mailDomain.Name
	_, err = h.mailAccountService.GetByFullEmail(c.Request.Context(), email)
	if err == nil {
		response.Fail(c, appi18n.Message(c, appi18n.KeyErrUsernameTaken))
		return
	}

	storageQuota := req.StorageQuota
	if storageQuota == 0 {
		storageQuota = 1000
	}

	dataPath := storagepath.MailUserDataPath(mailDomain.Name, email)

	storageID := 0
	if len(h.storageIDs) > 0 {
		storageID = h.storageIDs[rand.Intn(len(h.storageIDs))]
	}

	newUser, err := h.mailAccountService.Create(c.Request.Context(), domainGID, req.Username, req.Password, mailDomain.Name, storageQuota, dataPath, storageID)
	if err != nil {
		h.log.Warnf("create account failed: %v", err)
		response.Fail(c, messageMailAccountOp(c, err))
		return
	}

	// Provision user directories and default folders
	if newUser != nil {
		if provisionErr := h.provisionService.Provision(c.Request.Context(), newUser.ID); provisionErr != nil {
			h.log.Warnf("provision account %s failed: %v", newUser.ID, provisionErr)
		}
	}

	response.Success(c, nil)
}

// UpdateAccountHandler updates account fields; disabled accounts may only be re-enabled via active=true.
func (h *Handler) UpdateAccountHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	gid := shared.GlobalID(idStr)
	mailAccount, err := h.mailAccountService.GetByID(c.Request.Context(), gid)
	if err != nil {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundAccount))
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	if !mailAccount.Validate() && (req.Active == nil || !*req.Active) {
		response.Fail(c, appi18n.Message(c, appi18n.KeyErrAccountDisabled))
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}
	err = h.mailAccountService.Update(c.Request.Context(), gid, req.Username, active, req.StorageQuota)
	if err != nil {
		h.log.Errorf("update account failed: %v", err)
		response.Fail(c, messageMailAccountOp(c, err))
		return
	}

	response.Success(c, nil)
}

// DeleteAccountHandler soft-deletes an account.
func (h *Handler) DeleteAccountHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err := h.mailAccountService.SoftDelete(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		h.log.Errorf("soft-delete account failed: %v", err)
		response.Fail(c, messageMailAccountOp(c, err))
		return
	}

	response.Success(c, nil)
}

// PurgeAccountHandler permanently deletes a soft-deleted account and its data files.
func (h *Handler) PurgeAccountHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err := h.mailAccountService.PurgeDeleted(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		h.log.Errorf("purge account failed: %v", err)
		response.Fail(c, messageMailAccountOp(c, err))
		return
	}

	response.Success(c, nil)
}

// SetAccountPasswordHandler sets the account password (admin).
func (h *Handler) SetAccountPasswordHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	var req SetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrPasswordMinLen))
		return
	}

	err := h.mailAccountService.ResetPassword(c.Request.Context(), shared.GlobalID(idStr), req.Password)
	if err != nil {
		h.log.Errorf("reset password failed: %v", err)
		response.Fail(c, messageMailAccountOp(c, err))
		return
	}

	response.Success(c, nil)
}

// ToggleAccountActiveHandler toggles the active flag.
func (h *Handler) ToggleAccountActiveHandler(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}

	err := h.mailAccountService.ToggleActive(c.Request.Context(), shared.GlobalID(idStr))
	if err != nil {
		h.log.Errorf("toggle account active failed: %v", err)
		response.Fail(c, messageMailAccountOp(c, err))
		return
	}

	response.Success(c, nil)
}
