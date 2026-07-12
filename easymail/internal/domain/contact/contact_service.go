// internal/domain/contact/contact_service.go - Contact domain service

package contact

import (
	"context"

	"easymail/internal/domain/shared"
)

// ContactService defines the use cases for contact and group management.
type ContactService interface {
	// Group operations
	CreateGroup(ctx context.Context, MailUserID shared.GlobalID, name string) (*ContactGroup, error)
	RenameGroup(ctx context.Context, MailUserID, groupID shared.GlobalID, name string) error
	DeleteGroup(ctx context.Context, MailUserID, groupID shared.GlobalID) error
	ListGroups(ctx context.Context, MailUserID shared.GlobalID) ([]ContactGroup, error)

	// Contact operations
	CreateContact(ctx context.Context, MailUserID shared.GlobalID, name, email string) (*Contact, error)
	UpdateContactEmail(ctx context.Context, MailUserID, contactID shared.GlobalID, email string) error
	DeleteContact(ctx context.Context, MailUserID, contactID shared.GlobalID) error
	GetContact(ctx context.Context, MailUserID, contactID shared.GlobalID) (*Contact, error)
	ListContacts(ctx context.Context, MailUserID shared.GlobalID, keyword string, groupID *shared.GlobalID, ungrouped bool) ([]Contact, error)
}

type contactServiceImpl struct {
	contactRepo ContactRepository
	groupRepo   ContactGroupRepository
}

func NewContactService(contactRepo ContactRepository, groupRepo ContactGroupRepository) ContactService {
	return &contactServiceImpl{
		contactRepo: contactRepo,
		groupRepo:   groupRepo,
	}
}

func (s *contactServiceImpl) CreateGroup(ctx context.Context, MailUserID shared.GlobalID, name string) (*ContactGroup, error) {
	g, err := NewContactGroup(MailUserID, name)
	if err != nil {
		return nil, err
	}
	if err := s.groupRepo.Save(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *contactServiceImpl) RenameGroup(ctx context.Context, MailUserID, groupID shared.GlobalID, name string) error {
	g, err := s.groupRepo.FindByAccountAndID(ctx, MailUserID, groupID)
	if err != nil {
		return err
	}
	return g.Rename(name)
}

func (s *contactServiceImpl) DeleteGroup(ctx context.Context, MailUserID, groupID shared.GlobalID) error {
	return s.groupRepo.Delete(ctx, MailUserID, groupID)
}

func (s *contactServiceImpl) ListGroups(ctx context.Context, MailUserID shared.GlobalID) ([]ContactGroup, error) {
	return s.groupRepo.ListByAccount(ctx, MailUserID)
}

func (s *contactServiceImpl) CreateContact(ctx context.Context, MailUserID shared.GlobalID, name, email string) (*Contact, error) {
	c, err := NewContact(MailUserID, name, email)
	if err != nil {
		return nil, err
	}
	if err := s.contactRepo.Save(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *contactServiceImpl) UpdateContactEmail(ctx context.Context, MailUserID, contactID shared.GlobalID, email string) error {
	c, err := s.contactRepo.FindByAccountAndID(ctx, MailUserID, contactID)
	if err != nil {
		return err
	}
	return c.UpdateEmail(email)
}

func (s *contactServiceImpl) DeleteContact(ctx context.Context, MailUserID, contactID shared.GlobalID) error {
	return s.contactRepo.Delete(ctx, MailUserID, contactID)
}

func (s *contactServiceImpl) GetContact(ctx context.Context, MailUserID, contactID shared.GlobalID) (*Contact, error) {
	return s.contactRepo.FindByAccountAndID(ctx, MailUserID, contactID)
}

func (s *contactServiceImpl) ListContacts(ctx context.Context, MailUserID shared.GlobalID, keyword string, groupID *shared.GlobalID, ungrouped bool) ([]Contact, error) {
	return s.contactRepo.Search(ctx, MailUserID, keyword, groupID, ungrouped)
}

var _ ContactService = (*contactServiceImpl)(nil)
