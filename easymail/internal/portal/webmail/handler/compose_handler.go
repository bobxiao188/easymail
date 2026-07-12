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
	"io"
	"mime"
	"path/filepath"
	"strings"

	"easymail/internal/domain/messaging"
	mailservice "easymail/internal/domain/messaging/service"
	"easymail/internal/portal/webmail/middleware"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

const sendMultipartMaxMemory = 32 << 20

// SendMessage accepts multipart/form-data: to, cc, bcc, subject, html, text; repeated file field "attachments".
func (h *Handler) SendMessage(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	v, ok := c.Get("webmail_email")
	if !ok {
		response.Unauthorized(c, "")
		return
	}
	jwtEmail, _ := v.(string)

	if err := c.Request.ParseMultipartForm(sendMultipartMaxMemory); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailComposeInvalidMultipart))
		return
	}

	to := strings.TrimSpace(c.PostForm("to"))
	cc := strings.TrimSpace(c.PostForm("cc"))
	bcc := strings.TrimSpace(c.PostForm("bcc"))
	subject := strings.TrimSpace(c.PostForm("subject"))
	html := c.PostForm("html")
	text := strings.TrimSpace(c.PostForm("text"))

	var attaches []messaging.Attachment
	if c.Request.MultipartForm != nil && c.Request.MultipartForm.File != nil {
		for _, fh := range c.Request.MultipartForm.File["attachments"] {
			f, err := fh.Open()
			if err != nil {
				response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailComposeAttachmentOpenFailed))
				return
			}
			data, err := io.ReadAll(f)
			_ = f.Close()
			if err != nil {
				response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailComposeAttachmentReadFailed))
				return
			}
			name := filepath.Base(fh.Filename)
			if name == "" || name == "." {
				name = "attachment"
			}
			ct := fh.Header.Get("Content-Type")
			if ct == "" || ct == "application/octet-stream" {
				ct = mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
			}
			if ct == "" {
				ct = "application/octet-stream"
			}
			attaches = append(attaches, messaging.Attachment{
				Name:        name,
				ContentType: ct,
				Data:        data,
			})
		}
	}

	req := mailservice.ComposeRequest{
		To:          to,
		Cc:          cc,
		Bcc:         bcc,
		Subject:     subject,
		Text:        text,
		HTML:        html,
		Attachments: attaches,
	}

	if _, err := h.mailService.SendCompose(c.Request.Context(), aid, jwtEmail, req); err != nil {
		h.writeMailErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
