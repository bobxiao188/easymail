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

package i18n

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const (
	// GinContextKeyLocalizer stores *i18n.Localizer for the active request.
	GinContextKeyLocalizer = "easymail_i18n_localizer"
	// GinContextKeyLanguage stores the resolved BCP 47 language tag string (e.g. en, zh).
	GinContextKeyLanguage = "easymail_i18n_language"
)

// LocalizerFromGin returns the request localizer, or nil if middleware did not run.
func LocalizerFromGin(c *gin.Context) *i18n.Localizer {
	if c == nil {
		return nil
	}
	v, ok := c.Get(GinContextKeyLocalizer)
	if !ok {
		return nil
	}
	loc, _ := v.(*i18n.Localizer)
	return loc
}

// Message resolves messageID using the Gin context localizer, or falls back to English.
func Message(c *gin.Context, messageID string) string {
	return MessageWith(c, messageID, nil)
}

// MessageWith resolves messageID with template data (for templates in JSON translations).
func MessageWith(c *gin.Context, messageID string, templateData map[string]interface{}) string {
	loc := LocalizerFromGin(c)
	if loc == nil {
		loc = DefaultLocalizer()
	}
	s, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil || s == "" {
		s, _ = DefaultLocalizer().Localize(&i18n.LocalizeConfig{
			MessageID:    messageID,
			TemplateData: templateData,
		})
	}
	if s == "" {
		return messageID
	}
	return s
}

// LogMessage returns a localized string for non-HTTP code paths (defaults to English).
func LogMessage(messageID string, templateData map[string]interface{}) string {
	s, err := DefaultLocalizer().Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil || s == "" {
		return messageID
	}
	return s
}

// FolderDisplayName returns the localized display name for a system folder kind.
// It uses the Gin context to determine the language, falling back to English.
func FolderDisplayName(c *gin.Context, kind int64) string {
	var messageID string
	switch kind {
	case 1: // Inbox
		messageID = KeyFolderNameInbox
	case 2: // Sent
		messageID = KeyFolderNameSent
	case 3: // Draft
		messageID = KeyFolderNameDraft
	case 4: // Trash
		messageID = KeyFolderNameTrash
	case 5: // Spam
		messageID = KeyFolderNameSpam
	case 6: // Quarantine
		messageID = KeyFolderNameQuarantine
	default:
		return "" // Not a system folder
	}
	return Message(c, messageID)
}

// LanguageTagFromParam maps API language values (e.g. login body "en"|"zh") to a BCP 47 tag for Bundle localizers.
func LanguageTagFromParam(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh", "zh-cn", "zh-hans", "cn":
		return language.SimplifiedChinese.String()
	default:
		return language.English.String()
	}
}

// MessageForLanguage resolves messageID in a fixed language (e.g. LoginRequest.Language), without relying on Gin middleware.
func MessageForLanguage(lang, messageID string) string {
	return MessageForLanguageWith(lang, messageID, nil)
}

// MessageForLanguageWith is like MessageForLanguage with template data for interpolated translations.
func MessageForLanguageWith(lang string, messageID string, templateData map[string]interface{}) string {
	_ = Bundle()
	loc := i18n.NewLocalizer(Bundle(), LanguageTagFromParam(lang))
	s, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil || s == "" {
		s, _ = DefaultLocalizer().Localize(&i18n.LocalizeConfig{
			MessageID:    messageID,
			TemplateData: templateData,
		})
	}
	if s == "" {
		return messageID
	}
	return s
}
