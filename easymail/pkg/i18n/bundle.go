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
	"embed"
	"encoding/json"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed embed/*.json
var messageFS embed.FS

var (
	bundleInit sync.Once
	bundle     *i18n.Bundle
	// defaultLocalizer is English; used for logs and code paths without a request context.
	defaultLocalizer *i18n.Localizer
)

// Bundle returns the shared go-i18n bundle (English as default language tag).
func Bundle() *i18n.Bundle {
	bundleInit.Do(func() {
		b := i18n.NewBundle(language.English)
		b.RegisterUnmarshalFunc("json", json.Unmarshal)
		if _, err := b.LoadMessageFileFS(messageFS, "embed/active.en.json"); err != nil {
			panic("i18n: load embed/active.en.json: " + err.Error())
		}
		if _, err := b.LoadMessageFileFS(messageFS, "embed/active.zh.json"); err != nil {
			panic("i18n: load embed/active.zh.json: " + err.Error())
		}
		bundle = b
		defaultLocalizer = i18n.NewLocalizer(bundle, language.English.String())
	})
	return bundle
}

// DefaultLocalizer returns an English localizer for background tasks and log messages
// when no HTTP request context is available.
func DefaultLocalizer() *i18n.Localizer {
	_ = Bundle()
	return defaultLocalizer
}
