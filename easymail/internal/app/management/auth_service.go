package management

import (
	"context"
	"fmt"

	dom "easymail/internal/domain/management"
)

type MailUserAuthService interface {
	Authenticate(ctx context.Context, email, password string) (*dom.MailUser, error)
	ChangePassword(ctx context.Context, email, oldPassword, newPassword string) error
}

type mailUserAuthServiceImpl struct {
	userRepo   dom.MailUserRepository
	domainRepo dom.MailDomainRepository
}

func NewMailUserAuthService(userRepo dom.MailUserRepository, domainRepo dom.MailDomainRepository) MailUserAuthService {
	return &mailUserAuthServiceImpl{userRepo: userRepo, domainRepo: domainRepo}
}

func (s *mailUserAuthServiceImpl) Authenticate(ctx context.Context, email, password string) (*dom.MailUser, error) {
	user, err := s.userRepo.FindByFullEmail(ctx, email)
	if err != nil { return nil, fmt.Errorf("authentication failed: %w", dom.ErrMailUserNotFound) }
	d, err := s.domainRepo.FindByID(ctx, user.DomainID)
	if err != nil { return nil, fmt.Errorf("authentication failed: %w", dom.ErrMailUserDomainInvalid) }
	if !d.Validate() { return nil, fmt.Errorf("authentication failed: %w", dom.ErrMailUserDomainInvalid) }
	if !user.Validate() { return nil, fmt.Errorf("authentication failed: %w", dom.ErrMailUserInactive) }
	if !user.VerifyPassword(password) { return nil, fmt.Errorf("authentication failed: %w", dom.ErrMailUserInvalidPass) }
	if user.IsPasswordExpired() { return nil, fmt.Errorf("authentication failed: %w", dom.ErrMailUserInvalidPass) }
	return user, nil
}

func (s *mailUserAuthServiceImpl) ChangePassword(ctx context.Context, email, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByFullEmail(ctx, email)
	if err != nil { return fmt.Errorf("change password failed: %w", dom.ErrMailUserNotFound) }
	d, err := s.domainRepo.FindByID(ctx, user.DomainID)
	if err != nil { return fmt.Errorf("change password failed: %w", dom.ErrMailUserDomainInvalid) }
	if !d.Validate() { return fmt.Errorf("change password failed: %w", dom.ErrMailUserDomainInvalid) }
	if !user.Validate() { return fmt.Errorf("change password failed: %w", dom.ErrMailUserInactive) }
	if !user.VerifyPassword(oldPassword) { return fmt.Errorf("change password failed: %w", dom.ErrMailUserInvalidPass) }
	return s.userRepo.ChangePassword(ctx, user.ID, oldPassword, newPassword)
}

type AdminUserAuthService interface {
	Authenticate(ctx context.Context, username, password string) (*dom.AdminUser, error)
}

type adminUserAuthServiceImpl struct{ repo dom.AdminUserRepository }

func NewAdminUserAuthService(repo dom.AdminUserRepository) AdminUserAuthService {
	return &adminUserAuthServiceImpl{repo: repo}
}

func (s *adminUserAuthServiceImpl) Authenticate(ctx context.Context, username, password string) (*dom.AdminUser, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil { return nil, fmt.Errorf("admin authentication failed: %w", dom.ErrAdminUserNotFound) }
	if !user.Validate() { return nil, fmt.Errorf("admin authentication failed: %w", dom.ErrAdminUserInactive) }
	if !user.VerifyPassword(password) { return nil, fmt.Errorf("admin authentication failed: %w", dom.ErrAdminUserInvalidPass) }
	return user, nil
}

var _ MailUserAuthService = (*mailUserAuthServiceImpl)(nil)
var _ AdminUserAuthService = (*adminUserAuthServiceImpl)(nil)
