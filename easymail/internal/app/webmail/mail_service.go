/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package webmail

import (
	"context"
	"io"

	"easymail/internal/domain/messaging"
	mailservice "easymail/internal/domain/messaging/service"
	"easymail/internal/domain/shared"
	"easymail/pkg/constants"
)

// MailService handles mail operations (folders, messages, compose).
type MailService interface {
	ListFolders(ctx context.Context, userID shared.GlobalID) ([]mailservice.FolderDTO, error)
	CreateFolder(ctx context.Context, userID shared.GlobalID, name string) (*mailservice.FolderDTO, error)
	RenameFolder(ctx context.Context, userID shared.GlobalID, folderID int64, name string) error
	DeleteFolder(ctx context.Context, userID shared.GlobalID, folderID int64) error

	ListMessages(ctx context.Context, userID shared.GlobalID, folderID int64, query mailservice.ListQuery) (total int64, unread int64, items []mailservice.MessageDTO, err error)
	GetMessage(ctx context.Context, userID shared.GlobalID, mailID int64) (*messaging.Email, error)
	GetMessageBody(ctx context.Context, userID shared.GlobalID, mailID int64) (string, error)
	GetMessageRaw(ctx context.Context, userID shared.GlobalID, mailID int64) (io.ReadCloser, int64, error)
	OpenRawAttachment(ctx context.Context, userID shared.GlobalID, mailID int64, index int) (io.ReadCloser, string, string, int64, error)
	ListMessageAttachments(ctx context.Context, userID shared.GlobalID, mailID int64) ([]mailservice.AttachmentDTO, error)

	OpenAttachmentsZip(ctx context.Context, userID shared.GlobalID, mailID int64) ([]byte, string, error)
	// Single-message operations
	MarkRead(ctx context.Context, userID shared.GlobalID, mailID int64, status constants.ReadStatus) error
	MoveMessage(ctx context.Context, userID shared.GlobalID, mailID int64, folderID int64) error
	SetMessageFlagged(ctx context.Context, userID shared.GlobalID, mailID int64, flagged bool) error
	MoveToTrash(ctx context.Context, userID shared.GlobalID, mailID int64) error

	// Batches
	SendCompose(ctx context.Context, userID shared.GlobalID, email string, req mailservice.ComposeRequest) (int64, error)
	BatchMoveToTrash(ctx context.Context, userID shared.GlobalID, mailIDs []int64) error
	BatchMove(ctx context.Context, userID shared.GlobalID, mailIDs []int64, folderID int64) error
	BatchMarkRead(ctx context.Context, userID shared.GlobalID, mailIDs []int64, read bool) error
	BatchSetFlagged(ctx context.Context, userID shared.GlobalID, mailIDs []int64, flagged bool) error
	PurgeMessage(ctx context.Context, userID shared.GlobalID, mailID int64) error

	// UpdateMessage updates an existing email message in-place.
	UpdateMessage(ctx context.Context, email *messaging.Email) error

	// Labels
	ListLabels(ctx context.Context, userID shared.GlobalID) ([]shared.LabelDTO, error)
	CreateLabel(ctx context.Context, userID shared.GlobalID, name, color string) (*shared.LabelDTO, error)
	UpdateLabel(ctx context.Context, userID shared.GlobalID, labelID int64, name, color string) error
	DeleteLabel(ctx context.Context, userID shared.GlobalID, labelID int64) error
	SetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64, labelIDs []int64) error
	GetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64) ([]shared.LabelDTO, error)
	GetLabelsForEmails(ctx context.Context, userID shared.GlobalID, emailIDs []int64) (map[int64][]shared.LabelDTO, error)

	// Storage
	GetMailUsage(ctx context.Context, userID shared.GlobalID) (int64, error)

	ValidateFolder(ctx context.Context, userID shared.GlobalID, folderID int64) bool
}
