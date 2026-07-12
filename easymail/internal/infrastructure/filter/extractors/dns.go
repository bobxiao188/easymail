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

package extractors

import (
	"strings"
)

// normalizeDNSName trims and lowercases a DNS name.
func normalizeDNSName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")
	return strings.ToLower(s)
}

func boolToFloat64(b bool) float64 {
	if b { return 1 }
	return 0
}

// extractDomainFromAddr extracts the domain part from an email address like "user@example.com".
// Returns empty string if no domain found.
func extractDomainFromAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	// Handle angle brackets <user@domain>
	if i := strings.Index(addr, "<"); i >= 0 {
		if j := strings.Index(addr[i:], ">"); j >= 0 {
			addr = addr[i+1 : i+j]
		}
	}
	at := strings.LastIndex(addr, "@")
	if at < 0 || at == len(addr)-1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(addr[at+1:]))
}

// extractDomainFromHeaderAddr extracts the domain from a From: header value.
// Handles display names like "John <user@example.com>".
func extractDomainFromHeaderAddr(addr string) string {
	return extractDomainFromAddr(addr)
}