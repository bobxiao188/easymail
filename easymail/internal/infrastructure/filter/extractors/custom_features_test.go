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
	"net/textproto"
	"regexp"
	"testing"

	"easymail/internal/domain/filter"
	"easymail/internal/pkg/rfc2047"
)

func TestDecodeRFC2047Header_UTF8Subject(t *testing.T) {
	// "=?UTF-8?B?5rWL6K+V?=" is base64 for "测试" in UTF-8
	raw := "=?UTF-8?B?5rWL6K+V?="
	got := rfc2047.DecodeHeader(raw)
	if got != "测试" {
		t.Fatalf("DecodeRFC2047Header() = %q, want 测试", got)
	}
}

func TestMetaRegexSubjectMatchesDecodedChinese(t *testing.T) {
	re := regexp.MustCompile(`测试`)
	fc := &filter.MilterContext{
		Headers: textproto.MIMEHeader{
			"Subject": []string{"=?UTF-8?B?5rWL6K+V?="},
		},
	}
	texts := sourceTexts(fc, "subject")
	if len(texts) != 1 || texts[0] != "测试" {
		t.Fatalf("sourceTexts subject = %#v", texts)
	}
	if !re.MatchString(texts[0]) {
		t.Fatal("regex should match decoded subject")
	}
}

func TestNormalizeMetaSource_subjectAlias(t *testing.T) {
	if normalizeMetaSource("Subject") != "subject" {
		t.Fatalf("subject alias: got %q", normalizeMetaSource("Subject"))
	}
}
