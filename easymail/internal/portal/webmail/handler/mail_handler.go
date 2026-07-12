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
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"

	enmime "github.com/jhillyerd/enmime/v2"
	"github.com/microcosm-cc/bluemonday"

	mailexception "easymail/internal/domain/messaging/exception"
	mailservice "easymail/internal/domain/messaging/service"
	"easymail/internal/domain/shared"
	"easymail/internal/portal/webmail/middleware"
	"easymail/pkg/constants"
	"easymail/pkg/response"
	"easymail/pkg/types"

	messaging "easymail/internal/domain/messaging"
	appi18n "easymail/pkg/i18n"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListFolders(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	list, err := h.mailService.ListFolders(c.Request.Context(), aid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}
	// Localize system folder display names based on request language
	for i := range list {
		if localized := appi18n.FolderDisplayName(c, int64(list[i].Kind)); localized != "" {
			list[i].Name = localized
		}
	}
	response.Success(c, list)
}

func (h *Handler) ListMessages(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	fid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || fid < 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidFolderID))
		return
	}
	if !h.folderAllowed(c, aid, fid) {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
		return
	}
	q := mailservice.ListQuery{
		Page:       parseIntDefault(c.Query("page"), 1),
		PageSize:   parseIntDefault(c.Query("page_size"), 20),
		OrderField: c.DefaultQuery("order_field", "mail_time"),
		OrderDir:   c.DefaultQuery("order_dir", "DESC"),
		Search:     c.Query("search"),
		LabelID:    int64(parseIntDefault(c.Query("label_id"), 0)),
	}
	total, news, items, err := h.mailService.ListMessages(c.Request.Context(), aid, fid, q)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}
	meta := types.NewPaginationMeta(q.Page, q.PageSize, total)
	// Enrich each message with a has_attachments hint
	type messageRow struct {
		ID             int64                `json:"id"`
		Sender         string               `json:"sender"`
		Recipient      string               `json:"recipient"`
		CarbonCopy     string               `json:"carbonCopy"`
		BlindCopy      string               `json:"blindCopy"`
		Subject        string               `json:"subject"`
		Snippet        string               `json:"snippet"`
		MailTime       string               `json:"mailTime"`
		MailSize       int64                `json:"mailSize"`
		FolderID       int64                `json:"folderId"`
		ReadStatus     constants.ReadStatus `json:"readStatus"`
		Flagged        bool                 `json:"flagged"`
		Seen           bool                 `json:"seen"`
		HasAttachments bool                 `json:"hasAttachments"`
		Labels         []LabelItem          `json:"labels"`
	}
	rows := make([]messageRow, len(items))
	for i, m := range items {
		rows[i] = messageRow{
			ID:             m.ID,
			Sender:         m.Sender,
			Recipient:      m.Recipient,
			CarbonCopy:     m.CarbonCopy,
			BlindCopy:      m.BlindCopy,
			Subject:        m.Subject,
			Snippet:        m.Snippet,
			MailTime:       m.MailTime,
			MailSize:       m.MailSize,
			FolderID:       m.FolderID,
			ReadStatus:     m.ReadStatus,
			Flagged:        m.Flagged,
			Seen:           m.ReadStatus != constants.UnRead,
			HasAttachments: m.HasAttachments,
			Labels:         []LabelItem{},
		}
	}
	response.SuccessWithPagination(c, gin.H{"list": rows, "unread_in_folder": news}, meta)
}

// LabelItem represents a label summary in API responses
type LabelItem struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type messageAttachment struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type,omitempty"`
}

type messageDetail struct {
	ID          int64                `json:"id"`
	Sender      Recipient            `json:"from"`
	To          []Recipient          `json:"to"`
	Cc          []Recipient          `json:"cc"`
	Bcc         []Recipient          `json:"bcc"`
	Subject     string               `json:"subject"`
	MailTime    string               `json:"mailTime"`
	MailSize    int64                `json:"mailSize"`
	FolderID    int64                `json:"folderId"`
	Read        bool                 `json:"read"`
	Seen        bool                 `json:"seen"`
	ReadStatus  constants.ReadStatus `json:"readStatus"`
	Flagged     bool                 `json:"flagged"`
	Labels      []LabelItem          `json:"labels"`
	Attachments []messageAttachment  `json:"attachments,omitempty"`
	Body        string               `json:"body"`
	BodyHtml    string               `json:"bodyHtml,omitempty"`
	Headers     map[string]string    `json:"headers,omitempty"`
}

