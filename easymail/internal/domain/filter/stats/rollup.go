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
package stats

import "time"

// FilterMailStatsDaily is a per-calendar-day aggregate of filter_logs by normalized policy outcome.
type FilterMailStatsDaily struct {
	ID            int64     `json:"id"`
	StatDate      time.Time `json:"statDate"`
	ActionApplied string    `json:"actionApplied"`
	MailCount     int64     `json:"mailCount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// FilterStatsRollupWatermark tracks the last fully rolled calendar day for idempotent daily jobs.
type FilterStatsRollupWatermark struct {
	ID                    int64      `json:"id"`
	JobName               string     `json:"jobName"`
	LastCompletedStatDate *time.Time `json:"lastCompletedStatDate,omitempty"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}
