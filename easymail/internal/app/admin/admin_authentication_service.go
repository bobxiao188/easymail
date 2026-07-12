/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * For commercial licensing inquiries, please contact: 3680010825@qq.com
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package admin

import (
	"context"
	adminexception "easymail/internal/app/admin/exception"
	"easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/pkg/jwt"
)

type LoginRequest struct {
	Username string
	Password string
	Language string
}

type LoginResponse struct {
	Token string         `json:"token"`
	User  *AdminUserInfo `json:"user"`
}

type AdminUserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Language string `json:"language"`
	Skin     string `json:"skin"`
	IsAdmin  bool   `json:"is_admin"`
}

type AuthenticationService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	GetProfile(ctx context.Context, userID string) (*AdminUserInfo, error)
	UpdateProfile(ctx context.Context, userID string, nickname, email, avatar string) (*AdminUserInfo, error)
	ChangePassword(ctx context.Context, userID string, oldPassword, newPassword string) error
	UpdateLanguage(ctx context.Context, userID string, language string) error
	UpdateSkin(ctx context.Context, userID string, skin string) error
}

// AdminUserRepository defines the data-access port for admin user operations.
type AdminUserRepository interface {
	FindByID(ctx context.Context, id shared.GlobalID) (*management.AdminUser, error)
	FindByUsername(ctx context.Context, username string) (*management.AdminUser, error)
	Save(ctx context.Context, user *management.AdminUser) error
}

type authenticationServiceImpl struct {
	adminUserRepo  AdminUserRepository
	jwtSecret      string
	jwtExpireHours int
}

func NewAuthenticationService(adminUserRepo AdminUserRepository, jwtSecret string, jwtExpireHours int) AuthenticationService {
	return &authenticationServiceImpl{
		adminUserRepo:  adminUserRepo,
		jwtSecret:      jwtSecret,
		jwtExpireHours: jwtExpireHours,
	}
}

// Login validates credentials and issues a JWT. It returns sentinel errors (e.g. ErrInvalidCredentials);
// HTTP handlers should map them to localized messages using i18n.MessageForLanguage(req.Language, key).
func (s *authenticationServiceImpl) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.adminUserRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, adminexception.ErrInvalidCredentials
	}

	if !user.Validate() {
		return nil, adminexception.ErrUserInactive
	}

	if !user.VerifyPassword(req.Password) {
		return nil, adminexception.ErrInvalidCredentials
	}

	user.Language = req.Language

	token, err := jwt.GenerateToken(string(user.ID), user.Username, true, s.jwtSecret, s.jwtExpireHours)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User: &AdminUserInfo{
			ID:       string(user.ID),
			Username: user.Username,
			Nickname: user.Nickname,
			Email:    user.Email,
			Avatar:   user.Avatar,
			Language: user.Language,
			Skin:     user.Skin,
			IsAdmin:  true,
		},
	}, nil
}

func (s *authenticationServiceImpl) GetProfile(ctx context.Context, userID string) (*AdminUserInfo, error) {
	user, err := s.adminUserRepo.FindByID(ctx, shared.GlobalID(userID))
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, adminexception.ErrUserNotFound
	}

	return &AdminUserInfo{
		ID:       string(user.ID),
		Username: user.Username,
		Nickname: user.Nickname,
		Email:    user.Email,
		Avatar:   user.Avatar,
		Language: user.Language,
		Skin:     user.Skin,
		IsAdmin:  true,
	}, nil
}

func (s *authenticationServiceImpl) UpdateProfile(ctx context.Context, userID string, nickname, email, avatar string) (*AdminUserInfo, error) {
	user, err := s.adminUserRepo.FindByID(ctx, shared.GlobalID(userID))
	if err != nil {
		return nil, err
	}
	if user == nil || !user.Validate() {
		return nil, adminexception.ErrUserInactive
	}
	user.Nickname = nickname
	user.Email = email
	user.Avatar = avatar
	if err := s.adminUserRepo.Save(ctx, user); err != nil {
		return nil, err
	}
	return s.GetProfile(ctx, userID)
}

func (s *authenticationServiceImpl) ChangePassword(ctx context.Context, userID string, oldPassword, newPassword string) error {
	user, err := s.adminUserRepo.FindByID(ctx, shared.GlobalID(userID))
	if err != nil {
		return err
	}
	if user == nil || !user.Validate() {
		return adminexception.ErrUserInactive
	}
	if newPassword == "" || len(newPassword) < 8 {
		return adminexception.ErrPasswordTooShort
	}
	if newPassword == oldPassword {
		return adminexception.ErrNewPasswordMustDiffer
	}
	if !user.VerifyPassword(oldPassword) {
		return adminexception.ErrInvalidCredentials
	}
	hash, err := shared.Hash(newPassword)
	if err != nil {
		return err
	}
	user.SetPasswordHash(hash)
	return s.adminUserRepo.Save(ctx, user)
}

func (s *authenticationServiceImpl) UpdateLanguage(ctx context.Context, userID string, language string) error {
	user, err := s.adminUserRepo.FindByID(ctx, shared.GlobalID(userID))
	if err != nil {
		return err
	}
	if user == nil || !user.Validate() {
		return adminexception.ErrUserInactive
	}
	user.Language = language
	return s.adminUserRepo.Save(ctx, user)
}

func (s *authenticationServiceImpl) UpdateSkin(ctx context.Context, userID string, skin string) error {
	user, err := s.adminUserRepo.FindByID(ctx, shared.GlobalID(userID))
	if err != nil {
		return err
	}
	if user == nil || !user.Validate() {
		return adminexception.ErrUserInactive
	}
	user.Skin = skin
	return s.adminUserRepo.Save(ctx, user)
}
