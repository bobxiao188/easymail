// Package management contains the bounded context for admin operations.
// internal/domain/management/maildomain.go - MailDomain aggregate root

package management

import (
	"context"
	"errors"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors

var (
	ErrDomainNotFound    = errors.New("mail domain not found")
	ErrDomainNotDeleted  = errors.New("mail domain is not deleted")
	ErrDomainInactive    = errors.New("mail domain is inactive")
	ErrDomainExists      = errors.New("mail domain already exists")
	ErrDomainInvalidName = errors.New("mail domain name is invalid")
)

// MailDomain aggregate root

type MailDomain struct {
	// ID is the unique identifier (UUID v4), used as the database primary key.
	ID shared.GlobalID `json:"id"`

	// Name is the domain name, e.g. "example.com", unique in the system.
	Name        string `json:"name"`
	Description string `json:"description"`

	// Active indicates whether the domain is enabled. Disabled domains reject mail.
	Active bool `json:"active"`

	// IsDeleted is the soft-delete flag.
	IsDeleted bool `json:"isDeleted"`

	// DKIM configuration
	DKIMEnabled    bool   `json:"dkimEnabled"`
	DKIMSelector   string `json:"dkimSelector"`
	DKIMPrivateKey string `json:"dkimPrivateKey"`

	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
	DeleteTime time.Time `json:"deleteTime,omitempty"`
}

// Factory

func NewMailDomain(name, description string) (*MailDomain, error) {
	if name == "" {
		return nil, ErrDomainInvalidName
	}
	now := time.Now()
	return &MailDomain{
		ID:          shared.NewGlobalID(),
		Name:        name,
		Description: description,
		Active:      true,
		CreateTime:  now,
		UpdateTime:  now,
	}, nil
}

// Domain behavior

func (d *MailDomain) Validate() bool {
	if d == nil {
		return false
	}
	return d.Active && !d.IsDeleted
}

func (d *MailDomain) Activate() {
	d.Active = true
	d.UpdateTime = time.Now()
}

func (d *MailDomain) Deactivate() {
	d.Active = false
	d.UpdateTime = time.Now()
}

func (d *MailDomain) SoftDelete() {
	d.IsDeleted = true
	d.Active = false
	d.UpdateTime = time.Now()
	d.DeleteTime = time.Now()
}

func (d *MailDomain) Rename(name string) {
	d.Name = name
	d.UpdateTime = time.Now()
}

// EnableDKIM enables DKIM signing for this domain with the given selector and private key.
func (d *MailDomain) EnableDKIM(selector, privateKey string) {
	d.DKIMEnabled = true
	d.DKIMSelector = selector
	d.DKIMPrivateKey = privateKey
	d.UpdateTime = time.Now()
}

// DisableDKIM disables DKIM signing for this domain.
func (d *MailDomain) DisableDKIM() {
	d.DKIMEnabled = false
	d.DKIMSelector = ""
	d.DKIMPrivateKey = ""
	d.UpdateTime = time.Now()
}

// HasDKIM returns true if DKIM is enabled and configured for this domain.
func (d *MailDomain) HasDKIM() bool {
	return d.DKIMEnabled && d.DKIMSelector != "" && d.DKIMPrivateKey != ""
}

// MailDomainRepository port

type MailDomainRepository interface {
	Save(ctx context.Context, domain *MailDomain) error
	FindByID(ctx context.Context, id shared.GlobalID) (*MailDomain, error)
	FindByName(ctx context.Context, name string) (*MailDomain, error)
	FindValidatedByName(ctx context.Context, name string) (*MailDomain, error)
	FindAllValidated(ctx context.Context) ([]MailDomain, error)
	Search(ctx context.Context, keyword string, page, pageSize int, includeDeleted bool) ([]MailDomain, int64, error)
	SoftDelete(ctx context.Context, id shared.GlobalID) error
	HardDelete(ctx context.Context, id shared.GlobalID) error
	ToggleActive(ctx context.Context, id shared.GlobalID) error
}
