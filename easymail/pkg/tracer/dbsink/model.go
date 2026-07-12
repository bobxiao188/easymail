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

package dbsink

import (
	"time"
)

type SessionEvent struct {
	ID        uint      `gorm:"autoIncrement;primaryKey"`
	TS        time.Time `gorm:"index"`
	SessionID string    `gorm:"type:varchar(64);index"`
	Protocol  string    `gorm:"type:varchar(32);index"`
	Stage     string    `gorm:"type:varchar(64);index"`
	Remote    string    `gorm:"type:varchar(128)"`
	Local     string    `gorm:"type:varchar(128)"`
	Duration  int64     `gorm:"type:bigint"` // nanoseconds
	Err       string    `gorm:"type:text"`
	Fields    string    `gorm:"type:longtext"`
	Tags      string    `gorm:"type:longtext"`
}
