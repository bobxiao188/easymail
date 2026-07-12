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

package filter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SnapshotJSON renders features as a stable JSON object string (keys sorted).
func SnapshotJSON(f map[string]float64) string {
	if len(f) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]float64, len(f))
	for _, k := range keys {
		out[k] = f[k]
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// SnapshotString renders features as a compact string (e.g. for rule audit logging).
func SnapshotString(f map[string]float64) string {
	if len(f) == 0 {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{")
	first := true
	for k, v := range f {
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString(fmt.Sprintf("%q:%g", k, v))
	}
	b.WriteString("}")
	return b.String()
}