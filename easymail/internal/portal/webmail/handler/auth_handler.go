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
	"errors"

	"easymail/internal/domain/management"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a mailbox user and returns a JWT for subsequent API calls.
func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, management.ErrMailUserNotFound),
			errors.Is(err, management.ErrMailUserDomainInvalid),
			errors.Is(err, management.ErrMailUserInvalidPass):
			response.Unauthorized(c, appi18n.Message(c, appi18n.KeyWebmailAuthInvalidCredentials))
		case errors.Is(err, management.ErrMailUserInactive):
			response.Unauthorized(c, appi18n.Message(c, appi18n.KeyWebmailAuthInvalidCredentials))
		default:
			response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailAuthLoginFailed))
		}
		return
	}
	response.Success(c, resp)
}

// Healthz returns a simple liveness payload for load balancers.
func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

// Logout logs out the current user.
func (h *Handler) Logout(c *gin.Context) {
	// Clear session or token blacklist if needed
	response.Success(c, gin.H{"ok": true})
}
