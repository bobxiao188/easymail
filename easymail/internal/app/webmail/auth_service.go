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
	"easymail/internal/domain/shared"
)

// LoginResponse is returned after successful authentication.
type LoginResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// AuthService handles webmail user authentication.
type AuthService interface {
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	GetProfile(ctx context.Context, userID shared.GlobalID) (string, error)
	ChangePassword(ctx context.Context, email, oldPassword, newPassword string) error
}
