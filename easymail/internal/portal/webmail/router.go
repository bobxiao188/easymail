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

package webmail

import (
	"net/http"
	"os"
	"path/filepath"

	"easymail/internal/portal/webmail/handler"
	"easymail/internal/portal/webmail/middleware"
	"easymail/pkg/config"
	appi18n "easymail/pkg/i18n"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRouter(cfg *config.AppConfig, h *handler.Handler) *gin.Engine {
	r := gin.Default()
	wc := cfg.Webmail

	r.Use(appi18n.GinMiddleware())
	r.Use(middleware.CORS(wc.CORSAllowedOrigins))

	// Serve frontend SPA from static directory (if configured and exists)
	// Also serve root-level static files (logo, favicon, etc.) by checking the filesystem first.
	if staticPath := filepath.Clean(wc.StaticPath); staticPath != "" {
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
	r.GET("/healthz", h.Healthz)

	// API v1 group
	api := r.Group("/api/v1")
	{
		// Auth routes - no auth required for login
		api.POST("/auth/login", middleware.RateLimit(1*time.Minute, 10), h.Login)

		// Apply auth middleware to the api group.
		// All routes registered below this inherit the middleware.
		api.Use(middleware.AuthMiddleware(wc.JWT.Secret))
		{
			// Auth routes (after auth middleware for protected endpoints)
			api.POST("/auth/logout", h.Logout)
			api.POST("/auth/password/change", middleware.TrialMode(cfg.TrialMode), h.ChangePassword)

			// Profile routes
			api.GET("/profile", h.GetProfile)
			api.PUT("/profile", h.UpdateProfile)
			api.GET("/settings", h.GetSettings)
			api.PUT("/settings", h.UpdateSettings)

			// Messages routes (RESTful design)
			messages := api.Group("/messages")
			{
				// List/search messages with pagination and filters
				messages.GET("", h.GetEmailList)
				messages.GET("/search", h.SearchEmail)
				messages.GET("/stats", h.GetMailStats)

				// Single message operations
				messages.GET("/:id", h.GetMessage)
				messages.PATCH("/:id", h.PatchMessage)
				messages.DELETE("/:id", h.DeleteMessage)

				// Message sub-resources
				message := messages.Group("/:id")
				{
					// Labels
					message.POST("/labels", h.SetEmailLabels)
					message.GET("/labels", h.GetEmailLabels)

					// Actions
					message.POST("/move", h.MoveEmail)
					message.PATCH("/read", h.MarkAsRead)
					message.PATCH("/star", h.ToggleStar)
					message.POST("/reply", middleware.TrialMode(cfg.TrialMode), h.ReplyToEmail)
					message.POST("/forward", middleware.TrialMode(cfg.TrialMode), h.ForwardEmail)

					// Attachments
					message.GET("/attachments/:index", h.DownloadAttachment)
					message.GET("/attachments/zip", h.DownloadAllAttachments)

					// Body and raw content
					message.GET("/body", h.GetMessageBody)
					message.GET("/raw", h.GetMessageRaw)
				}

				// Batch operations
				messages.POST("/batch", h.BatchMessages)

				// Send email
				messages.POST("/send", middleware.TrialMode(cfg.TrialMode), h.SendEmail)

				// Attachment upload
				messages.POST("/attachment/upload", h.UploadAttachment)
			}

			// Drafts routes (RESTful design)
			drafts := api.Group("/drafts")
			{
				drafts.POST("", h.SaveDraft)
				drafts.GET("/:id", h.EditDraft)
				drafts.PATCH("/:id", h.UpdateDraft)
				drafts.DELETE("/:id", h.DeleteDraft)
			}

			// Contacts routes (RESTful design)
			contacts := api.Group("/contacts")
			{
				contacts.GET("", h.ListContacts)
				contacts.GET("/:id", h.GetContact)
				contacts.POST("", h.CreateContact)
				contacts.PATCH("/:id", h.UpdateContact)
				contacts.DELETE("/:id", h.DeleteContact)
			}

			// Contact groups routes (RESTful design)
			contactGroups := api.Group("/contact-groups")
			{
				contactGroups.GET("", h.ListContactGroups)
				contactGroups.GET("/:id", h.GetContactGroup)
				contactGroups.POST("", h.CreateContactGroup)
				contactGroups.PATCH("/:id", h.UpdateContactGroup)
				contactGroups.DELETE("/:id", h.DeleteContactGroup)
				// Get contacts in a group
				contactGroups.GET("/:id/contacts", h.GetGroupContacts)
			}

			// Labels routes (RESTful design)
			labels := api.Group("/labels")
			{
				labels.GET("", h.ListLabels)
				labels.POST("", h.CreateLabel)
				labels.PUT("/:id", h.UpdateLabel)
				labels.PATCH("/:id", h.UpdateLabel)
				labels.DELETE("/:id", h.DeleteLabel)
			}

			// Folders routes (RESTful design)
			folders := api.Group("/folders")
			{
				folders.GET("", h.ListFolders)
				folders.POST("", h.CreateFolder)
				folders.PATCH("/:id", h.RenameFolder)
				folders.DELETE("/:id", h.DeleteFolder)
				// Get messages in a folder
				folders.GET("/:id/messages", h.ListMessages)
			}
		}
	}
	return r
}
