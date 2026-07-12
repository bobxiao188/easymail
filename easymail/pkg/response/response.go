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

// Package response provides JSON helpers for Gin HTTP handlers with optional i18n message IDs.
package response

import (
	"net/http"

	appi18n "easymail/pkg/i18n"
	"easymail/pkg/types"

	"github.com/gin-gonic/gin"
)

// Response is the standard JSON envelope for admin and webmail APIs.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// PaginatedResponse is returned for list endpoints that include pagination metadata.
type PaginatedResponse struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    interface{}          `json:"data"`
	Meta    types.PaginationMeta `json:"meta"`
}

// Success writes HTTP 200 with code 0 and a localized success message.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: appi18n.Message(c, appi18n.KeyAPISuccess),
		Data:    data,
	})
}

// SuccessWithPagination writes HTTP 200 with paginated payload and localized success message.
func SuccessWithPagination(c *gin.Context, data interface{}, meta types.PaginationMeta) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    0,
		Message: appi18n.Message(c, appi18n.KeyAPISuccess),
		Data:    data,
		Meta:    meta,
	})
}

// ErrorWithStatus writes a JSON error with the given HTTP status and application code.
func ErrorWithStatus(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// Fail writes HTTP 400 with code -1.
func Fail(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusBadRequest, -1, message)
}

// Unauthorized writes HTTP 401. If message is empty, a localized unauthorized string is used.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = appi18n.Message(c, appi18n.KeyAuthUnauthorized)
	}
	ErrorWithStatus(c, http.StatusUnauthorized, 401, message)
}

// NotFound writes HTTP 404.
func NotFound(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusNotFound, 404, message)
}

// BadRequest writes HTTP 400 with code 400.
func BadRequest(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusBadRequest, 400, message)
}

// InternalError writes HTTP 500 with code 500.
func InternalError(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusInternalServerError, 500, message)
}