func (h *Handler) GetMessage(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}
	e, err := h.mailService.GetMessage(c.Request.Context(), aid, mid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Auto mark as read when viewing email
	if e.ReadStatus == constants.UnRead {
		if markErr := h.mailService.MarkRead(c.Request.Context(), aid, mid, constants.WebRead); markErr != nil {
			log.Printf("Failed to auto mark email as read: %v", markErr)
		} else {
			// Update the read status in the email object
			e.ReadStatus = constants.WebRead
		}
	}

	// Extract body from email using enmime
	body, err := h.extractBodyFromEmail(c.Request.Context(), aid, mid)
	if err != nil {
		log.Printf("Failed to extract body from email: %v, trying DB", err)
		// Fallback to database body
		body = e.Body
	}

	detail := messageDetail{
		ID: e.ID,
		Sender: func() Recipient {
			parsed := parseAddresses(e.Sender)
			if len(parsed) > 0 {
				return parsed[0]
			}
			return Recipient{Email: e.Sender}
		}(),
		To:         parseAddresses(e.Recipient),
		Cc:         parseAddresses(e.CarbonCopy),
		Bcc:        parseAddresses(e.BlindCopy),
		Subject:    e.Subject,
		MailTime:   e.MailTime,
		MailSize:   e.MailSize,
		FolderID:   e.FolderID,
		Read:       e.ReadStatus != constants.UnRead,
		Seen:       e.ReadStatus != constants.UnRead,
		ReadStatus: e.ReadStatus,
		Flagged:    e.Flagged,
		Labels:     []LabelItem{},
		Body:       body,
	}

	// Load labels for this email
	if lbls, err := h.mailService.GetEmailLabels(c.Request.Context(), aid, e.ID); err == nil {
		labels := make([]LabelItem, len(lbls))
		for i, l := range lbls {
			labels[i] = LabelItem{ID: l.ID, Name: l.Name, Color: l.Color}
		}
		detail.Labels = labels
	}

	// Check if body is HTML
	if strings.Contains(body, "<html") || strings.Contains(body, "<div") || strings.Contains(body, "<p") || strings.Contains(body, "<br") {
		detail.BodyHtml = body
	}

	if atts, err := h.mailService.ListMessageAttachments(c.Request.Context(), aid, mid); err == nil && len(atts) > 0 {
		detail.Attachments = make([]messageAttachment, 0, len(atts))
		for _, a := range atts {
			detail.Attachments = append(detail.Attachments, messageAttachment{
				Index:       a.Index,
				Name:        a.Name,
				Size:        a.Size,
				ContentType: a.ContentType,
			})
		}
	}

	// Get raw headers for headers tab
	rc, _, err := h.mailService.GetMessageRaw(c.Request.Context(), aid, mid)
	if err == nil {
		defer rc.Close()
		// Parse headers from raw email
		headers := make(map[string]string)

		scanner := bufio.NewScanner(rc)
		inHeader := true
		for scanner.Scan() && inHeader {
			line := scanner.Text()
			if line == "" {
				inHeader = false
				continue
			}

			if strings.Contains(line, ":") {
				idx := strings.Index(line, ":")
				key := strings.TrimSpace(line[:idx])
				value := strings.TrimSpace(line[idx+1:])

				headers[strings.ToLower(key)] = value
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Scanner error: %v", err)
		}

		detail.Headers = headers
	} else {
		log.Printf("Failed to get raw message: %v", err)
	}

	response.Success(c, detail)
}

// DownloadAttachment streams one attachment for a message (GET .../attachments/:index).

// DownloadAllAttachments downloads all attachments as a ZIP archive.
func (h *Handler) DownloadAllAttachments(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid < 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidFolderID))
		return
	}
	data, filename, err := h.mailService.OpenAttachmentsZip(c.Request.Context(), aid, mid)
	if err != nil {
		log.Printf("webmail handler error (zip): %v", err)
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(200, "application/zip", data)
}

func (h *Handler) DownloadAttachment(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil || idx < 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidAttachmentIndex))
		return
	}
	rc, filename, contentType, size, err := h.mailService.OpenRawAttachment(c.Request.Context(), aid, mid, idx)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}
	defer rc.Close()

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Status(200)
	_, _ = io.Copy(c.Writer, rc)
}

func (h *Handler) GetMessageBody(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}
	body, err := h.mailService.GetMessageBody(c.Request.Context(), aid, mid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}
	// Return structured body
	isHtml := strings.Contains(body, "<html") || strings.Contains(body, "<div") || strings.Contains(body, "<p") || strings.Contains(body, "<br")
	if isHtml {
		response.Success(c, gin.H{"html": body, "text": ""})
	} else {
		response.Success(c, gin.H{"html": "", "text": body})
	}
}

func (h *Handler) GetMessageRaw(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}
	rc, size, err := h.mailService.GetMessageRaw(c.Request.Context(), aid, mid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}
	defer rc.Close()
	c.Header("Content-Type", "message/rfc822")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Status(200)
	_, _ = io.Copy(c.Writer, rc)
}

type patchMessageReq struct {
	Read     *bool  `json:"read"`
	Seen     *bool  `json:"seen"`
	FolderID *int64 `json:"folderId"`
	Flagged  *bool  `json:"flagged"`
}

