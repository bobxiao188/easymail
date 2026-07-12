// internal/domain/mailbox/folder.go - Folder entity

package mailbox

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors for folder operations

var (
	ErrFolderNotFound     = errors.New("folder: not found")
	ErrFolderNameRequired = errors.New("folder: name is required")
	ErrFolderDuplicate    = errors.New("folder: already exists")
	ErrFolderSystem       = errors.New("folder: system folder, cannot modify or delete")
)

// FolderKind represents the type of folder (system or custom).
type FolderKind int

const (
	_                   FolderKind = 0 // 占位，从 1 开始
	FolderInbox         FolderKind = 1
	FolderSent          FolderKind = 2
	FolderDraft         FolderKind = 3
	FolderTrash         FolderKind = 4
	FolderSpam          FolderKind = 5
	FolderQuarantine    FolderKind = 6
	FolderUserCustomMin FolderKind = 100
)

func IsSystemFolderKind(kind FolderKind) bool {
	return kind >= 1 && kind < FolderUserCustomMin
}

// IMAPNameForKind maps internal folder kind to a common IMAP mailbox name (UTF-8; wire encoding in IMAP layer).
func IMAPNameForKind(kind FolderKind) string {
	switch kind {
	case FolderInbox:
		return "INBOX"
	case FolderSent:
		return "Sent"
	case FolderTrash:
		return "Trash"
	case FolderSpam:
		return "Junk"
	case FolderDraft:
		return "Drafts"
	case FolderQuarantine:
		return "Quarantine"
	default:
		return "Folder"
	}
}

// KindFromIMAPName parses common folder names case-insensitively (including INBOX).
func KindFromIMAPName(name string) (FolderKind, bool) {
	s := strings.TrimSpace(name)
	switch strings.ToUpper(s) {
	case "INBOX":
		return FolderInbox, true
	case "SENT", "SENT ITEMS":
		return FolderSent, true
	case "TRASH", "DELETED", "DELETED ITEMS":
		return FolderTrash, true
	case "JUNK", "SPAM", "BULK MAIL":
		return FolderSpam, true
	case "DRAFTS", "DRAFT":
		return FolderDraft, true
	default:
		return 0, false
	}
}

// Folder represents a mailbox folder.
type Folder struct {
	ID         shared.GlobalID
	MailUserID  shared.GlobalID
	FolderName string
	FolderKind FolderKind
	// UIDValidity is the IMAP UIDVALIDITY for this mailbox. It MUST change
	// whenever the set of UIDs is renumbered (e.g. mailbox reset), otherwise
	// IMAP clients cache stale UIDs and fail to download new messages.
	UIDValidity uint32
	CreateTime time.Time
	UpdateTime time.Time
}

// GenerateUIDValidity returns a non-zero random UIDVALIDITY value.
// RFC 3501 requires UIDVALIDITY to be > 0.
func GenerateUIDValidity() uint32 {
	v := rand.Uint32()
	if v == 0 {
		v = 1
	}
	return v
}

// Factory

func NewFolder(mailUserID shared.GlobalID, name string, kind FolderKind) (*Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrFolderNameRequired
	}
	now := time.Now()
	return &Folder{
		ID:         shared.NewGlobalID(),
		MailUserID:  mailUserID,
		FolderName: name,
		FolderKind: kind,
		CreateTime: now,
		UpdateTime: now,
	}, nil
}

// Folder behavior

func (f *Folder) Rename(name string) error {
	if IsSystemFolderKind(f.FolderKind) {
		return ErrFolderSystem
	}
	f.FolderName = strings.TrimSpace(name)
	f.UpdateTime = time.Now()
	return nil
}

func (f *Folder) BelongsToMailUser(mailUserID shared.GlobalID) bool {
	return f.MailUserID == mailUserID
}

// CanDelete checks whether the folder can be deleted.
func (f *Folder) CanDelete() error {
	if IsSystemFolderKind(f.FolderKind) {
		return ErrFolderSystem
	}
	return nil
}

// FolderRepository port

type FolderRepository interface {
	Save(ctx context.Context, folder *Folder) error
	FindByID(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) (*Folder, error)
	FindByMailUserAndKind(ctx context.Context, mailUserID shared.GlobalID, kind FolderKind) (*Folder, error)
	ListByMailUser(ctx context.Context, mailUserID shared.GlobalID) ([]*Folder, error)
	Update(ctx context.Context, folder *Folder) error
	UpdateName(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID, name string) error
	SoftDelete(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) error
}


