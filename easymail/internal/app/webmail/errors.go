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

import "errors"

var (
	ErrNotFound        = errors.New("contact not found")
	ErrDuplicate       = errors.New("contact already exists")
	ErrInvalidEmail    = errors.New("invalid email address")
	ErrInvalidGroup    = errors.New("invalid group")
	ErrInvalidArgument = errors.New("invalid argument")
)

