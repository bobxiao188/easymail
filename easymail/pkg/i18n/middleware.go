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

var supported = []language.Tag{
	language.English,
	language.SimplifiedChinese,
}

var langMatcher = language.NewMatcher(supported)

// GinMiddleware selects the response language: query ?lang=zh|en, then Accept-Language, then English.
func GinMiddleware() gin.HandlerFunc {
	_ = Bundle()
	return func(c *gin.Context) {
		tag := resolveTag(c)
		loc := i18n.NewLocalizer(Bundle(), tag.String())
		c.Set(GinContextKeyLocalizer, loc)
		c.Set(GinContextKeyLanguage, tag.String())
		c.Next()
	}
}

func resolveTag(c *gin.Context) language.Tag {
	if q := strings.TrimSpace(strings.ToLower(c.Query("lang"))); q != "" {
		switch q {
		case "zh", "zh-cn", "zh-hans", "cn":
			return language.SimplifiedChinese
		default:
			return language.English
		}
	}
	al := c.GetHeader("Accept-Language")
	if al != "" {
		tags, _, err := language.ParseAcceptLanguage(al)
		if err == nil && len(tags) > 0 {
			t, _, _ := langMatcher.Match(tags...)
			return t
		}
	}
	return language.English
}