func (h *Handler) PatchMessage(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}
	var req patchMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	// Accept both "read" and "seen" JSON fields
	read := req.Read
	if read == nil && req.Seen != nil {
		read = req.Seen
	}
	ctx := c.Request.Context()
	if read != nil {
		if *read {
			if err := h.mailService.MarkRead(ctx, aid, mid, constants.WebRead); err != nil {
				h.writeMailErr(c, err)
				return
			}
		} else {
			if err := h.mailService.MarkRead(ctx, aid, mid, constants.UnRead); err != nil {
				h.writeMailErr(c, err)
				return
			}
		}
	}
	if req.FolderID != nil && *req.FolderID > 0 {
		if !h.folderAllowed(c, aid, *req.FolderID) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
			return
		}
		if err := h.mailService.MoveMessage(ctx, aid, mid, *req.FolderID); err != nil {
			h.writeMailErr(c, err)
			return
		}
	}
	if req.Flagged != nil {
		if err := h.mailService.SetMessageFlagged(ctx, aid, mid, *req.Flagged); err != nil {
			h.writeMailErr(c, err)
			return
		}
	}
	if read == nil && req.FolderID == nil && req.Flagged == nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailNoChanges))
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) DeleteMessage(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}
	ctx := c.Request.Context()
	permanent := c.Query("permanent") == "1" || strings.EqualFold(c.Query("permanent"), "true")
	var delErr error
	if permanent {
		delErr = h.mailService.PurgeMessage(ctx, aid, mid)
	} else {
		// Move to Trash folder instead of just marking is_deleted=true
		trashID := h.findSystemFolderID(ctx, aid, constants.Trash)
		if trashID > 0 {
			delErr = h.mailService.MoveMessage(ctx, aid, mid, trashID)
		} else {
			delErr = h.mailService.MoveToTrash(ctx, aid, mid)
		}
	}
	if delErr != nil {
		h.writeMailErr(c, delErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

type createFolderReq struct {
	Name string `json:"name"`
}

type renameFolderReq struct {
	Name string `json:"name"`
}

type batchMessagesReq struct {
	IDs      []int64 `json:"ids"`
	IDsOld   []int64 `json:"messageIds"`
	Op       string  `json:"op"`
	Action   string  `json:"action"`
	FolderID *int64  `json:"folderId"`
	Dest     *int64  `json:"dest"`
	Seen     *bool   `json:"seen"`
}

func (h *Handler) CreateFolder(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	var req createFolderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	dto, err := h.mailService.CreateFolder(c.Request.Context(), aid, req.Name)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *Handler) RenameFolder(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	fid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || fid < 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidFolderID))
		return
	}
	var req renameFolderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if err := h.mailService.RenameFolder(c.Request.Context(), aid, fid, req.Name); err != nil {
		h.writeMailErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) DeleteFolder(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	fid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || fid < 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidFolderID))
		return
	}
	if err := h.mailService.DeleteFolder(c.Request.Context(), aid, fid); err != nil {
		h.writeMailErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) BatchMessages(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	var req batchMessagesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	// Accept both "op" and "action" JSON fields
	op := req.Op
	if op == "" {
		op = req.Action
	}
	folderID := req.FolderID
	if folderID == nil {
		folderID = req.Dest
	}
	// Accept "message_ids" as alias for "ids"
	ids := req.IDs
	if len(ids) == 0 {
		ids = req.IDsOld
	}
	// Infer operation from fields when op is not provided
	op = strings.TrimSpace(strings.ToLower(op))
	ctx := c.Request.Context()
	var err error

	// Standard operations: delete, move, mark_read, mark_unread, toggle_star
	switch op {
	case "toggle_star":
		// For each message, toggle its starred status
		for _, mid := range ids {
			msg, getErr := h.mailService.GetMessage(ctx, aid, mid)
			if getErr != nil {
				err = getErr
				continue
			}
			newFlagged := !msg.Flagged
			if setErr := h.mailService.SetMessageFlagged(ctx, aid, mid, newFlagged); setErr != nil {
				err = setErr
			}
		}
	case "delete":
		// Move to trash (soft delete)
		trashID := h.findSystemFolderID(ctx, aid, constants.Trash)
		if trashID > 0 {
			err = h.mailService.BatchMove(ctx, aid, ids, trashID)
		} else {
			err = h.mailService.BatchMoveToTrash(ctx, aid, ids)
		}
	case "permanent_delete":
		// Permanently delete (hard delete) - physically remove from storage
		for _, mid := range ids {
			if purgeErr := h.mailService.PurgeMessage(ctx, aid, mid); purgeErr != nil {
				err = purgeErr
				// Continue processing other messages even if one fails
			}
		}
	case "move":
		// move operation requires folder_id
		if folderID == nil || *folderID < 0 {
			response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailFolderIDRequired))
			return
		}
		if !h.folderAllowed(c, aid, *folderID) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
			return
		}
		err = h.mailService.BatchMove(ctx, aid, ids, *folderID)
	case "mark_read":
		err = h.mailService.BatchMarkRead(ctx, aid, ids, true)
	case "mark_unread":
		err = h.mailService.BatchMarkRead(ctx, aid, ids, false)
	default:
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailUnknownOp))
		return
	}
	if err != nil {
		h.writeMailErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) folderAllowed(c *gin.Context, MailUserID shared.GlobalID, folderID int64) bool {
	return h.mailService.ValidateFolder(c.Request.Context(), MailUserID, folderID)
}

