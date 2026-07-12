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
	"encoding/json"
	"time"

	appAdmin "easymail/internal/app/admin"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDashboardDataHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	services, err := h.dashboardService.GetServiceStatus(ctx)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrServiceStatus, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}
	localizeServiceStatuses(c, services)

	dailyStats, err := h.dashboardService.GetMailStatsDaily(ctx, 7)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrMailDaily, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	monthlyStats, err := h.dashboardService.GetMailStatsMonthly(ctx, 6)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrMailMonthly, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	topDomains, err := h.dashboardService.GetTopSenders(ctx, "domain", 24, 10)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrTopDomain, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	topAddresses, err := h.dashboardService.GetTopSenders(ctx, "address", 24, 10)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrTopAddress, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	filterPolicyDaily, _ := h.dashboardService.GetFilterPolicyStatsDaily(ctx, 7)

	data := map[string]interface{}{
		"services":            services,
		"dailyStats":          dailyStats,
		"monthlyStats":        monthlyStats,
		"topSendersByDomain":  topDomains,
		"topSendersByAddress": topAddresses,
		"filterPolicyDaily":   filterPolicyDaily,
	}

	response.Success(c, data)
}

// GetStatsSummaryHandler returns one round-trip payload for the admin overview (delivery mail stats, filter policy daily, top senders, services).
// REST path is /api/stats/summary (domain name "stats"/overview; distinct from the legacy /dashboard bundle).
func (h *Handler) GetStatsSummaryHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	generatedAt := time.Now().UTC().Format(time.RFC3339)

	days := 7
	if d := c.Query("days"); d != "" {
		if num, err := json.Number(d).Int64(); err == nil && num > 0 && num <= 90 {
			days = int(num)
		}
	}
	months := 6
	if m := c.Query("months"); m != "" {
		if num, err := json.Number(m).Int64(); err == nil && num > 0 && num <= 36 {
			months = int(num)
		}
	}
	topHours := 24
	if hr := c.Query("topHours"); hr != "" {
		if num, err := json.Number(hr).Int64(); err == nil && num > 0 && num <= 168 {
			topHours = int(num)
		}
	}
	topLimit := 10
	if l := c.Query("topLimit"); l != "" {
		if num, err := json.Number(l).Int64(); err == nil && num > 0 && num <= 100 {
			topLimit = int(num)
		}
	}

	services, err := h.dashboardService.GetServiceStatus(ctx)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrServiceStatus, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}
	localizeServiceStatuses(c, services)

	dailyStats, err := h.dashboardService.GetMailStatsDaily(ctx, days)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrMailDaily, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	monthlyStats, err := h.dashboardService.GetMailStatsMonthly(ctx, months)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrMailMonthly, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	topDomains, err := h.dashboardService.GetTopSenders(ctx, "domain", topHours, topLimit)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrTopDomain, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	topAddresses, err := h.dashboardService.GetTopSenders(ctx, "address", topHours, topLimit)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrTopAddress, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	filterPolicyDaily, _ := h.dashboardService.GetFilterPolicyStatsDaily(ctx, days)

	response.Success(c, map[string]interface{}{
		"generatedAt":         generatedAt,
		"services":            services,
		"dailyStats":          dailyStats,
		"monthlyStats":        monthlyStats,
		"topSendersByDomain":  topDomains,
		"topSendersByAddress": topAddresses,
		"filterPolicyDaily":   filterPolicyDaily,
	})
}

func (h *Handler) GetServiceStatusHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	services, err := h.dashboardService.GetServiceStatus(ctx)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrServiceStatus, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}
	localizeServiceStatuses(c, services)

	response.Success(c, services)
}

func (h *Handler) GetMailStatsDailyHandler(c *gin.Context) {
	days := 7
	if d := c.Query("days"); d != "" {
		if num, err := json.Number(d).Int64(); err == nil && num > 0 {
			days = int(num)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := h.dashboardService.GetMailStatsDaily(ctx, days)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrMailDaily, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	response.Success(c, stats)
}

func (h *Handler) GetMailStatsMonthlyHandler(c *gin.Context) {
	months := 6
	if m := c.Query("months"); m != "" {
		if num, err := json.Number(m).Int64(); err == nil && num > 0 {
			months = int(num)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := h.dashboardService.GetMailStatsMonthly(ctx, months)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrMailMonthly, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	response.Success(c, stats)
}

func (h *Handler) GetTopSendersHandler(c *gin.Context) {
	senderType := c.Query("type")
	if senderType != "domain" && senderType != "address" {
		senderType = "domain"
	}

	hours := 24
	if hr := c.Query("hours"); hr != "" {
		if num, err := json.Number(hr).Int64(); err == nil && num > 0 {
			hours = int(num)
		}
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if num, err := json.Number(l).Int64(); err == nil && num > 0 && num <= 100 {
			limit = int(num)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	senders, err := h.dashboardService.GetTopSenders(ctx, senderType, hours, limit)
	if err != nil {
		response.InternalError(c, appi18n.MessageWith(c, appi18n.KeyDashboardErrTopSenders, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}

	response.Success(c, senders)
}

// localizeServiceStatuses replaces Description message IDs with localized text for API consumers.
func localizeServiceStatuses(c *gin.Context, ss []appAdmin.ServiceStatus) {
	for i := range ss {
		ss[i].Description = appi18n.Message(c, ss[i].Description)
	}
}
