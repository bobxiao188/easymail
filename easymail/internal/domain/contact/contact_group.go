// internal/domain/contact/contact_group.go - ContactGroup entity

package contact

import (
	"context"
	"errors"
	"strings"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors for ContactGroup operations

var (
	ErrGroupNotFound    = errors.New("contact: group not found")
	ErrGroupDuplicate   = errors.New("contact: duplicate group")
	ErrGroupInvalidName = errors.New("contact: invalid group name")
)

// ContactGroup entity

type ContactGroup struct {
	ID         shared.GlobalID
	MailUserID shared.GlobalID
	GroupName  string
	IsDefault  bool
	CreateTime time.Time
}

// Factory

func NewContactGroup(MailUserID shared.GlobalID, name string) (*ContactGroup, error) {
	return NewContactGroupWithDefault(MailUserID, name, false)
}

func NewContactGroupWithDefault(MailUserID shared.GlobalID, name string, isDefault bool) (*ContactGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrGroupInvalidName
	}
	return &ContactGroup{
		ID:         shared.NewGlobalID(),
		MailUserID: MailUserID,
		GroupName:  name,
		IsDefault:  isDefault,
		CreateTime: time.Now(),
	}, nil
}

// ContactGroup behavior

func (g *ContactGroup) BelongsToAccount(MailUserID shared.GlobalID) bool {
	return g.MailUserID == MailUserID
}

func (g *ContactGroup) IsDefaultGroup() bool {
	return g.IsDefault
}

func (g *ContactGroup) Rename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrGroupInvalidName
	}
	g.GroupName = name
	return nil
}

// ContactGroupRepository port

type ContactGroupRepository interface {
	Save(ctx context.Context, group *ContactGroup) error
	FindByID(ctx context.Context, id shared.GlobalID) (*ContactGroup, error)
	FindByAccountAndID(ctx context.Context, MailUserID, groupID shared.GlobalID) (*ContactGroup, error)
	FindByName(ctx context.Context, MailUserID shared.GlobalID, name string) (*ContactGroup, error)
	Delete(ctx context.Context, MailUserID, groupID shared.GlobalID) error
	ListByAccount(ctx context.Context, MailUserID shared.GlobalID) ([]ContactGroup, error)
}
