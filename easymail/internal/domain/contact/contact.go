// Package contact contains the bounded context for contacts.
// internal/domain/contact/contact.go - Contact aggregate root

package contact

import (
	"context"
	"errors"
	"strings"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors

var (
	ErrContactNotFound     = errors.New("contact: not found")
	ErrContactDuplicate    = errors.New("contact: duplicate")
	ErrContactInvalidEmail = errors.New("contact: invalid email address")
	ErrContactInvalidName  = errors.New("contact: invalid name")
)

// Contact aggregate root

type Contact struct {
	ID             shared.GlobalID
	MailUserID     shared.GlobalID
	ContactName    string
	ContactEmail   string
	ContactPhone   string
	ContactAddress string
	ContactCity    string
	ContactState   string
	ContactZip     string
	ContactCountry string
	CreateTime     time.Time
	ContactGroupID *shared.GlobalID
}

// Factory

func NewContact(MailUserID shared.GlobalID, name, email string) (*Contact, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" {
		return nil, ErrContactInvalidName
	}
	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrContactInvalidEmail
	}
	return &Contact{
		ID:           shared.NewGlobalID(),
		MailUserID:   MailUserID,
		ContactName:  name,
		ContactEmail: email,
		CreateTime:   time.Now(),
	}, nil
}

// Contact behavior
func (c *Contact) BelongsToAccount(MailUserID shared.GlobalID) bool {
	return c.MailUserID == MailUserID
}

func (c *Contact) UpdateName(name string) {
	c.ContactName = strings.TrimSpace(name)
}

func (c *Contact) UpdateEmail(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return ErrContactInvalidEmail
	}
	c.ContactEmail = email
	return nil
}

func (c *Contact) MoveToGroup(groupID *shared.GlobalID) {
	c.ContactGroupID = groupID
}

// ContactRepository port
type ContactRepository interface {
	Save(ctx context.Context, contact *Contact) error
	FindByID(ctx context.Context, id shared.GlobalID) (*Contact, error)
	FindByAccountAndID(ctx context.Context, MailUserID, contactID shared.GlobalID) (*Contact, error)
	FindByEmail(ctx context.Context, MailUserID shared.GlobalID, email string) (*Contact, error)
	Delete(ctx context.Context, MailUserID, contactID shared.GlobalID) error
	Search(ctx context.Context, MailUserID shared.GlobalID, keyword string, groupID *shared.GlobalID, ungrouped bool) ([]Contact, error)
	// 分页查询
	Count(ctx context.Context, MailUserID shared.GlobalID, groupID *shared.GlobalID, ungrouped bool) (int64, error)
	SearchPaged(ctx context.Context, MailUserID shared.GlobalID, keyword string, groupID *shared.GlobalID, ungrouped bool, page, pageSize int) ([]Contact, error)
}
