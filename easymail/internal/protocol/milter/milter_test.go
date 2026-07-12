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
	"testing"
)

func TestDecodeCStrings(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want []string
	}{
		{"empty", nil, nil},
		{"empty slice", []byte{}, nil},
		{"single", []byte("hello\x00"), []string{"hello"}},
		{"multiple", []byte("a\x00b\x00c\x00"), []string{"a", "b", "c"}},
		{"no trailing null", []byte("hello"), []string{"hello"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeCStrings(tt.data)
			if len(got) != len(tt.want) {
				t.Fatalf("decodeCStrings() = %q, want %q", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("decodeCStrings()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestReadCString(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"with null", []byte("hello\x00world"), "hello"},
		{"without null", []byte("hello"), "hello"},
		{"empty", []byte{}, ""},
		{"only null", []byte("\x00"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readCString(tt.data)
			if got != tt.want {
				t.Errorf("readCString(%q) = %q, want %q", string(tt.data), got, tt.want)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	m := &Message{Code: 'a', Data: []byte("test")}
	if m.Code != 'a' {
		t.Errorf("Code = %c", m.Code)
	}
	if string(m.Data) != "test" {
		t.Errorf("Data = %q", string(m.Data))
	}
}

func TestResponseConstants(t *testing.T) {
	if byte(RespAccept) != 'a' {
		t.Errorf("RespAccept = %c", RespAccept)
	}
	if byte(RespContinue) != 'c' {
		t.Errorf("RespContinue = %c", RespContinue)
	}
	if byte(RespDiscard) != 'd' {
		t.Errorf("RespDiscard = %c", RespDiscard)
	}
	if byte(RespReject) != 'r' {
		t.Errorf("RespReject = %c", RespReject)
	}
	if byte(RespTempFail) != 't' {
		t.Errorf("RespTempFail = %c", RespTempFail)
	}
}

func TestSimpleResponse_Response(t *testing.T) {
	r := RespAccept
	msg := r.Response()
	if msg.Code != 'a' {
		t.Errorf("Response().Code = %c", msg.Code)
	}
}

func TestSimpleResponse_Continue(t *testing.T) {
	tests := []struct {
		r    SimpleResponse
		want bool
	}{
		{RespAccept, false},
		{RespContinue, true},
		{RespDiscard, false},
		{RespReject, false},
		{RespTempFail, false},
	}

	for _, tt := range tests {
		t.Run(string(byte(tt.r)), func(t *testing.T) {
			got := tt.r.Continue()
			if got != tt.want {
				t.Errorf("Continue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomResponse_Response(t *testing.T) {
	r := NewResponse('y', []byte("data"))
	msg := r.Response()
	if msg.Code != 'y' || string(msg.Data) != "data" {
		t.Errorf("Response() = {%c,%q}", msg.Code, string(msg.Data))
	}
}

func TestCustomResponse_Continue(t *testing.T) {
	tests := []struct {
		r    *CustomResponse
		want bool
	}{
		{NewResponse('a', nil), false},
		{NewResponse('c', nil), true},
		{NewResponse('d', nil), false},
		{NewResponse('r', nil), false},
		{NewResponse('t', nil), false},
		{NewResponse('x', nil), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.r.code), func(t *testing.T) {
			got := tt.r.Continue()
			if got != tt.want {
				t.Errorf("Continue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewResponseStr(t *testing.T) {
	r := NewResponseStr('y', "data")
	msg := r.Response()
	if msg.Code != 'y' || string(msg.Data) != "data\x00" {
		t.Errorf("NewResponseStr() = {%c,%q}", msg.Code, string(msg.Data))
	}
}

func TestErrors(t *testing.T) {
	if errCloseSession.Error() != "stop current milter processing" {
		t.Errorf("errCloseSession = %q", errCloseSession.Error())
	}
}

func TestResponseInterface(t *testing.T) {
	var _ Response = RespAccept
	var _ Response = (*CustomResponse)(nil)
}


