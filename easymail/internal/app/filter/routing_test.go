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
	"testing"

	"easymail/pkg/constants"
	"easymail/pkg/config"
)

func TestFolderKindForInbound_LooseHeadersLF(t *testing.T) {
	o := &LMTPRouteOptions{Config: config.FilterConfig{}}
	// With LF-only body and no From line, net/mail.ReadMessage often fails; filter headers must still apply.
	raw := "Subject: x\nX-Easymail-Filter-Action: spam\nX-Easymail-Filter-Rule-Id: 1\n\nbody\n"
	k := FolderKindForInbound([]byte(raw), o)
	if k != constants.Spam {
		t.Fatalf("want Spam kind, got %v", k)
	}
}

func TestParseFilterHeadersFromBody_Loose(t *testing.T) {
	raw := "Received: a\nX-Easymail-Filter-Action: quarantine\n\nx"
	a, _, _ := ParseFilterHeadersFromBody([]byte(raw))
	if a != "quarantine" {
		t.Fatalf("action=%q", a)
	}
}

func TestFolderKindForInbound_RejectFallsBackToDefault(t *testing.T) {
	o := &LMTPRouteOptions{Config: config.FilterConfig{DefaultAction: "quarantine"}}
	raw := "Subject: x\nX-Easymail-Filter-Action: reject\n\nbody\n"
	k := FolderKindForInbound([]byte(raw), o)
	if k != constants.Quarantine {
		t.Fatalf("want default kind Quarantine, got %v", k)
	}
}
