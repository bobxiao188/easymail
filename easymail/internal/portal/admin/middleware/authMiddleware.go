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

package middleware

import (
	"strings"

	"easymail/pkg/i18n"
	"easymail/pkg/jwt"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Bearer JWT and stores user claims in the Gin context.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, i18n.Message(c, i18n.KeyAuthHeaderInvalid))
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := jwt.ParseToken(tokenString, jwtSecret)
		if err != nil {
			response.Unauthorized(c, i18n.Message(c, i18n.KeyAuthTokenInvalid))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}