// findSystemFolderID looks up the actual database folder ID for a system folder kind
func (h *Handler) findSystemFolderID(ctx context.Context, userID shared.GlobalID, kind constants.FolderID) int64 {
	list, err := h.mailService.ListFolders(ctx, userID)
	if err != nil {
		return 0
	}
	for _, f := range list {
		if f.Kind == kind {
			return f.ID
		}
	}
	return 0
}

func (h *Handler) writeMailErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, mailexception.ErrNotFound):
		response.NotFound(c, appi18n.Message(c, appi18n.KeyMailNotFound))
	case errors.Is(err, mailexception.ErrForbidden):
		response.ErrorWithStatus(c, 403, 403, appi18n.Message(c, appi18n.KeyMailForbidden))
	case errors.Is(err, mailexception.ErrInvalidArgument):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidArgument))
	case errors.Is(err, mailexception.ErrPurgeNotAllowed):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailPurgeOrder))
	case errors.Is(err, mailexception.ErrFolderSystem):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailFolderSystem))
	case errors.Is(err, mailexception.ErrFolderNotEmpty):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailFolderNotEmpty))
	case errors.Is(err, mailexception.ErrAlreadyExists):
		response.ErrorWithStatus(c, http.StatusConflict, 409, "folder already exists")
	case errors.Is(err, mailexception.ErrComposeNoRecipient):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailRecipientRequired))
	case errors.Is(err, mailexception.ErrComposeExternalRecipient):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailLocalOnly))
	case errors.Is(err, mailexception.ErrComposeAddress):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailInvalidEmail))
	case errors.Is(err, mailexception.ErrComposeEmptyBody):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailBodyEmpty))
	case errors.Is(err, mailexception.ErrComposeSMTPConnect):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailSMTPConnectFailed))
	case errors.Is(err, mailexception.ErrComposeSMTPSend):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailSMTPSendFailed))
	case errors.Is(err, mailexception.ErrComposeRecipientNotFound):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailRecipientNotFound))
	case errors.Is(err, mailexception.ErrComposeSaveFailed):
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailSaveFailed))
	case errors.Is(err, mailexception.ErrComposeWriteFile):
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailWriteFileFailed))
	default:
		log.Printf("webmail handler error: %+v", err)
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
	}
}

// EditDraft gets a draft by ID.
func (h *Handler) EditDraft(c *gin.Context) {
	uid, ok := middleware.MailUserID(c)
	if !ok || uid == "" {
		response.Unauthorized(c, "")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidDraftID))
		return
	}

	ctx := c.Request.Context()

	// Get draft from service
	draft, err := h.mailService.GetMessage(ctx, uid, id)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Load attachments
	var attachments []AttachmentInput
	if atts, err := h.mailService.ListMessageAttachments(ctx, uid, id); err == nil && len(atts) > 0 {
		attachments = make([]AttachmentInput, len(atts))
		for i, a := range atts {
			// For drafts, we need to read the attachment data
			if rc, _, _, _, readErr := h.mailService.OpenRawAttachment(ctx, uid, id, a.Index); readErr == nil {
				defer rc.Close()
				data, _ := io.ReadAll(rc)
				attachments[i] = AttachmentInput{
					Name:   a.Name,
					Size:   a.Size,
					Base64: base64Encode(data),
				}
			}
		}
	}

	response.Success(c, DraftDetail{
		ID:          draft.ID,
		Subject:     draft.Subject,
		Text:        draft.Body,
		To:          parseAddresses(draft.Recipient),
		Cc:          parseAddresses(draft.CarbonCopy),
		Bcc:         parseAddresses(draft.BlindCopy),
		MailTime:    draft.MailTime,
		FolderID:    draft.FolderID,
		Attachments: attachments,
	})
}

// UpdateDraft updates a draft.
func (h *Handler) UpdateDraft(c *gin.Context) {
	uid, ok := middleware.MailUserID(c)
	if !ok || uid == "" {
		response.Unauthorized(c, "")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidDraftID))
		return
	}

	var req ComposeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidRequestBody))
		return
	}

	ctx := c.Request.Context()

	// Default to Draft folder if not specified
	folderID := req.FolderID
	if folderID == nil || *folderID == 0 {
		defaultFolderID := int64(constants.Draft)
		folderID = &defaultFolderID
	}

	// Validate folder
	if !h.folderAllowed(c, uid, *folderID) {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
		return
	}

	// Convert attachments
	attachments := make([]messaging.Attachment, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attachments = append(attachments, messaging.Attachment{
			Name:        att.Name,
			ContentType: "application/octet-stream",
			Data:        base64Decode(att.Base64),
		})
	}

	// Determine sender email
	senderEmail := c.GetString("email")
	if req.From != nil && req.From.Email != "" {
		senderEmail = req.From.Email
	}

	// Convert handler Recipient -> service Recipient
	toRecipients := toServiceRecipients(req.To)
	ccRecipients := toServiceRecipients(req.Cc)
	var fromRecipient *mailservice.Recipient
	if req.From != nil {
		fromRecipient = &mailservice.Recipient{
			Name:  req.From.Name,
			Email: req.From.Email,
		}
	}

	// Update draft
	draftID, err := h.mailService.SendCompose(ctx, uid, senderEmail, mailservice.ComposeRequest{
		ID:           id,
		Subject:      req.Subject,
		Text:         req.Text,
		HTML:         req.HTML,
		To:           recipientsToStr(req.To),
		Cc:           recipientsToStr(req.Cc),
		Bcc:          recipientsToStr(req.Bcc),
		From:         fromRecipient,
		ToRecipients: toRecipients,
		CcRecipients: ccRecipients,
		Attachments:  attachments,
		FolderID:     *folderID,
		SaveSent:     false,
	})
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	response.Success(c, gin.H{"id": draftID})
}

