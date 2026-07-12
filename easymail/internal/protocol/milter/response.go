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

// Response represents a response structure returned by callback
// handlers to indicate how the milter server should proceed
type Response interface {
	Response() *Message
	Continue() bool
}

// SimpleResponse type to define list of pre-defined responses
type SimpleResponse byte

// Response returns a Message object reference
func (r SimpleResponse) Response() *Message {
	return &Message{byte(r), nil}
}

// Continue to process milter messages only if current code is Continue
func (r SimpleResponse) Continue() bool {
	return byte(r) == rspContinue
}

// Define standard responses with no data
const (
	RespAccept   = SimpleResponse(rspAccept)
	RespContinue = SimpleResponse(rspContinue)
	RespDiscard  = SimpleResponse(rspDiscard)
	RespReject   = SimpleResponse(rspReject)
	RespTempFail = SimpleResponse(rspTempFail)
)

// CustomResponse is a response instance used by callback handlers to indicate
// how the milter should continue processing of current message
type CustomResponse struct {
	code byte
	data []byte
}

// Response returns message instance with data
func (c *CustomResponse) Response() *Message {
	return &Message{c.code, c.data}
}

// Continue returns false if milter chain should be stopped, true otherwise
func (c *CustomResponse) Continue() bool {
	for _, q := range []byte{rspAccept, rspDiscard, rspReject, rspTempFail} {
		if c.code == q {
			return false
		}
	}
	return true
}

// NewResponse generates a new CustomResponse suitable for WritePacket
func NewResponse(code byte, data []byte) *CustomResponse {
	return &CustomResponse{code, data}
}

// NewResponseStr generates a new CustomResponse with string payload
func NewResponseStr(code byte, data string) *CustomResponse {
	return NewResponse(code, []byte(data+null))
}

