package repository

import (
	"context"

	"easymail/internal/domain/messaging"
	"easymail/internal/domain/shared"
	"easymail/pkg/constants"
)

// FolderCounts holds aggregate counts for a single folder.
type FolderCounts struct {
	Total  int64
	Unread int64
}

// EmailRepository defines data access for emails.
type EmailRepository interface {
	Save(ctx context.Context, msg *messaging.Email) error
	GetMailQuantity(ctx context.Context, accID shared.GlobalID) (int64, error)
	GetMailUsage(ctx context.Context, accID shared.GlobalID) (int64, error)
	MarkRead(ctx context.Context, accID shared.GlobalID, mailID int64, readSource constants.ReadStatus) error
	GetMail(ctx context.Context, accID shared.GlobalID, mailID int64) (*messaging.Email, error)
	DeleteMail(ctx context.Context, accID shared.GlobalID, mailID int64) error
	HardDeleteMail(ctx context.Context, accID shared.GlobalID, mailID int64) error
	MoveMail(ctx context.Context, accID shared.GlobalID, mailID int64, folderID int64) error
	CountActiveInFolder(ctx context.Context, accID shared.GlobalID, folderID int64) (int64, error)
	CountUnreadInFolder(ctx context.Context, accID shared.GlobalID, folderID int64) (int64, error)
	CountByFolder(ctx context.Context, accID shared.GlobalID) (map[int64]FolderCounts, error)
	ListAllIDs(ctx context.Context, accID shared.GlobalID, folderID int64) ([]int64, error)
	SetFlagged(ctx context.Context, accID shared.GlobalID, mailID int64, flagged bool) error
	SetDeleted(ctx context.Context, accID shared.GlobalID, mailID int64, deleted bool) error
	QueryByFolder(ctx context.Context, accID shared.GlobalID, folderID int64, orderField, orderDir string, page, pageSize int, search string, labelID int64) (total int64, unread int64, emails []messaging.Email, err error)

	GetFolderNumericID(ctx context.Context, accID shared.GlobalID, globalID string) (int64, error)
	SetFolderNumericID(ctx context.Context, accID shared.GlobalID, globalID string, numericID int64) error
	GetGlobalIDByNumericID(ctx context.Context, accID shared.GlobalID, numericID int64) (string, error)

	GetNextCustomFolderID(ctx context.Context, accID shared.GlobalID) (int64, error)
	// Labels
	ListLabels(ctx context.Context, accID shared.GlobalID) ([]shared.LabelDTO, error)
	CreateLabel(ctx context.Context, accID shared.GlobalID, name, color string) (*shared.LabelDTO, error)
	UpdateLabel(ctx context.Context, accID shared.GlobalID, labelID int64, name, color string) error
	DeleteLabel(ctx context.Context, accID shared.GlobalID, labelID int64) error
	SetEmailLabels(ctx context.Context, accID shared.GlobalID, emailID int64, labelIDs []int64) error
	GetEmailLabels(ctx context.Context, accID shared.GlobalID, emailID int64) ([]shared.LabelDTO, error)
	GetLabelsForEmails(ctx context.Context, accID shared.GlobalID, emailIDs []int64) (map[int64][]shared.LabelDTO, error)

	// SMTP delivery status
	UpdateSMTPStatus(ctx context.Context, accID shared.GlobalID, emailID int64, status, errMsg, sentAt string) error
}