// DeleteDraft deletes a draft by ID.
func (h *Handler) DeleteDraft(c *gin.Context) {
	response.Success(c, gin.H{"ok": true})
}

// GetEmailList handles GET /api/email - returns emails with folder filter
func (h *Handler) GetEmailList(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	// Parse query parameters
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("pageSize"), 20)
	folderID, _ := strconv.ParseInt(c.Query("folderId"), 10, 64)
	keyword := c.Query("keyword")
	labelID, _ := strconv.ParseInt(c.Query("labelId"), 10, 64)

	// If no folder specified, default to inbox
	if folderID <= 0 {
		folderID = h.findSystemFolderID(c.Request.Context(), aid, constants.Inbox)
		if folderID == 0 {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyMailInboxNotFound))
			return
		}
	}

	// Validate that the folder exists and belongs to the user
	if !h.mailService.ValidateFolder(c.Request.Context(), aid, folderID) {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
		return
	}

	// Build query
	q := mailservice.ListQuery{
		Page:       page,
		PageSize:   pageSize,
		OrderField: c.DefaultQuery("order_field", "mail_time"),
		OrderDir:   c.DefaultQuery("order_dir", "DESC"),
		Search:     keyword,
		LabelID:    labelID,
	}

	total, _, items, err := h.mailService.ListMessages(c.Request.Context(), aid, folderID, q)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Transform to frontend-compatible format
	type emailListItem struct {
		ID             int64       `json:"id"`
		Subject        string      `json:"subject"`
		From           Recipient   `json:"from"`
		To             []Recipient `json:"to,omitempty"`
		Cc             []Recipient `json:"cc,omitempty"`
		Snippet        string      `json:"snippet"`
		MailTime       string      `json:"mailTime"`
		Seen           bool        `json:"isRead"`
		Flagged        bool        `json:"isStarred"`
		HasAttachments bool        `json:"hasAttachments"`
		FolderID       int64       `json:"folderId"`
		Labels         []LabelItem `json:"labels"`
	}

	rows := make([]emailListItem, len(items))
	emailIDs := make([]int64, len(items))
	for i, m := range items {
		emailIDs[i] = m.ID
		rows[i] = emailListItem{
			ID:      m.ID,
			Subject: m.Subject,
			From: func() Recipient {
				parsed := parseAddresses(m.Sender)
				if len(parsed) > 0 {
					return parsed[0]
				}
				return Recipient{Email: m.Sender}
			}(),
			To:             parseAddresses(m.Recipient),
			Cc:             parseAddresses(m.CarbonCopy),
			Snippet:        m.Snippet,
			MailTime:       m.MailTime,
			Seen:           m.ReadStatus != constants.UnRead,
			Flagged:        m.Flagged,
			HasAttachments: m.HasAttachments,
			FolderID:       m.FolderID,
			Labels:         []LabelItem{},
		}
	}

	// Batch load labels for all emails
	if labelMap, err := h.mailService.GetLabelsForEmails(c.Request.Context(), aid, emailIDs); err == nil {
		for i, m := range rows {
			if lbls, ok := labelMap[m.ID]; ok {
				labels := make([]LabelItem, len(lbls))
				for j, l := range lbls {
					labels[j] = LabelItem{ID: l.ID, Name: l.Name, Color: l.Color}
				}
				rows[i].Labels = labels
			}
		}
	}

	meta := types.NewPaginationMeta(q.Page, q.PageSize, total)
	response.SuccessWithPagination(c, gin.H{"items": rows, "total": total}, meta)
}

