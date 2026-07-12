// Package management contains the bounded context for admin operations.
// internal/domain/management/adminuser.go - AdminUser aggregate root

package management

import (
	"context"
	"errors"
	"strings"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors for AdminUser operations.
var (
	ErrAdminUserNotFound      = errors.New("admin user not found")
	ErrAdminUserInactive      = errors.New("admin user is inactive")
	ErrAdminUserExists        = errors.New("admin user already exists")
	ErrAdminUserInvalidPass   = errors.New("invalid password")
	ErrAdminUserPassMismatch  = errors.New("new password must differ from old password")
)

// AdminUser represents an administrator account for the management console.
type AdminUser struct {
	ID           shared.GlobalID `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Nickname     string `json:"nickname"`
	Email        string `json:"email"`
	Avatar       string `json:"avatar"`
	Language     string `json:"language"`
	Skin         string `json:"skin"`
	Active       bool `json:"active"`
	IsDeleted    bool `json:"isDeleted"`
	CreateTime   time.Time `json:"createTime"`
	UpdateTime   time.Time `json:"updateTime"`
	DeleteTime   time.Time `json:"deleteTime,omitempty"`
}

// NewAdminUser creates a new AdminUser with pre-hashed password.
func NewAdminUser(username, passwordHash, nickname, email string) (*AdminUser, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(passwordHash) == "" {
		return nil, ErrAdminUserInvalidPass
	}
	now := time.Now()
	return &AdminUser{
		ID:           shared.NewGlobalID(),
		Username:     strings.ToLower(username),
		PasswordHash: passwordHash,
		Nickname:     nickname,
		Email:        email,
		Language:     "zh",
		Skin:         "dark",
		Active:       true,
		CreateTime:   now,
		UpdateTime:   now,
	}, nil
}

func (u *AdminUser) Validate() bool {
	if u == nil { return false }
	return u.Active && !u.IsDeleted
}

func (u *AdminUser) VerifyPassword(password string) bool {
	if u == nil || u.PasswordHash == "" { return false }
	return shared.Verify(u.PasswordHash, password) == nil
}

func (u *AdminUser) Activate()   { u.Active = true; u.UpdateTime = time.Now() }
func (u *AdminUser) Deactivate() { u.Active = false; u.UpdateTime = time.Now() }
func (u *AdminUser) SoftDelete() { u.IsDeleted = true; u.Active = false; u.UpdateTime = time.Now(); u.DeleteTime = time.Now() }
func (u *AdminUser) SetPasswordHash(hash string) { u.PasswordHash = hash; u.UpdateTime = time.Now() }

// AdminUserRepository port
type AdminUserRepository interface {
	Save(ctx context.Context, user *AdminUser) error
	FindByID(ctx context.Context, id shared.GlobalID) (*AdminUser, error)
	FindByUsername(ctx context.Context, username string) (*AdminUser, error)
	Search(ctx context.Context, keyword string, page, pageSize int) ([]AdminUser, int64, error)
	SoftDelete(ctx context.Context, id shared.GlobalID) error
	ToggleActive(ctx context.Context, id shared.GlobalID) error
	UpdatePassword(ctx context.Context, id shared.GlobalID, hash string) error
}


