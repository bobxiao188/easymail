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
	"sync"

	"easymail/internal/domain/messaging"
	mailservice "easymail/internal/domain/messaging/service"
	"easymail/internal/domain/shared"
	"easymail/pkg/constants"
)

type mailServiceImpl struct {
	backend     mailservice.Backend
	folderCache map[string][]mailservice.FolderDTO
	folderMu    sync.Mutex
}

func NewMailService(backend mailservice.Backend) MailService {
	return &mailServiceImpl{
		backend:     backend,
		folderCache: make(map[string][]mailservice.FolderDTO),
	}
}

func (s *mailServiceImpl) loadFolders(ctx context.Context, userID shared.GlobalID) ([]mailservice.FolderDTO, error) {
	key := string(userID)
	s.folderMu.Lock()
	cached, ok := s.folderCache[key]
	s.folderMu.Unlock()
	if ok {
		return cached, nil
	}
	folders, err := s.backend.ListFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.folderMu.Lock()
	s.folderCache[key] = folders
	s.folderMu.Unlock()
	return folders, nil
}

func (s *mailServiceImpl) invalidateFolderCache(userID shared.GlobalID) {
	s.folderMu.Lock()
	delete(s.folderCache, string(userID))
	s.folderMu.Unlock()
}

func (s *mailServiceImpl) ValidateFolder(ctx context.Context, userID shared.GlobalID, folderID int64) bool {
	folders, err := s.loadFolders(ctx, userID)
	if err != nil {
		return false
	}
	for _, f := range folders {
		if f.ID == folderID {
			return true
		}
	}
	return false
}

func (s *mailServiceImpl) ListFolders(ctx context.Context, userID shared.GlobalID) ([]mailservice.FolderDTO, error) {
	return s.loadFolders(ctx, userID)
}

func (s *mailServiceImpl) CreateFolder(ctx context.Context, userID shared.GlobalID, name string) (*mailservice.FolderDTO, error) {
	s.invalidateFolderCache(userID)
	return s.backend.CreateFolder(ctx, userID, name)
}

func (s *mailServiceImpl) RenameFolder(ctx context.Context, userID shared.GlobalID, folderID int64, name string) error {
	s.invalidateFolderCache(userID)
	return s.backend.RenameFolder(ctx, userID, folderID, name)
}

func (s *mailServiceImpl) DeleteFolder(ctx context.Context, userID shared.GlobalID, folderID int64) error {
	s.invalidateFolderCache(userID)
	return s.backend.DeleteFolder(ctx, userID, folderID)
}

func (s *mailServiceImpl) ListMessages(ctx context.Context, userID shared.GlobalID, folderID int64, query mailservice.ListQuery) (int64, int64, []mailservice.MessageDTO, error) {
	return s.backend.ListMessages(ctx, userID, folderID, query)
}

func (s *mailServiceImpl) GetMessage(ctx context.Context, userID shared.GlobalID, mailID int64) (*messaging.Email, error) {
	return s.backend.GetMessage(ctx, userID, mailID)
}

func (s *mailServiceImpl) GetMessageBody(ctx context.Context, userID shared.GlobalID, mailID int64) (string, error) {
	return s.backend.GetMessageBodyHTML(ctx, userID, mailID)
}

func (s *mailServiceImpl) GetMessageRaw(ctx context.Context, userID shared.GlobalID, mailID int64) (io.ReadCloser, int64, error) {
	return s.backend.OpenMessageRaw(ctx, userID, mailID)
}

func (s *mailServiceImpl) OpenRawAttachment(ctx context.Context, userID shared.GlobalID, mailID int64, index int) (io.ReadCloser, string, string, int64, error) {
	return s.backend.OpenAttachment(ctx, userID, mailID, index)
}

func (s *mailServiceImpl) ListMessageAttachments(ctx context.Context, userID shared.GlobalID, mailID int64) ([]mailservice.AttachmentDTO, error) {
	return s.backend.ListMessageAttachments(ctx, userID, mailID)
}

// ---- Single-message convenience methods ----

func (s *mailServiceImpl) OpenAttachmentsZip(ctx context.Context, userID shared.GlobalID, mailID int64) ([]byte, string, error) {
	return s.backend.GetAttachmentsZip(ctx, userID, mailID)
}

func (s *mailServiceImpl) MarkRead(ctx context.Context, userID shared.GlobalID, mailID int64, status constants.ReadStatus) error {
	// Invalidate folder cache so unread count is recalculated
	s.invalidateFolderCache(userID)
	return s.backend.MarkRead(ctx, userID, mailID, status)
}

