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

package handler

import (
	"errors"

	"easymail/internal/app/webmail"
	"easymail/internal/domain/management"
	mailexception "easymail/internal/domain/messaging/exception"
	appi18n "easymail/pkg/i18n"

	"github.com/gin-gonic/gin"
)

// messageContactGroupOp returns an i18n message for contact group operation errors.
func messageContactGroupOp(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, webmail.ErrNotFound):
		return appi18n.Message(c, appi18n.KeyWebmailContactsNotFound)
	case errors.Is(err, webmail.ErrInvalidGroup):
		return appi18n.Message(c, appi18n.KeyWebmailContactsInvalidGroup)
	case errors.Is(err, webmail.ErrInvalidArgument):
		return appi18n.Message(c, appi18n.KeyWebmailContactsInvalidArgument)
	default:
		return appi18n.Message(c, appi18n.KeyErrOperationFailed)
	}
}

// messageChangePassword returns an i18n message for password change errors.
func messageChangePassword(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, management.ErrMailUserNotFound),
		errors.Is(err, management.ErrMailUserDomainInvalid):
		return appi18n.Message(c, appi18n.KeyWebmailAuthInvalidCredentials)
	case errors.Is(err, management.ErrMailUserInvalidPass):
		return appi18n.Message(c, appi18n.KeyWebmailAuthOldPasswordIncorrect)
	default:
		return appi18n.Message(c, appi18n.KeyErrInternalServer)
	}
}

// messageMailOp returns an i18n message for mail operation errors.
func messageMailOp(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, mailexception.ErrNotFound):
		return appi18n.Message(c, appi18n.KeyMailNotFound)
	case errors.Is(err, mailexception.ErrForbidden):
		return appi18n.Message(c, appi18n.KeyMailForbidden)
	case errors.Is(err, mailexception.ErrInvalidArgument):
		return appi18n.Message(c, appi18n.KeyMailInvalidArgument)
	case errors.Is(err, mailexception.ErrPurgeNotAllowed):
		return appi18n.Message(c, appi18n.KeyMailPurgeOrder)
	case errors.Is(err, mailexception.ErrFolderSystem):
		return appi18n.Message(c, appi18n.KeyMailFolderSystem)
	case errors.Is(err, mailexception.ErrFolderNotEmpty):
		return appi18n.Message(c, appi18n.KeyMailFolderNotEmpty)
	case errors.Is(err, mailexception.ErrComposeNoRecipient):
		return appi18n.Message(c, appi18n.KeyWebmailRecipientRequired)
	case errors.Is(err, mailexception.ErrComposeExternalRecipient):
		return appi18n.Message(c, appi18n.KeyWebmailLocalOnly)
	case errors.Is(err, mailexception.ErrComposeAddress):
		return appi18n.Message(c, appi18n.KeyWebmailInvalidEmail)
	case errors.Is(err, mailexception.ErrComposeEmptyBody):
		return appi18n.Message(c, appi18n.KeyWebmailBodyEmpty)
	default:
		return appi18n.Message(c, appi18n.KeyErrInternalServer)
	}
}
