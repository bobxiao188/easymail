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
package rule

import (
	"context"
	"easymail/internal/domain/filter"
)

// ContentPlugin is a pluggable analyzer for DATA/Body stage.
// Implementations live in package extractors; registered via RegisterContentPlugin.
type ContentPlugin interface {
	Key() string
	Stage() filter.Stage
	Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error)
}
