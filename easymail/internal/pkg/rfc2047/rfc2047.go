/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * License: AGPLv3
 */

package rfc2047

import (
	"mime"
	"strings"
)

// DecodeHeader decodes RFC 2047 encoded-words in a header value (e.g. Subject).
func DecodeHeader(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	dec := new(mime.WordDecoder)
	out, err := dec.DecodeHeader(raw)
	if err != nil {
		return raw
	}
	return out
}
