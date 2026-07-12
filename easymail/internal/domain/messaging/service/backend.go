// Package service defines application ports for email operations.
package service

import (
	"io"
	"context"

	"easymail/internal/domain/messaging"
	"easymail/internal/domain/shared"
	"easymail/pkg/constants"
)

type FolderDTO struct {
	ID            int64 `json:"id"`
	Name          string `json:"name"`
	IMAPName      string `json:"imapName"`
	Kind          constants.FolderID `json:"kind"`
	UIDValidity   uint32 `json:"-"`
	UnreadCount   int64 `json:"unreadCount"`
	TotalCount    int64 `json:"totalCount"`
}

type MessageDTO struct {
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
	HasAttachments bool                 `json:"hasAttachments"`
}

type ListQuery struct {
	Page       int
	PageSize   int
	OrderField string
	OrderDir   string
	Search     string
	LabelID    int64 // Filter by label ID (0 means no filter)
}

// AttachmentDTO is the read model for attachments.
type AttachmentDTO struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

type Backend interface {
	// Folders
	ListFolders(ctx context.Context, userID shared.GlobalID) ([]FolderDTO, error)
	CreateFolder(ctx context.Context, userID shared.GlobalID, name string) (*FolderDTO, error)
	RenameFolder(ctx context.Context, userID shared.GlobalID, folderID int64, name string) error
	DeleteFolder(ctx context.Context, userID shared.GlobalID, folderID int64) error

	// Messages
	ListMessages(ctx context.Context, userID shared.GlobalID, folderID int64, query ListQuery) (total int64, unread int64, items []MessageDTO, err error)
	GetMessage(ctx context.Context, userID shared.GlobalID, mailID int64) (*messaging.Email, error)
	GetMessageBodyHTML(ctx context.Context, userID shared.GlobalID, mailID int64) (string, error)
	OpenMessageRaw(ctx context.Context, userID shared.GlobalID, mailID int64) (io.ReadCloser, int64, error)
	MarkRead(ctx context.Context, userID shared.GlobalID, mailID int64, status constants.ReadStatus) error
	MoveMessage(ctx context.Context, userID shared.GlobalID, mailID int64, folderID int64) error
	SetMessageFlagged(ctx context.Context, userID shared.GlobalID, mailID int64, flagged bool) error
	SetMessageDeleted(ctx context.Context, userID shared.GlobalID, mailID int64, deleted bool) error
	MoveToTrash(ctx context.Context, userID shared.GlobalID, mailID int64) error
	PurgeMessage(ctx context.Context, userID shared.GlobalID, mailID int64) error

	// Attachments
	ListMessageAttachments(ctx context.Context, userID shared.GlobalID, mailID int64) ([]AttachmentDTO, error)
	OpenAttachment(ctx context.Context, userID shared.GlobalID, mailID int64, index int) (io.ReadCloser, string, string, int64, error)
	GetAttachmentsZip(ctx context.Context, userID shared.GlobalID, mailID int64) ([]byte, string, error)

	// Batch operations
	BatchMoveToTrash(ctx context.Context, userID shared.GlobalID, mailIDs []int64) error
	BatchMove(ctx context.Context, userID shared.GlobalID, mailIDs []int64, folderID int64) error
	BatchMarkRead(ctx context.Context, userID shared.GlobalID, mailIDs []int64, read bool) error
	BatchSetFlagged(ctx context.Context, userID shared.GlobalID, mailIDs []int64, flagged bool) error

		// SaveRawMessage persists a new email message into a folder, returning the assigned ID.
	SaveMessage(ctx context.Context, email *messaging.Email) (int64, error)

	// UpdateMessage updates an existing email message in-place.
	UpdateMessage(ctx context.Context, email *messaging.Email) error

	// Compose
	SendCompose(ctx context.Context, userID shared.GlobalID, email string, req ComposeRequest) (int64, error)


	// Labels
	ListLabels(ctx context.Context, userID shared.GlobalID) ([]shared.LabelDTO, error)
	CreateLabel(ctx context.Context, userID shared.GlobalID, name, color string) (*shared.LabelDTO, error)
	UpdateLabel(ctx context.Context, userID shared.GlobalID, labelID int64, name, color string) error
	DeleteLabel(ctx context.Context, userID shared.GlobalID, labelID int64) error
	SetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64, labelIDs []int64) error
	GetEmailLabels(ctx context.Context, userID shared.GlobalID, emailID int64) ([]shared.LabelDTO, error)
	GetLabelsForEmails(ctx context.Context, userID shared.GlobalID, emailIDs []int64) (map[int64][]shared.LabelDTO, error)
	// ListAllMailIDs returns all mail IDs in a folder, ordered ascending
	ListAllMailIDs(ctx context.Context, userID shared.GlobalID, folderID int64) ([]int64, error)
	
	// GetMailUsage returns the total storage used by a user (sum of all mail sizes)
	GetMailUsage(ctx context.Context, userID shared.GlobalID) (int64, error)
}

// DKIMSigner signs an email with DKIM-Signature header.
// It modifies the email bytes in place, prepending the DKIM-Signature header.
type DKIMSigner interface {
	// Sign signs the given email bytes using the specified domain's DKIM key.
	// The email is modified in place with the DKIM-Signature header prepended.
	Sign(ctx context.Context, email *[]byte, domain string) error
}

// MailUserFinder looks up a mail user ID by full email address.
type MailUserFinder interface {
	FindByFullEmail(ctx context.Context, email string) (shared.GlobalID, error)
}

// MailUserFinderFunc is a function adapter for MailUserFinder.
type MailUserFinderFunc func(ctx context.Context, email string) (shared.GlobalID, error)

func (f MailUserFinderFunc) FindByFullEmail(ctx context.Context, email string) (shared.GlobalID, error) {
	return f(ctx, email)
}

func IMAPNameForKind(kind constants.FolderID) string {
	switch kind {
	case constants.Inbox:
		return "INBOX"
	case constants.Sent:
		return "Sent"
	case constants.Draft:
		return "Drafts"
	case constants.Trash:
		return "Trash"
	case constants.Spam:
		return "Junk"
	case constants.Quarantine:
		return "Quarantine"
	default:
		return constants.DefaultFolderDisplayName(kind)
	}
}

func KindFromIMAPName(name string) (constants.FolderID, bool) {
	switch name {
	case "INBOX", "Inbox", "inbox":
		return constants.Inbox, true
	case "Sent", "sent", "SENT":
		return constants.Sent, true
	case "Drafts", "drafts", "DRAFTS", "Draft", "draft", "DRAFT":
		return constants.Draft, true
	case "Trash", "trash", "TRASH", "Deleted", "deleted", "DELETED":
		return constants.Trash, true
	case "Junk", "junk", "JUNK", "Spam", "spam", "SPAM", "Bulk", "bulk", "BULK":
		return constants.Spam, true
	case "Quarantine", "quarantine", "QUARANTINE":
		return constants.Quarantine, true
	default:
		return 0, false
	}
}
