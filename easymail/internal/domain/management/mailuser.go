// Package management contains the bounded context for admin operations.
// internal/domain/management/mailuser.go - MailUser aggregate root

package management

import (
	"context"
	"errors"
	"strings"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors for MailUser operations.
var (
	ErrMailUserNotFound      = errors.New("mail user not found")
	ErrMailUserNotDeleted    = errors.New("mail user is not deleted")
	ErrMailUserInactive      = errors.New("mail user is inactive")
	ErrMailUserExists        = errors.New("mail user already exists")
	ErrMailUserInvalidEmail  = errors.New("invalid email address")
	ErrMailUserInvalidPass   = errors.New("invalid password")
	ErrMailUserPassMismatch  = errors.New("new password must differ from old password")
	ErrMailUserDomainInvalid = errors.New("referenced domain is inactive or not found")
)

// MailUser represents a mail account (mailbox user) in the system.
// Each MailUser belongs to a MailDomain and authenticates via SMTP/IMAP/Dovecot.
type MailUser struct {
	ID               shared.GlobalID `json:"id"`
	DomainID         shared.GlobalID `json:"domainId"`
	Username         string `json:"username"`
	PasswordHash     string `json:"-"`
	Email            string `json:"email"`
	Active           bool `json:"active"`
	IsDeleted        bool `json:"isDeleted"`
	StorageQuota     int64 `json:"storageQuota"`
	DataPath         string `json:"dataPath"`
	StorageID        int    `json:"storageId"`
	PasswordExpireAt time.Time `json:"passwordExpireAt,omitempty"`
	CreateTime       time.Time `json:"createTime"`
	UpdateTime       time.Time `json:"updateTime"`
	DeleteTime       time.Time `json:"deleteTime,omitempty"`
}

// NewMailUser creates a new MailUser with pre-hashed password.
func NewMailUser(domainID shared.GlobalID, username, passwordHash, email string, storageQuota int64, dataPath string, storageID int) (*MailUser, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(passwordHash) == "" {
		return nil, ErrMailUserInvalidEmail
	}
	now := time.Now()
	return &MailUser{
		ID:          shared.NewGlobalID(),
		DomainID:    domainID,
		Username:    strings.ToLower(username),
		PasswordHash: passwordHash,
		Email:       strings.ToLower(email),
		Active:      true,
		StorageQuota: storageQuota,
		DataPath:     dataPath,
		StorageID:    storageID,
		CreateTime:  now,
		UpdateTime:  now,
	}, nil
}

func (u *MailUser) FullEmail() string { return u.Email }

func (u *MailUser) Validate() bool {
	if u == nil { return false }
	return u.Active && !u.IsDeleted
}

func (u *MailUser) VerifyPassword(password string) bool {
	if u == nil || u.PasswordHash == "" { return false }
	return shared.Verify(u.PasswordHash, password) == nil
}

func (u *MailUser) IsPasswordExpired() bool {
	if u.PasswordExpireAt.IsZero() { return false }
	return time.Now().After(u.PasswordExpireAt)
}

func (u *MailUser) Activate()     { u.Active = true; u.UpdateTime = time.Now() }
func (u *MailUser) Deactivate()   { u.Active = false; u.UpdateTime = time.Now() }
func (u *MailUser) SoftDelete()   { u.IsDeleted = true; u.Active = false; u.UpdateTime = time.Now(); u.DeleteTime = time.Now() }
func (u *MailUser) SetPasswordHash(hash string) { u.PasswordHash = hash; u.UpdateTime = time.Now() }

// MailUserRepository port
type MailUserRepository interface {
	Save(ctx context.Context, user *MailUser) error
	FindByID(ctx context.Context, id shared.GlobalID) (*MailUser, error)
	FindByFullEmail(ctx context.Context, email string) (*MailUser, error)
	FindByUsername(ctx context.Context, domainID shared.GlobalID, username string) (*MailUser, error)
	Search(ctx context.Context, domainID shared.GlobalID, keyword string, status int, page, pageSize int) ([]MailUser, int64, error)
	FindByDomainID(ctx context.Context, domainID shared.GlobalID) ([]MailUser, error)
	SoftDelete(ctx context.Context, id shared.GlobalID) error
	HardDelete(ctx context.Context, id shared.GlobalID) error
	ToggleActive(ctx context.Context, id shared.GlobalID) error
	UpdatePassword(ctx context.Context, id shared.GlobalID, hash string) error
	ChangePassword(ctx context.Context, id shared.GlobalID, oldPassword, newPassword string) error
}


