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

	appAdmin "easymail/internal/app/admin"
	appAdminEx "easymail/internal/app/admin/exception"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

// LoginHandler authenticates an admin user and returns a JWT.
func (h *Handler) LoginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Language string `json:"language" binding:"required,oneof=en zh"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	// Login errors follow the client-selected UI language (body.language), not only Accept-Language / ?lang=.
	msg := func(key string) string {
		return appi18n.MessageForLanguage(req.Language, key)
	}

	resp, err := h.authenticationService.Login(c.Request.Context(), appAdmin.LoginRequest{
		Username: req.Username,
		Password: req.Password,
		Language: req.Language,
	})

	if err != nil {
		switch {
		case errors.Is(err, appAdminEx.ErrInvalidCredentials):
			response.Unauthorized(c, msg(appi18n.KeyAuthInvalidCredentials))
		case errors.Is(err, appAdminEx.ErrUserInactive):
			response.Unauthorized(c, msg(appi18n.KeyAuthInvalidCredentials))
		default:
			response.InternalError(c, msg(appi18n.KeyErrInternalServer))
		}
		return
	}

	response.Success(c, resp)
}
