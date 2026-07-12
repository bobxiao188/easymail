/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package webmail

import (
	"context"
	managementSvc "easymail/internal/app/management"
	"easymail/internal/domain/shared"
	"easymail/pkg/jwt"
)

type authServiceImpl struct {
	backend        managementSvc.MailUserAuthService
	jwtSecret      string
	jwtExpireHours int
}

func NewAuthService(backend managementSvc.MailUserAuthService, jwtSecret string, jwtExpireHours int) AuthService {
	return &authServiceImpl{
		backend:        backend,
		jwtSecret:      jwtSecret,
		jwtExpireHours: jwtExpireHours,
	}
}

func (s *authServiceImpl) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	acc, err := s.backend.Authenticate(ctx, email, password)
	if err != nil {
		return nil, err
	}
	tok, err := jwt.GenerateWebmailToken(string(acc.ID), email, s.jwtSecret, s.jwtExpireHours)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		Token: tok,
		Email: email,
	}, nil
}

func (s *authServiceImpl) GetProfile(_ context.Context, userID shared.GlobalID) (string, error) {
	return string(userID), nil
}

func (s *authServiceImpl) ChangePassword(ctx context.Context, email, oldPassword, newPassword string) error {
	return s.backend.ChangePassword(ctx, email, oldPassword, newPassword)
}
