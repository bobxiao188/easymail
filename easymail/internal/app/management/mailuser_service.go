// internal/app/management/mailuser_service.go
package management

import (
	"context"
	"fmt"
	"strings"

	domain "easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/pkg/logger/easylog"
)

type MailUserService interface {
	Create(ctx context.Context, domainID shared.GlobalID, username, password, domainName string, storageQuota int64, dataPath string, storageID int) (*domain.MailUser, error)
	GetByID(ctx context.Context, id shared.GlobalID) (*domain.MailUser, error)
	GetByFullEmail(ctx context.Context, email string) (*domain.MailUser, error)
	List(ctx context.Context, domainID shared.GlobalID, keyword string, status int, page, pageSize int) ([]domain.MailUser, int64, error)
	Update(ctx context.Context, id shared.GlobalID, username string, active bool, storageQuota int64) error
	Delete(ctx context.Context, id shared.GlobalID) error
	SoftDelete(ctx context.Context, id shared.GlobalID) error
	PurgeDeleted(ctx context.Context, id shared.GlobalID) error
	ToggleActive(ctx context.Context, id shared.GlobalID) error
	ChangePassword(ctx context.Context, id shared.GlobalID, oldPassword, newPassword string) error
	ResetPassword(ctx context.Context, id shared.GlobalID, newPassword string) error
}

type mailUserServiceImpl struct {
	repo         domain.MailUserRepository
	storageRoot  string
	provisionSvc UserProvisionService
	_log         *easylog.Logger
}

func NewMailUserService(repo domain.MailUserRepository, storageRoot string, provisionSvc UserProvisionService, logger *easylog.Logger) MailUserService {
	return &mailUserServiceImpl{
		repo:         repo,
		storageRoot:  storageRoot,
		provisionSvc: provisionSvc,
		_log:         logger.WithModule("mailuser-service"),
	}
}

func (s *mailUserServiceImpl) Create(ctx context.Context, domainID shared.GlobalID, username, password, domainName string, storageQuota int64, dataPath string, storageID int) (*domain.MailUser, error) {
	if strings.TrimSpace(password) == "" {
		return nil, domain.ErrMailUserInvalidPass
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := shared.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("password hash failed: %w", err)
	}
	email := strings.ToLower(username) + "@" + strings.ToLower(domainName)
	user, err := domain.NewMailUser(domainID, username, hash, email, storageQuota, dataPath, storageID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *mailUserServiceImpl) GetByID(ctx context.Context, id shared.GlobalID) (*domain.MailUser, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *mailUserServiceImpl) GetByFullEmail(ctx context.Context, email string) (*domain.MailUser, error) {
	return s.repo.FindByFullEmail(ctx, email)
}

func (s *mailUserServiceImpl) List(ctx context.Context, domainID shared.GlobalID, keyword string, status int, page, pageSize int) ([]domain.MailUser, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.repo.Search(ctx, domainID, keyword, status, page, pageSize)
}

func (s *mailUserServiceImpl) Update(ctx context.Context, id shared.GlobalID, username string, active bool, storageQuota int64) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	user.Username = strings.ToLower(username)
	user.StorageQuota = storageQuota
	if active {
		user.Activate()
	} else {
		user.Deactivate()
	}
	return s.repo.Save(ctx, user)
}

func (s *mailUserServiceImpl) Delete(ctx context.Context, id shared.GlobalID) error {
	s._log.Infof("Starting delete operation for user ID: %s", id)

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s._log.Errorf("Failed to find user by ID %s: %v", id, err)
		return err
	}
	s._log.Infof("Found user: Email=%s, DataPath=%s", user.Email, user.DataPath)

	// Delete storage directory (close SQLite pool connection first)
	if s.storageRoot != "" && user.DataPath != "" {
		s._log.Infof("Deprovisioning user storage: root=%s, dataPath=%s", s.storageRoot, user.DataPath)
		if s.provisionSvc != nil {
			if err := s.provisionSvc.Deprovision(ctx, s.storageRoot, user.DataPath); err != nil {
				s._log.Warnf("Failed to deprovision storage: %v", err)
			} else {
				s._log.Infof("Successfully deprovisioned storage")
			}
		}
	} else {
		if s.storageRoot == "" {
			s._log.Infof("Skipping storage deletion: storageRoot is not configured")
		}
		if user.DataPath == "" {
			s._log.Infof("Skipping storage deletion: user DataPath is empty")
		}
	}

	s._log.Infof("Performing hard delete from MySQL for user ID: %s", id)
	if err := s.repo.HardDelete(ctx, id); err != nil {
		s._log.Errorf("Failed to hard delete user from MySQL ID %s: %v", id, err)
		return err
	}
	s._log.Infof("Successfully deleted user ID: %s, Email: %s", id, user.Email)
	return nil
}

func (s *mailUserServiceImpl) SoftDelete(ctx context.Context, id shared.GlobalID) error {
	return s.repo.SoftDelete(ctx, id)
}

func (s *mailUserServiceImpl) PurgeDeleted(ctx context.Context, id shared.GlobalID) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !user.IsDeleted {
		return domain.ErrMailUserNotDeleted
	}

	// Deprovision storage (close SQLite pool, remove data directory)
	if s.storageRoot != "" && user.DataPath != "" && s.provisionSvc != nil {
		if err := s.provisionSvc.Deprovision(ctx, s.storageRoot, user.DataPath); err != nil {
			s._log.Warnf("purge account %s: deprovision failed: %v", id, err)
		}
	}

	return s.repo.HardDelete(ctx, id)
}

func (s *mailUserServiceImpl) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	return s.repo.ToggleActive(ctx, id)
}

func (s *mailUserServiceImpl) ChangePassword(ctx context.Context, id shared.GlobalID, oldPassword, newPassword string) error {
	if newPassword == oldPassword {
		return domain.ErrMailUserPassMismatch
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !user.VerifyPassword(oldPassword) {
		return domain.ErrMailUserInvalidPass
	}
	hash, err := shared.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("password hash failed: %w", err)
	}
	user.SetPasswordHash(hash)
	return s.repo.Save(ctx, user)
}

func (s *mailUserServiceImpl) ResetPassword(ctx context.Context, id shared.GlobalID, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	hash, err := shared.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("password hash failed: %w", err)
	}
	user.SetPasswordHash(hash)
	return s.repo.Save(ctx, user)
}

var _ MailUserService = (*mailUserServiceImpl)(nil)