// SearchEmail handles GET /api/email/search
func (h *Handler) SearchEmail(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	keyword := c.Query("keyword")
	if keyword == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailKeywordRequired))
		return
	}

	folder := c.Query("folder")
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("pageSize"), 20)

	// Find folder ID
	var folderID int64 = 0
	if folder != "" {
		// Check if it's a custom folder format "folder/101"
		if strings.HasPrefix(folder, "folder/") {
			// Extract folder ID from "folder/101"
			folderIDStr := strings.TrimPrefix(folder, "folder/")
			var err error
			folderID, err = strconv.ParseInt(folderIDStr, 10, 64)
			if err != nil || folderID <= 0 {
				response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidFolderID))
				return
			}
			// Validate that the folder exists and belongs to the user
			if !h.mailService.ValidateFolder(c.Request.Context(), aid, folderID) {
				response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
				return
			}
		} else {
			// It's a system folder name
			folderID = h.findSystemFolderIDByName(c.Request.Context(), aid, folder)
		}
	}

	// Default to inbox if no folder specified
	if folderID == 0 {
		folderID = h.findSystemFolderID(c.Request.Context(), aid, constants.Inbox)
	}

	q := mailservice.ListQuery{
		Page:       page,
		PageSize:   pageSize,
		OrderField: "mail_time",
		OrderDir:   "DESC",
		Search:     keyword,
	}

	total, _, items, err := h.mailService.ListMessages(c.Request.Context(), aid, folderID, q)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Transform response
	type emailListItem struct {
		ID             int64  `json:"id"`
		Subject        string `json:"subject"`
		From           string `json:"from"`
		MailTime       string `json:"mailTime"`
		Seen           bool   `json:"isRead"`
		Flagged        bool   `json:"isStarred"`
		HasAttachments bool   `json:"hasAttachments"`
	}

	rows := make([]emailListItem, len(items))
	for i, m := range items {
		rows[i] = emailListItem{
			ID:             m.ID,
			Subject:        m.Subject,
			From:           m.Sender,
			MailTime:       m.MailTime,
			Seen:           m.ReadStatus != constants.UnRead,
			Flagged:        m.Flagged,
			HasAttachments: m.HasAttachments,
		}
	}

	meta := types.NewPaginationMeta(q.Page, q.PageSize, total)
	response.SuccessWithPagination(c, gin.H{"items": rows, "total": total}, meta)
}

// GetMailStats handles GET /api/email/stats
func (h *Handler) GetMailStats(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	ctx := c.Request.Context()

	// Get folders with counts (ListFolders already includes UnreadCount and TotalCount)
	folders, err := h.mailService.ListFolders(ctx, aid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Map to frontend format
	type MailStats struct {
		InboxCount   int   `json:"inboxCount"`
		UnreadCount  int   `json:"unreadCount"`
		SentCount    int   `json:"sentCount"`
		DraftCount   int   `json:"draftCount"`
		TrashCount   int   `json:"trashCount"`
		SpamCount    int   `json:"spamCount"`
		StorageUsed  int64 `json:"storageUsed"`
		StorageLimit int64 `json:"storageLimit"`
	}

	result := MailStats{}
	totalUnread := 0

	for _, f := range folders {
		switch f.Kind {
		case constants.Inbox:
			result.InboxCount = int(f.TotalCount)
		case constants.Sent:
			result.SentCount = int(f.TotalCount)
		case constants.Draft:
			result.DraftCount = int(f.TotalCount)
		case constants.Trash:
			result.TrashCount = int(f.TotalCount)
		case constants.Spam:
			result.SpamCount = int(f.TotalCount)
		}
		totalUnread += int(f.UnreadCount)
	}
	result.UnreadCount = totalUnread

	// Get storage usage
	storageUsed, err := h.mailService.GetMailUsage(ctx, aid)
	if err == nil {
		result.StorageUsed = storageUsed
	}
	// Default storage limit: 1GB
	result.StorageLimit = 1073741824

	response.Success(c, result)
}

// SaveDraft saves an email as draft
func (h *Handler) SaveDraft(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	var req ComposeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidRequestBody))
		return
	}

	ctx := c.Request.Context()

	// Default to Draft folder if not specified
	folderID := req.FolderID
	if folderID == nil || *folderID == 0 {
		defaultFolderID := int64(constants.Draft)
		folderID = &defaultFolderID
	}

	// Validate folder
	if !h.folderAllowed(c, aid, *folderID) {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
		return
	}

	// Convert attachments
	attachments := make([]messaging.Attachment, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attachments = append(attachments, messaging.Attachment{
			Name:        att.Name,
			ContentType: "application/octet-stream",
			Data:        base64Decode(att.Base64),
		})
	}

	// Determine sender email
	senderEmail := c.GetString("email")
	if req.From != nil && req.From.Email != "" {
		senderEmail = req.From.Email
	}

	// Convert handler Recipient -> service Recipient
	toRecipients := toServiceRecipients(req.To)
	ccRecipients := toServiceRecipients(req.Cc)
	var fromRecipient *mailservice.Recipient
	if req.From != nil {
		fromRecipient = &mailservice.Recipient{
			Name:  req.From.Name,
			Email: req.From.Email,
		}
	}

	// Save draft
	draftID, err := h.mailService.SendCompose(ctx, aid, senderEmail, mailservice.ComposeRequest{
		Subject:      req.Subject,
		Text:         req.Text,
		HTML:         req.HTML,
		To:           recipientsToStr(req.To),
		Cc:           recipientsToStr(req.Cc),
		Bcc:          recipientsToStr(req.Bcc),
		From:         fromRecipient,
		ToRecipients: toRecipients,
		CcRecipients: ccRecipients,
		Attachments:  attachments,
		FolderID:     *folderID,
		SaveSent:     false,
	})
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	response.Success(c, gin.H{"id": draftID})
}

