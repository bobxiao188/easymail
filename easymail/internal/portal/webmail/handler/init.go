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

package handler

import (
	webmailApp "easymail/internal/app/webmail"
	"easymail/pkg/logger/easylog"
)

// Handler exposes webmail HTTP endpoints (auth, mail, contacts).
type Handler struct {
	authService    webmailApp.AuthService
	mailService    webmailApp.MailService
	contactService webmailApp.ContactService
	profileService webmailApp.ProfileService
	log            *easylog.Logger
}

// New wires the webmail handler with auth, mail, and contact services.
func New(
	authService webmailApp.AuthService,
	mailService webmailApp.MailService,
	contactService webmailApp.ContactService,
	profileService webmailApp.ProfileService,
	logger *easylog.Logger,
) *Handler {
	return &Handler{
		authService:    authService,
		mailService:    mailService,
		contactService: contactService,
		profileService: profileService,
		log:            logger,
	}
}
