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

package tokenizer

import (
	"strings"

	"easymail/internal/domain/filter/feature"
)

// SpaceTokenizer splits on Unicode whitespace via strings.Fields.
type SpaceTokenizer struct{}

// NewSpaceTokenizer returns a tokenizer that only splits on whitespace.
func NewSpaceTokenizer() feature.TextTokenizer {
	return SpaceTokenizer{}
}

// NewDefaultTokenizer is an alias for NewSpaceTokenizer.
func NewDefaultTokenizer() feature.TextTokenizer {
	return NewSpaceTokenizer()
}

// Tokenize implements feature.TextTokenizer.
func (SpaceTokenizer) Tokenize(text string) []string {
	return strings.Fields(strings.TrimSpace(text))
}
