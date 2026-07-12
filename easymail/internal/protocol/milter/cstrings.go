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

package milter

import (
	"bytes"
	"strings"
)

// NULL terminator
const null = "\x00"

// DecodeCStrings splits a C style strings into a Go slice
func decodeCStrings(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.Split(strings.Trim(string(data), null), null)
}

// ReadCString reads and returns a C style string from []byte
func readCString(data []byte) string {
	pos := bytes.IndexByte(data, 0)
	if pos == -1 {
		return string(data)
	}
	return string(data[0:pos])
}

