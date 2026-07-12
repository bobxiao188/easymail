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

package admin

import (
	"net/http"
	"os"
	"path/filepath"

	"easymail/internal/portal/admin/handler"
	"easymail/internal/portal/admin/middleware"
	"easymail/pkg/config"
	appi18n "easymail/pkg/i18n"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRouter(cfg *config.AppConfig, h *handler.Handler) *gin.Engine {
	r := gin.Default()
	r.MaxMultipartMemory = 512 << 20 // ONNX uploads (admin classify-models)

	r.Use(appi18n.GinMiddleware())
	r.Use(middleware.CORS(cfg.Admin.CORSAllowedOrigins))

		// Serve frontend SPA from static directory (if configured and exists)
	// Also serve root-level static files (logo, favicon, etc.) by checking the filesystem first.
	ac := cfg.Admin
	if staticPath := filepath.Clean(ac.StaticPath); staticPath != "" {
		if _, err := os.Stat(staticPath); err == nil {
			r.StaticFS("/assets", http.Dir(filepath.Join(staticPath, "assets")))
			r.NoRoute(func(c *gin.Context) {
				p := c.Request.URL.Path
				if len(p) >= 4 && p[:4] == "/api" {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				// Serve root-level static files (logo, favicon, etc.) if they exist on disk
				filePath := filepath.Join(staticPath, p)
				if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
					http.ServeFile(c.Writer, c.Request, filePath)
					return
				}
				// Fall back to SPA index.html for client-side routing
				http.ServeFile(c.Writer, c.Request, filepath.Join(staticPath, "index.html"))
			})
		}
	}

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/healthz", h.HealthzHandler)

	// API v1 group
	api := r.Group("/api/v1")
	{
		// Admin login - no auth required
		api.POST("/admin/login", middleware.RateLimit(1*time.Minute, 10), h.LoginHandler)

		// Apply auth middleware for admin routes
		api.Use(middleware.AuthMiddleware(cfg.Admin.JWT.Secret))
		{
			// Admin profile routes
			api.GET("/admin/profile", h.GetProfileHandler)
			api.PUT("/admin/profile", h.UpdateProfileHandler)
			api.PUT("/admin/password", h.ChangePasswordHandler)
			api.PUT("/admin/language", h.UpdateLanguageHandler)
			api.PUT("/admin/skin", h.UpdateSkinHandler)

			// Domains routes (RESTful design)
			domains := api.Group("/admin/domains")
			{
				domains.GET("", h.ListDomainsHandler)
				domains.GET("/:id", h.GetDomainHandler)
				domains.POST("", h.CreateDomainHandler)
				domains.PUT("/:id", h.UpdateDomainHandler)
				domains.DELETE("/:id", h.DeleteDomainHandler)
				domains.PUT("/:id/toggle", h.ToggleDomainActiveHandler)
				domains.PUT("/:id/dkim", h.UpdateDKIMSettingsHandler)
				domains.DELETE("/:id/purge", h.PurgeDomainHandler)
			}

			// Accounts routes (RESTful design)
			accounts := api.Group("/admin/accounts")
			{
				accounts.GET("", h.ListAccountsHandler)
				accounts.GET("/:id", h.GetAccountHandler)
				accounts.POST("", h.CreateAccountHandler)
				accounts.PUT("/:id", h.UpdateAccountHandler)
				accounts.DELETE("/:id", h.DeleteAccountHandler)
				accounts.PUT("/:id/password", h.SetAccountPasswordHandler)
				accounts.PUT("/:id/toggle", h.ToggleAccountActiveHandler)
				accounts.DELETE("/:id/purge", h.PurgeAccountHandler)
			}

			// Filter routes (RESTful design)
			filter := api.Group("/admin/filter")
			{
				filter.GET("/features", h.ListFilterFeaturesHandler)

				customFeatures := filter.Group("/custom-features")
				{
					customFeatures.GET("", h.ListFilterCustomFeaturesHandler)
					customFeatures.GET("/:id", h.GetFilterCustomFeatureHandler)
					customFeatures.POST("", h.CreateFilterCustomFeatureHandler)
					customFeatures.PUT("/:id", h.UpdateFilterCustomFeatureHandler)
					customFeatures.PATCH("/:id", h.PatchFilterCustomFeatureHandler)
					customFeatures.DELETE("/:id", h.DeleteFilterCustomFeatureHandler)
				}

				rules := filter.Group("/rules")
				{
					rules.GET("", h.ListFilterRulesHandler)
					rules.GET("/:id", h.GetFilterRuleHandler)
					rules.POST("", h.CreateFilterRuleHandler)
					rules.PUT("/:id", h.UpdateFilterRuleHandler)
					rules.PATCH("/:id", h.PatchFilterRuleHandler)
					rules.DELETE("/:id", h.DeleteFilterRuleHandler)
				}

				filter.GET("/delivery-logs", h.ListFilterFilterLogsHandler)
				filter.GET("/delivery-logs/:id", h.GetFilterFilterLogHandler)
			}

			// Classify models routes (RESTful design)
			classifyModels := api.Group("/admin/classify-models")
			{
				classifyModels.GET("", h.ListClassifyModelsHandler)
				classifyModels.GET("/:id", h.GetClassifyModelHandler)
				classifyModels.POST("", h.CreateClassifyModelHandler)
				classifyModels.PUT("/:id", h.UpdateClassifyModelHandler)
				classifyModels.DELETE("/:id", h.DeleteClassifyModelHandler)
				classifyModels.POST("/:id/train", h.StartClassifyModelTrainHandler)
				classifyModels.POST("/:id/predict", h.PredictClassifyModelHandler)
				classifyModels.POST("/import", h.ImportClassifyModelHandler)
				classifyModels.GET("/:id/export", h.ExportClassifyModelHandler)

				// Model samples (associated with specific model)
				modelSamples := classifyModels.Group("/:id/samples")
				{
					modelSamples.GET("", h.ListModelSamplesHandler)
					modelSamples.POST("", h.CreateModelSamplesHandler)
					modelSamples.GET("/labels", h.ListModelSampleLabelsHandler)
					modelSamples.GET("/export", h.ExportModelSamplesTrainTxtHandler)
					modelSamples.PUT("/:sampleId", h.UpdateModelSampleHandler)
					modelSamples.DELETE("/:sampleId", h.DeleteModelSampleHandler)
				}
			}

		// Training routes (ad-hoc model training launched from the admin UI)
		training := api.Group("/admin/training")
		{
			training.POST("", h.StartTrainingHandler)
			training.GET("/:id", h.GetTrainingHandler)
		}

		// Public samples routes (global training samples, not tied to specific model)
		samples := api.Group("/admin/samples")
		{
			samples.GET("", h.ListSamplesHandler)
			samples.GET("/tags", h.ListTagsHandler)
			samples.GET("/stats", h.DescribeSamplesHandler)
			samples.POST("", h.CreateSampleHandler)
			samples.POST("/batch", h.CreateSampleHandler)
			samples.POST("/batch-delete", h.BatchDeleteSamplesHandler)
			samples.POST("/batch-update", h.BatchUpdateSamplesHandler)
			samples.PUT("/:id", h.UpdateSampleHandler)
			samples.DELETE("/:id", h.DeleteSampleHandler)
		}

			// Public sample categories routes (managed categories)
			sampleCategories := api.Group("/admin/sample-categories")
			{
				sampleCategories.GET("", h.ListCategoriesHandler)
				sampleCategories.GET("/:id", h.GetCategoryHandler)
				sampleCategories.POST("", h.CreateCategoryHandler)
				sampleCategories.PUT("/:id", h.UpdateCategoryHandler)
				sampleCategories.DELETE("/:id", h.DeleteCategoryHandler)
			}

			// Dashboard routes (RESTful design)
			dashboard := api.Group("/admin/dashboard")
			{
				dashboard.GET("", h.GetDashboardDataHandler)
				dashboard.GET("/services", h.GetServiceStatusHandler)

				// Mail stats
				mailStats := dashboard.Group("/mail-stats")
				{
					mailStats.GET("/daily", h.GetMailStatsDailyHandler)
					mailStats.GET("/monthly", h.GetMailStatsMonthlyHandler)
				}

				dashboard.GET("/top-senders", h.GetTopSendersHandler)
			}

			// Stats routes
			api.GET("/admin/stats/summary", h.GetStatsSummaryHandler)

			// Postfix configuration management routes
			postfix := api.Group("/admin/postfix")
			{
				// Agents
				agents := postfix.Group("/agents")
				{
					agents.GET("", h.ListPostfixAgentsHandler)
					agents.GET("/:id", h.GetPostfixAgentHandler)
					agents.POST("", h.CreatePostfixAgentHandler)
					agents.PUT("/:id", h.UpdatePostfixAgentHandler)
					agents.DELETE("/:id", h.DeletePostfixAgentHandler)
					agents.GET("/:id/status", h.CheckPostfixAgentStatusHandler)
				}

				// Config parameters
				configs := postfix.Group("/configs")
				{
					configs.GET("", h.ListPostfixConfigParamsHandler)
					configs.GET("/:id", h.GetPostfixConfigParamHandler)
					configs.POST("", h.CreatePostfixConfigParamHandler)
					configs.PUT("/:id", h.UpdatePostfixConfigParamHandler)
					configs.DELETE("/:id", h.DeletePostfixConfigParamHandler)
				}

				// Global settings
				postfix.GET("/settings", h.GetPostfixSettingsHandler)
				postfix.PUT("/settings", h.UpdatePostfixSettingsHandler)

				// Available variables
				postfix.GET("/variables", h.GetPostfixVariablesHandler)

				// Local IP addresses
				postfix.GET("/local-ips", h.GetLocalIPsHandler)

				// Config generation preview
				postfix.GET("/preview", h.PreviewPostfixConfigHandler)

				// Config delivery
				postfix.POST("/agents/:id/push", h.PushPostfixConfigHandler)
				postfix.POST("/agents/:id/apply", h.ApplyPostfixConfigHandler)
				postfix.POST("/agents/:id/rollback", h.RollbackPostfixConfigHandler)
				postfix.POST("/agents/:id/push-and-apply", h.PushAndApplyPostfixConfigHandler)

				// Delivery logs
				postfix.GET("/agents/:id/logs", h.ListPostfixDeliveryLogsHandler)

				// Config status summary
				postfix.GET("/status", h.GetPostfixConfigStatusHandler)

				// Queue management
				queue := postfix.Group("/queue")
				{
					queue.GET("/agents/:id", h.ListQueueHandler)
					queue.GET("/agents/:id/stats", h.GetQueueStatsHandler)
					queue.POST("/agents/:id/delete", h.DeleteQueueMessagesHandler)
					queue.POST("/agents/:id/resend", h.ResendQueueMessagesHandler)
					queue.POST("/agents/:id/flush", h.FlushQueueHandler)
				}
			}
		}
	}

	return r
}
