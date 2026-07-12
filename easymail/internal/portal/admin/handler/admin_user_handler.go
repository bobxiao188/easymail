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
	"strings"

	appAdminEx "easymail/internal/app/admin/exception"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetProfileHandler returns the authenticated admin user's profile.
func (h *Handler) GetProfileHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	userInfo, err := h.authenticationService.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, appAdminEx.ErrUserNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundUser))
			return
		}
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrProfileLoadFailed))
		return
	}

	response.Success(c, userInfo)
}

// UpdateProfileRequest binds JSON for profile updates.
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Avatar   string `json:"avatar"`
}

// UpdateProfileHandler updates nickname, email, and avatar.
func (h *Handler) UpdateProfileHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Email = strings.TrimSpace(req.Email)

	userInfo, err := h.authenticationService.UpdateProfile(c.Request.Context(), userID.(string), req.Nickname, req.Email, req.Avatar)
	if err != nil {
		response.Fail(c, messageAdminInactiveOrOpFailed(c, err))
		return
	}
	response.Success(c, userInfo)
}

// ChangePasswordRequest binds old and new password fields.
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// UpdateLanguageRequest binds UI language (en or zh).
type UpdateLanguageRequest struct {
	Language string `json:"language" binding:"required,oneof=en zh"`
}

// UpdateSkinRequest binds UI theme.
type UpdateSkinRequest struct {
	Skin string `json:"skin" binding:"required,oneof=dark light"`
}

// ChangePasswordHandler changes the admin user's password.
func (h *Handler) ChangePasswordHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if err := h.authenticationService.ChangePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword); err != nil {
		response.Fail(c, messageAdminChangePassword(c, err))
		return
	}
	response.Success(c, nil)
}

// UpdateLanguageHandler persists the preferred UI language.
func (h *Handler) UpdateLanguageHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	var req UpdateLanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidLanguageParam))
		return
	}

	if err := h.authenticationService.UpdateLanguage(c.Request.Context(), userID.(string), req.Language); err != nil {
		response.Fail(c, messageAdminInactiveOrOpFailed(c, err))
		return
	}
	response.Success(c, nil)
}

// UpdateSkinHandler persists the preferred UI skin.
func (h *Handler) UpdateSkinHandler(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "")
		return
	}

	var req UpdateSkinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidSkinParam))
		return
	}

	if err := h.authenticationService.UpdateSkin(c.Request.Context(), userID.(string), req.Skin); err != nil {
		response.Fail(c, messageAdminInactiveOrOpFailed(c, err))
		return
	}
	response.Success(c, nil)
}
