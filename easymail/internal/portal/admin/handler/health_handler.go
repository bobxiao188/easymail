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
	"context"
	"net/http"
	"time"

	"easymail/pkg/database"
	appi18n "easymail/pkg/i18n"

	"github.com/gin-gonic/gin"
)

// HealthzHandler reports MySQL and Redis connectivity for load balancers and ops.
func (h *Handler) HealthzHandler(c *gin.Context) {
	type dep struct {
		OK      bool   `json:"ok"`
		Message string `json:"message,omitempty"`
	}
	resp := struct {
		OK    bool `json:"ok"`
		MySQL dep  `json:"mysql"`
		Redis dep  `json:"redis"`
	}{OK: true}

	db := database.GetDB()
	if db == nil {
		resp.OK = false
		resp.MySQL = dep{OK: false, Message: appi18n.Message(c, appi18n.KeyHealthDBNil)}
	} else {
		sqlDB, err := db.DB()
		if err != nil {
			resp.OK = false
			resp.MySQL = dep{OK: false, Message: appi18n.MessageWith(c, appi18n.KeyHealthMySQLError, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)})}
		} else {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := sqlDB.PingContext(ctx); err != nil {
				resp.OK = false
				resp.MySQL = dep{OK: false, Message: appi18n.MessageWith(c, appi18n.KeyHealthMySQLError, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)})}
			} else {
				resp.MySQL = dep{OK: true}
			}
		}
	}

	rc := database.GetRedisClient()
	if rc == nil {
		resp.OK = false
		resp.Redis = dep{OK: false, Message: appi18n.Message(c, appi18n.KeyHealthRedisNil)}
	} else {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := rc.Ping(ctx).Err(); err != nil {
			resp.OK = false
			resp.Redis = dep{OK: false, Message: appi18n.MessageWith(c, appi18n.KeyHealthRedisError, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)})}
		} else {
			resp.Redis = dep{OK: true}
		}
	}

	if resp.OK {
		c.JSON(http.StatusOK, resp)
		return
	}
	c.JSON(http.StatusServiceUnavailable, resp)
}