func (s *mailServiceImpl) MoveMessage(ctx context.Context, userID shared.GlobalID, mailID int64, folderID int64) error {
	// Invalidate folder cache so folder counts are recalculated
	s.invalidateFolderCache(userID)
	return s.backend.MoveMessage(ctx, userID, mailID, folderID)
}

func (s *mailServiceImpl) SetMessageFlagged(ctx context.Context, userID shared.GlobalID, mailID int64, flagged bool) error {
	return s.backend.SetMessageFlagged(ctx, userID, mailID, flagged)
}

func (s *mailServiceImpl) MoveToTrash(ctx context.Context, userID shared.GlobalID, mailID int64) error {
	// Invalidate folder cache so folder counts are recalculated
	s.invalidateFolderCache(userID)
	return s.backend.MoveToTrash(ctx, userID, mailID)
}

// ---- Compose ----

func (s *mailServiceImpl) SendCompose(ctx context.Context, userID shared.GlobalID, email string, req mailservice.ComposeRequest) (int64, error) {
	// Invalidate folder cache so folder counts are recalculated
	s.invalidateFolderCache(userID)
	return s.backend.SendCompose(ctx, userID, email, req)
}

// ---- Batch operations ----

func (s *mailServiceImpl) BatchMoveToTrash(ctx context.Context, userID shared.GlobalID, mailIDs []int64) error {
	// Invalidate folder cache so folder counts are recalculated
	s.invalidateFolderCache(userID)
	return s.backend.BatchMoveToTrash(ctx, userID, mailIDs)
}

func (s *mailServiceImpl) BatchMove(ctx context.Context, userID shared.GlobalID, mailIDs []int64, folderID int64) error {
	// Invalidate folder cache so folder counts are recalculated
	s.invalidateFolderCache(userID)
	return s.backend.BatchMove(ctx, userID, mailIDs, folderID)
}

func (s *mailServiceImpl) BatchMarkRead(ctx context.Context, userID shared.GlobalID, mailIDs []int64, read bool) error {
	// Invalidate folder cache so unread count is recalculated
	s.invalidateFolderCache(userID)
	return s.backend.BatchMarkRead(ctx, userID, mailIDs, read)
}

func (s *mailServiceImpl) BatchSetFlagged(ctx context.Context, userID shared.GlobalID, mailIDs []int64, flagged bool) error {
	return s.backend.BatchSetFlagged(ctx, userID, mailIDs, flagged)
}


// --- Label operations ---

func (s *mailServiceImpl) ListLabels(ctx context.Context, userID shared.GlobalID) ([]shared.LabelDTO, error) {
	return s.backend.ListLabels(ctx, userID)
}

func (s *mailServiceImpl) CreateLabel(ctx context.Context, userID shared.GlobalID, name, color string) (*shared.LabelDTO, error) {
	return s.backend.CreateLabel(ctx, userID, name, color)
}

func (s *mailServiceImpl) UpdateLabel(ctx context.Context, userID shared.GlobalID, labelID int64, name, color string) error {
	return s.backend.UpdateLabel(ctx, userID, labelID, name, color)
}

func (s *mailServiceImpl) DeleteLabel(ctx context.Context, userID shared.GlobalID, labelID int64) error {
	return s.backend.DeleteLabel(ctx, userID, labelID)
}

func (s *mailServiceImpl) SetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64, labelIDs []int64) error {
	return s.backend.SetEmailLabels(ctx, userID, emailID, labelIDs)
}

func (s *mailServiceImpl) GetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64) ([]shared.LabelDTO, error) {
	return s.backend.GetEmailLabels(ctx, userID, emailID)
}

func (s *mailServiceImpl) GetLabelsForEmails(ctx context.Context, userID shared.GlobalID, emailIDs []int64) (map[int64][]shared.LabelDTO, error) {
	return s.backend.GetLabelsForEmails(ctx, userID, emailIDs)
}

// ---- Storage ----

func (s *mailServiceImpl) GetMailUsage(ctx context.Context, userID shared.GlobalID) (int64, error) {
	return s.backend.GetMailUsage(ctx, userID)
}

func (s *mailServiceImpl) PurgeMessage(ctx context.Context, userID shared.GlobalID, mailID int64) error {
	// Invalidate folder cache so folder counts are recalculated
	s.invalidateFolderCache(userID)
	return s.backend.PurgeMessage(ctx, userID, mailID)
}

func (s *mailServiceImpl) UpdateMessage(ctx context.Context, email *messaging.Email) error {
	return s.backend.UpdateMessage(ctx, email)
}
