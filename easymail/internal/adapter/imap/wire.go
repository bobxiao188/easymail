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

package imap

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"easymail/internal/domain/messaging"
)

// formatEnvelope builds the RFC 3501 ENVELOPE parenthesized list (including outer envelope parens).
func formatEnvelope(e *messaging.Email) string {
	date, _ := time.Parse(time.RFC3339, e.MailTime)
	if date.IsZero() {
		date = time.Now()
	}
	dateStr := date.Format("02-Jan-2006 15:04:05 -0700")
	subj := quoteIMAPString(e.Subject)
	from := envelopeAddrList(e.Sender)
	sender := from
	replyTo := "NIL"
	to := envelopeAddrList(e.Recipient)
	cc := "NIL"
	bcc := "NIL"
	inReplyTo := "NIL"
	msgid := "NIL"
	return fmt.Sprintf(`(%s %s %s %s %s %s %s %s %s %s)`,
		quoteIMAPString(dateStr), subj, from, sender, replyTo, to, cc, bcc, inReplyTo, msgid)
}

func envelopeAddrList(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "NIL"
	}
	one := singleAddress(raw)
	return "(" + one + ")"
}

func singleAddress(raw string) string {
	a := parseOneAddr(raw)
	name := "NIL"
	if a.name != "" {
		name = quoteIMAPString(a.name)
	}
	return fmt.Sprintf(`(%s NIL %s %s)`, name, quoteIMAPString(a.mailbox), quoteIMAPString(a.host))
}

func escapeQuoted(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString("\\\\")
		case r == '"':
			b.WriteString("\"")
		case r < 0x20: // control chars: escape as hex
			fmt.Fprintf(&b, `\%03o`, r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func quoteIMAPString(s string) string {
	if s == "" {
		return `""`
	}
	return `"` + escapeQuoted(s) + `"`
}

type addrParts struct {
	name, mailbox, host string
}

func parseOneAddr(s string) *addrParts {
	// Minimal parse: last "@" separates local part and domain.
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "@"); i > 0 && i < len(s)-1 {
		return &addrParts{mailbox: s[:i], host: s[i+1:]}
	}
	return &addrParts{mailbox: s, host: "local"}
}

func formatFlagsList(flags []string) string {
	if len(flags) == 0 {
		return "()"
	}
	var b strings.Builder
	b.WriteByte('(')
	for i, f := range flags {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(f)
	}
	b.WriteByte(')')
	return b.String()
}

func uint32dec(n uint32) string {
	return strconv.FormatUint(uint64(n), 10)
}

