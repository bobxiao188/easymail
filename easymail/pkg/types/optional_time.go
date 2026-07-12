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

package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// OptionalTime 鐢ㄤ簬鍖哄垎鈥滃瓧娈垫湭浼犫€濅笌鈥滃瓧娈垫樉寮忎紶 null锛堟竻绌猴級鈥?// - Present=false锛氳姹備腑娌℃湁璇ュ瓧娈碉紙涓嶆洿鏂帮級
// - Present=true && Value=nil锛氭樉寮忎紶 null锛堟竻绌猴級
// - Present=true && Value!=nil锛氳缃负鎸囧畾鏃堕棿
type OptionalTime struct {
	Present bool
	Value   *time.Time
}

func (o *OptionalTime) UnmarshalJSON(b []byte) error {
	o.Present = true
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) || len(b) == 0 {
		o.Value = nil
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		o.Value = nil
		return nil
	}

	// 鍓嶇閫氬父涓篟FC3339锛涜繖閲屽悓鏃跺吋瀹筊FC3339Nano
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
	}
	o.Value = &t
	return nil
}
