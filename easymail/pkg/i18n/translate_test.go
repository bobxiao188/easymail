/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package i18n

import (
	"strings"
	"testing"
)

func TestMessageForLanguage_loginKeys(t *testing.T) {
	t.Parallel()
	en := MessageForLanguage("en", KeyAuthInvalidCredentials)
	zh := MessageForLanguage("zh", KeyAuthInvalidCredentials)
	if en == zh {
		t.Fatalf("en and zh should differ: %q vs %q", en, zh)
	}
	if !strings.Contains(zh, "瀵嗙爜") && !strings.Contains(zh, "鐢ㄦ埛") {
		t.Fatalf("zh should be Chinese: %q", zh)
	}
}

func TestLanguageTagFromParam(t *testing.T) {
	t.Parallel()
	if LanguageTagFromParam("zh") != LanguageTagFromParam("ZH") {
		t.Fatal("case insensitive")
	}
	if got := LanguageTagFromParam("en"); got == "" {
		t.Fatal("en tag")
	}
}
