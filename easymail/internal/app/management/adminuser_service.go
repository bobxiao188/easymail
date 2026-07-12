// internal/app/management/adminuser_service.go
package management

import (
	"context"
	"fmt"
	"strings"

	domain "easymail/internal/domain/management"
	"easymail/internal/domain/shared"
)

type AdminUserService interface {
	Create(ctx context.Context, username, password, nickname, email string) (*domain.AdminUser, error)
	GetByID(ctx context.Context, id shared.GlobalID) (*domain.AdminUser, error)
	GetByUsername(ctx context.Context, username string) (*domain.AdminUser, error)
	List(ctx context.Context, keyword string, page, pageSize int) ([]domain.AdminUser, int64, error)
	UpdateProfile(ctx context.Context, id shared.GlobalID, username, nickname, email string, active bool) error
	Delete(ctx context.Context, id shared.GlobalID) error
	ToggleActive(ctx context.Context, id shared.GlobalID) error
	ChangePassword(ctx context.Context, id shared.GlobalID, oldPassword, newPassword string) error
	ResetPassword(ctx context.Context, id shared.GlobalID, newPassword string) error
}

type adminUserServiceImpl struct{ repo domain.AdminUserRepository }

func NewAdminUserService(repo domain.AdminUserRepository) AdminUserService {
	return &adminUserServiceImpl{repo: repo}
}

func (s *adminUserServiceImpl) Create(ctx context.Context, username, password, nickname, email string) (*domain.AdminUser, error) {
	if strings.TrimSpace(password) == "" {
		return nil, domain.ErrAdminUserInvalidPass
	}
	hash, err := shared.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("password hash failed: %w", err)
	}
	user, err := domain.NewAdminUser(username, hash, nickname, email)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *adminUserServiceImpl) GetByID(ctx context.Context, id shared.GlobalID) (*domain.AdminUser, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *adminUserServiceImpl) GetByUsername(ctx context.Context, username string) (*domain.AdminUser, error) {
	return s.repo.FindByUsername(ctx, username)
}

func (s *adminUserServiceImpl) List(ctx context.Context, keyword string, page, pageSize int) ([]domain.AdminUser, int64, error) {
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 10 }
	return s.repo.Search(ctx, keyword, page, pageSize)
}

func (s *adminUserServiceImpl) UpdateProfile(ctx context.Context, id shared.GlobalID, username, nickname, email string, active bool) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil { return err }
	user.Username = strings.ToLower(username)
	user.Nickname = nickname
	user.Email = email
	if active { user.Activate() } else { user.Deactivate() }
	return s.repo.Save(ctx, user)
}

func (s *adminUserServiceImpl) Delete(ctx context.Context, id shared.GlobalID) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil { return err }
	user.SoftDelete()
	return s.repo.Save(ctx, user)
}

func (s *adminUserServiceImpl) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	return s.repo.ToggleActive(ctx, id)
}

func (s *adminUserServiceImpl) ChangePassword(ctx context.Context, id shared.GlobalID, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if newPassword == oldPassword { return domain.ErrAdminUserPassMismatch }
	user, err := s.repo.FindByID(ctx, id)
	if err != nil { return err }
	if !user.VerifyPassword(oldPassword) { return domain.ErrAdminUserInvalidPass }
	hash, err := shared.Hash(newPassword)
	if err != nil { return fmt.Errorf("password hash failed: %w", err) }
	user.SetPasswordHash(hash)
	return s.repo.Save(ctx, user)
}

func (s *adminUserServiceImpl) ResetPassword(ctx context.Context, id shared.GlobalID, newPassword string) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil { return err }
	hash, err := shared.Hash(newPassword)
	if err != nil { return fmt.Errorf("password hash failed: %w", err) }
	user.SetPasswordHash(hash)
	return s.repo.Save(ctx, user)
}

var _ AdminUserService = (*adminUserServiceImpl)(nil)
