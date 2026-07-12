// internal/app/management/maildomain_service.go
package management

import (
	"context"
	"time"

	domain "easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/pkg/logger/easylog"
)

type MailDomainService interface {
	Create(ctx context.Context, name, description string) (*domain.MailDomain, error)
	GetByID(ctx context.Context, id shared.GlobalID) (*domain.MailDomain, error)
	List(ctx context.Context, keyword string, page, pageSize int, includeDeleted bool) ([]domain.MailDomain, int64, error)
	Update(ctx context.Context, id shared.GlobalID, name, description string, active bool, isDeleted bool) error
	Delete(ctx context.Context, id shared.GlobalID) error
	ToggleActive(ctx context.Context, id shared.GlobalID) error
	UpdateWithFields(ctx context.Context, id shared.GlobalID, name *string, description *string, active *bool, isDeleted *bool) error
	UpdateDKIMSettings(ctx context.Context, id shared.GlobalID, enabled bool, selector, privateKey string) error
	// PurgeDomain physically deletes a soft-deleted domain, all its mail accounts,
	// and releases all associated data files.
	PurgeDomain(ctx context.Context, id shared.GlobalID) error
}

type mailDomainServiceImpl struct {
	repo         domain.MailDomainRepository
	mailUserRepo domain.MailUserRepository
	provisionSvc UserProvisionService
	storageRoot  string
	_log         *easylog.Logger
}

func NewMailDomainService(repo domain.MailDomainRepository, mailUserRepo domain.MailUserRepository, provisionSvc UserProvisionService, storageRoot string, logger *easylog.Logger) MailDomainService {
	return &mailDomainServiceImpl{
		repo:         repo,
		mailUserRepo: mailUserRepo,
		provisionSvc: provisionSvc,
		storageRoot:  storageRoot,
		_log:         logger.WithModule("maildomain-service"),
	}
}

func (s *mailDomainServiceImpl) Create(ctx context.Context, name, description string) (*domain.MailDomain, error) {
	d, err := domain.NewMailDomain(name, description)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *mailDomainServiceImpl) GetByID(ctx context.Context, id shared.GlobalID) (*domain.MailDomain, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *mailDomainServiceImpl) List(ctx context.Context, keyword string, page, pageSize int, includeDeleted bool) ([]domain.MailDomain, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.repo.Search(ctx, keyword, page, pageSize, includeDeleted)
}

func (s *mailDomainServiceImpl) Update(ctx context.Context, id shared.GlobalID, name, description string, active bool, isDeleted bool) error {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	d.Rename(name)
	d.Description = description
	if active {
		d.Activate()
	} else {
		d.Deactivate()
	}
	if isDeleted {
		d.SoftDelete()
	} else if d.IsDeleted {
		d.IsDeleted = false
		d.Active = true
		d.DeleteTime = time.Time{}
	}
	return s.repo.Save(ctx, d)
}

func (s *mailDomainServiceImpl) Delete(ctx context.Context, id shared.GlobalID) error {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	d.SoftDelete()
	return s.repo.Save(ctx, d)
}

func (s *mailDomainServiceImpl) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	return s.repo.ToggleActive(ctx, id)
}

func (s *mailDomainServiceImpl) UpdateWithFields(ctx context.Context, id shared.GlobalID, name *string, description *string, active *bool, isDeleted *bool) error {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if name != nil {
		d.Rename(*name)
	}
	if description != nil {
		d.Description = *description
	}
	if active != nil {
		if *active {
			d.Activate()
		} else {
			d.Deactivate()
		}
	}
	if isDeleted != nil {
		if *isDeleted {
			d.SoftDelete()
		} else if d.IsDeleted {
			d.IsDeleted = false
			d.Active = true
			d.DeleteTime = time.Time{}
		}
	}
	return s.repo.Save(ctx, d)
}

func (s *mailDomainServiceImpl) UpdateDKIMSettings(ctx context.Context, id shared.GlobalID, enabled bool, selector, privateKey string) error {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if enabled {
		d.EnableDKIM(selector, privateKey)
	} else {
		d.DisableDKIM()
	}
	return s.repo.Save(ctx, d)
}

func (s *mailDomainServiceImpl) PurgeDomain(ctx context.Context, id shared.GlobalID) error {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !d.IsDeleted {
		return domain.ErrDomainNotDeleted
	}

	// Find all mail accounts belonging to this domain
	users, err := s.mailUserRepo.FindByDomainID(ctx, id)
	if err != nil {
		return err
	}

	// Deprovision storage and hard-delete each account
	for _, u := range users {
		if s.storageRoot != "" && u.DataPath != "" && s.provisionSvc != nil {
			if err := s.provisionSvc.Deprovision(ctx, s.storageRoot, u.DataPath); err != nil {
				s._log.Warnf("purge domain %s: deprovision user %s failed: %v", id, u.ID, err)
			}
		}
		if err := s.mailUserRepo.HardDelete(ctx, u.ID); err != nil {
			s._log.Warnf("purge domain %s: hard delete user %s failed: %v", id, u.ID, err)
		}
	}

	// Hard delete the domain itself
	return s.repo.HardDelete(ctx, id)
}

var _ MailDomainService = (*mailDomainServiceImpl)(nil)