// UploadAttachment uploads an attachment file
func (h *Handler) UploadAttachment(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidFile))
		return
	}

	// Get original filename
	filename := file.Filename

	// Get file size
	fileSize := file.Size

	// TODO: Save attachment to storage and return ID
	// For now, return a mock response
	response.Success(c, gin.H{
		"id":           1,
		"name":         filename,
		"size":         fileSize,
		"content_type": file.Header.Get("Content-Type"),
	})
}

// SendEmail sends an email
func (h *Handler) SendEmail(c *gin.Context) {
	log.Printf("[SendEmail] Starting email send process")

	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	var req ComposeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidRequestBody))
		return
	}

	ctx := c.Request.Context()

	// Determine sender: use "from" from request body if provided, otherwise fall back to auth context
	var senderEmail string
	if req.From != nil && req.From.Email != "" {
		senderEmail = req.From.Email
	} else {
		senderEmail = c.GetString("email")
	}
	if senderEmail == "" {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyMailSenderEmailNotFound))
		return
	}
	log.Printf("[SendEmail] Using sender email: %s", senderEmail)

	// Convert attachments
	attachments := make([]messaging.Attachment, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attachments = append(attachments, messaging.Attachment{
			Name:        att.Name,
			ContentType: "application/octet-stream",
			Data:        base64Decode(att.Base64),
		})
	}
	log.Printf("[SendEmail] Converted %d attachments", len(attachments))

	// Send email
	saveSent := true
	if req.SaveSent != nil {
		saveSent = *req.SaveSent
	}

	log.Printf("[SendEmail] Configuration:")
	log.Printf("  - SaveSent: %v", saveSent)
	log.Printf("  - DraftID: %v", req.DraftID)

	// Convert handler Recipient to service Recipient
	var fromRecipient *mailservice.Recipient
	if req.From != nil {
		fromRecipient = &mailservice.Recipient{
			Name:  req.From.Name,
			Email: req.From.Email,
		}
	}

	if _, err := h.mailService.SendCompose(ctx, aid, senderEmail, mailservice.ComposeRequest{
		Subject:      req.Subject,
		Text:         req.Text,
		HTML:         req.HTML,
		To:           recipientsToStr(req.To),
		Cc:           recipientsToStr(req.Cc),
		Bcc:          recipientsToStr(req.Bcc),
		From:         fromRecipient,
		ToRecipients: toServiceRecipients(req.To),
		CcRecipients: toServiceRecipients(req.Cc),
		Attachments:  attachments,
		SaveSent:     saveSent,
		Signature:    req.Signature,
	}); err != nil {
		log.Printf("[SendEmail] Error sending email: %v", err)
		h.writeMailErr(c, err)
		return
	}

	log.Printf("[SendEmail] Email sent successfully")

	// If draftID is provided, delete the draft
	if req.DraftID != nil && *req.DraftID > 0 {
		if err := h.mailService.PurgeMessage(ctx, aid, *req.DraftID); err != nil {
			log.Printf("[SendEmail] Failed to delete draft: %v", err)
			// Draft deletion failure does not affect email send success, only log it
		} else {
			log.Printf("[SendEmail] Draft deleted successfully")
		}
	}

	response.Success(c, gin.H{"ok": true})
}

// AttachmentInput represents an attachment input from client
type AttachmentInput struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Base64 string `json:"base64"`
}

// ReplyToEmail replies to an existing email
func (h *Handler) ReplyToEmail(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}

	var req struct {
		Text string `json:"text" binding:"required"`
		HTML string `json:"html"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidRequestBody))
		return
	}

	ctx := c.Request.Context()

	// Get original email
	original, err := h.mailService.GetMessage(ctx, aid, mid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Create reply subject
	subject := original.Subject
	if !strings.HasPrefix(strings.ToUpper(subject), "RE:") {
		subject = "RE: " + subject
	}

	// Prepare reply request
	userEmail := c.GetString("email")
	if userEmail == "" {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyMailUserEmailNotFound))
		return
	}

	replyReq := mailservice.ComposeRequest{
		Subject: subject,
		Text:    req.Text,
		HTML:    req.HTML,
		To:      original.Sender,
		Cc:      original.CarbonCopy,
	}

	// Send reply
	if _, err := h.mailService.SendCompose(ctx, aid, userEmail, replyReq); err != nil {
		h.writeMailErr(c, err)
		return
	}

	response.Success(c, gin.H{"ok": true})
}

// ForwardEmail forwards an existing email
func (h *Handler) ForwardEmail(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}

	var req struct {
		Text string `json:"text"`
		HTML string `json:"html"`
		To   string `json:"to" binding:"required"`
		Cc   string `json:"cc"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidRequestBody))
		return
	}

	ctx := c.Request.Context()

	// Get original email
	original, err := h.mailService.GetMessage(ctx, aid, mid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Create forward subject
	subject := original.Subject
	if !strings.HasPrefix(strings.ToUpper(subject), "FW:") {
		subject = "FW: " + subject
	}

	// Prepare forward request
	userEmail := c.GetString("email")
	if userEmail == "" {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyMailUserEmailNotFound))
		return
	}

	forwardReq := mailservice.ComposeRequest{
		Subject: subject,
		Text:    req.Text,
		HTML:    req.HTML,
		To:      req.To,
		Cc:      req.Cc,
	}

	// Send forward
	if _, err := h.mailService.SendCompose(ctx, aid, userEmail, forwardReq); err != nil {
		h.writeMailErr(c, err)
		return
	}

	response.Success(c, gin.H{"ok": true})
}

