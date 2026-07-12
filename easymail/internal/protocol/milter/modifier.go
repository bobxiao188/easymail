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

// Modifier instance is provided to milter handlers to modify email messages

package milter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/textproto"
	"strings"
)

// Modifier provides access to Macros, Headers and Body data to callback handlers. It also defines a
// number of functions that can be used by callback handlers to modify processing of the email message
type Modifier struct {
	Macros  map[string]string
	Headers textproto.MIMEHeader

	writePacket func(*Message) error
}

// AddRecipient appends a new envelope recipient for current message
func (m *Modifier) AddRecipient(r string) error {
	data := []byte(fmt.Sprintf("<%s>", r) + null)
	return m.writePacket(NewResponse('+', data).Response())
}

// DeleteRecipient removes an envelope recipient address from message
func (m *Modifier) DeleteRecipient(r string) error {
	data := []byte(fmt.Sprintf("<%s>", r) + null)
	return m.writePacket(NewResponse('-', data).Response())
}

// ReplaceBody substitutes message body with provided body
func (m *Modifier) ReplaceBody(body []byte) error {
	return m.writePacket(NewResponse('b', body).Response())
}

// AddHeader appends a new email message header the message
func (m *Modifier) AddHeader(name, value string) error {
	data := []byte(name + null + value + null)
	return m.writePacket(NewResponse('h', data).Response())
}

// Quarantine a message by giving a reason to hold it
func (m *Modifier) Quarantine(reason string) error {
	return m.writePacket(NewResponse('q', []byte(reason+null)).Response())
}

// ChangeHeader replaces the header at the specified position with a new one
func (m *Modifier) ChangeHeader(index int, name, value string) error {
	buffer := new(bytes.Buffer)
	// encode header index in the beginning
	if err := binary.Write(buffer, binary.BigEndian, uint32(index)); err != nil {
		return err
	}
	// add header name and value to buffer
	data := []byte(name + null + value + null)
	if _, err := buffer.Write(data); err != nil {
		return err
	}
	// prepare and send response packet
	return m.writePacket(NewResponse('m', buffer.Bytes()).Response())
}

// InsertHeader inserts the header at the specified position
func (m *Modifier) InsertHeader(index int, name, value string) error {
	buffer := new(bytes.Buffer)
	// encode header index in the beginning
	if err := binary.Write(buffer, binary.BigEndian, uint32(index)); err != nil {
		return err
	}
	// add header name and value to buffer
	data := []byte(name + null + value + null)
	if _, err := buffer.Write(data); err != nil {
		return err
	}
	// prepare and send response packet
	return m.writePacket(NewResponse('i', buffer.Bytes()).Response())
}

// ChangeFrom replaces the FROM envelope header with a new one
func (m *Modifier) ChangeFrom(value string) error {
	buffer := new(bytes.Buffer)
	// add header name and value to buffer
	data := []byte(value + null)
	if _, err := buffer.Write(data); err != nil {
		return err
	}
	// prepare and send response packet
	return m.writePacket(NewResponse('e', buffer.Bytes()).Response())
}

// RemoveHeader deletes every occurrence of fieldName from the queued message by issuing SMFIR_CHGHEADER
// with an empty value (Postfix removes the header). Requires negotiation OptChangeHeader.
// Indices are 1-based per header name, matching libmilter smfi_chgheader.
func (m *Modifier) RemoveHeader(fieldName string) error {
	if m == nil || m.Headers == nil || strings.TrimSpace(fieldName) == "" {
		return nil
	}
	canonical := textproto.CanonicalMIMEHeaderKey(fieldName)
	vals := m.Headers[canonical]
	if len(vals) == 0 {
		return nil
	}
	for i := len(vals); i >= 1; i-- {
		if err := m.ChangeHeader(i, fieldName, ""); err != nil {
			return err
		}
	}
	m.Headers.Del(canonical)
	return nil
}

// CustomReply sends a custom reply code to the client
// https://www.ietf.org/archive/id/draft-ietf-milter-protocol-18.html#name-response-codes
// SMFIR_REPLYCODE = 'y', indicates a custom reply code.
// Format: code + " " + xcode + " " + message + null
func (m *Modifier) CustomReply(code, xcode, message string) error {
	data := []byte(fmt.Sprintf("%s %s %s", code, xcode, message) + null)
	return m.writePacket(NewResponse('y', data).Response())
}

// newModifier creates a new Modifier instance from milterSession
func newModifier(s *milterSession) *Modifier {
	return &Modifier{
		Macros:      s.macros,
		Headers:     s.headers,
		writePacket: s.WritePacket,
	}
}

