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

// Package i18n loads embedded go-i18n message catalogs (default English, optional Simplified Chinese).
//
// Use [Message] / [MessageWith] with a [*gin.Context] from HTTP handlers so [GinMiddleware] can resolve language
// from the query (?lang=), Accept-Language, or default English.
//
// Use [LogMessage] for process logs outside HTTP (always uses the default English localizer unless you pass a custom bundle).
//
// Message IDs are defined as constants in this package (Key*); translations live in embed/active.*.json.
package i18n