// MoveEmail handles POST /api/email/:id/move
func (h *Handler) MoveEmail(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}

	var req struct {
		FolderID *int64 `json:"folderId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidRequestBody))
		return
	}

	ctx := c.Request.Context()
	folderID := *req.FolderID

	// Validate that the folder exists
	if !h.mailService.ValidateFolder(ctx, aid, folderID) {
		response.NotFound(c, appi18n.Message(c, appi18n.KeyMailFolderNotFound))
		return
	}

	if err := h.mailService.MoveMessage(ctx, aid, mid, folderID); err != nil {
		h.writeMailErr(c, err)
		return
	}

	response.Success(c, gin.H{"ok": true})
}

// MarkAsRead handles PATCH /api/email/:id/read
func (h *Handler) MarkAsRead(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}

	var req struct {
		IsRead bool `json:"isRead" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidRequestBody))
		return
	}

	ctx := c.Request.Context()
	readStatus := constants.WebRead
	if !req.IsRead {
		readStatus = constants.UnRead
	}

	if err := h.mailService.MarkRead(ctx, aid, mid, readStatus); err != nil {
		h.writeMailErr(c, err)
		return
	}

	response.Success(c, gin.H{"ok": true})
}

// extractSnippet extracts a snippet from the email body
func extractSnippet(body string, maxLen int) string {
	if body == "" {
		return ""
	}
	// Remove HTML tags
	body = strings.ReplaceAll(body, "<br>", " ")
	body = strings.ReplaceAll(body, "<p>", " ")
	body = strings.ReplaceAll(body, "</p>", " ")
	body = strings.ReplaceAll(body, "<div>", " ")
	body = strings.ReplaceAll(body, "</div>", " ")

	// Get plain text
	snippet := body
	if len(snippet) > maxLen {
		snippet = snippet[:maxLen] + "..."
	}
	return snippet
}

// ToggleStar handles PATCH /api/email/:id/star
func (h *Handler) ToggleStar(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	mid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || mid <= 0 {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyMailInvalidMessageID))
		return
	}

	ctx := c.Request.Context()

	// Get current message to check starred status
	msg, err := h.mailService.GetMessage(ctx, aid, mid)
	if err != nil {
		h.writeMailErr(c, err)
		return
	}

	// Toggle starred status
	newFlagged := !msg.Flagged
	if err := h.mailService.SetMessageFlagged(ctx, aid, mid, newFlagged); err != nil {
		h.writeMailErr(c, err)
		return
	}

	response.Success(c, gin.H{"starred": newFlagged})
}

// findSystemFolderIDByName finds a system folder by its name
func (h *Handler) findSystemFolderIDByName(ctx context.Context, userID shared.GlobalID, name string) int64 {
	list, err := h.mailService.ListFolders(ctx, userID)
	if err != nil {
		return 0
	}

	nameMap := map[string]constants.FolderID{
		"inbox":      constants.Inbox,
		"sent":       constants.Sent,
		"draft":      constants.Draft,
		"drafts":     constants.Draft,
		"trash":      constants.Trash,
		"spam":       constants.Spam,
		"junk":       constants.Spam,
		"quarantine": constants.Quarantine,
	}

	kind, ok := nameMap[name]
	if !ok {
		return 0
	}

	for _, f := range list {
		if f.Kind == kind {
			return f.ID
		}
	}
	return 0
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func base64Decode(s string) []byte {
	d, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return []byte(s)
	}
	return d
}

// extractBodyFromEmail extracts HTML and text body from email using enmime
func (h *Handler) extractBodyFromEmail(ctx context.Context, userID shared.GlobalID, emailID int64) (string, error) {
	// Open raw email file
	rc, _, err := h.mailService.GetMessageRaw(ctx, userID, emailID)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	// Read raw email content
	rawEmail, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Parse with enmime
	env, err := enmime.ReadEnvelope(bytes.NewReader(rawEmail))
	if err != nil {
		log.Printf("Failed to parse email with enmime: %v", err)
		return "", err
	}

	// Prefer HTML body, fallback to text
	body := env.HTML
	if body == "" {
		body = env.Text
	}

	// Sanitize HTML to remove dangerous elements
	if body != "" {
		body = sanitizeBody(body)
	}

	return body, nil
}

// sanitizeBody sanitizes HTML body using bluemonday to remove XSS threats
func sanitizeBody(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	p := bluemonday.UGCPolicy()
	return p.Sanitize(s)
}
