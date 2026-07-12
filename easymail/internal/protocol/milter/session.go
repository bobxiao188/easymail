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
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"net/textproto"
	"strings"
	"time"

	"easymail/pkg/logger/easylog"
)

// OptAction sets which actions the milter wants to perform.
// Multiple options can be set using a bitmask.
type OptAction uint32

// OptProtocol masks out unwanted parts of the SMTP transaction.
// Multiple options can be set using a bitmask.
type OptProtocol uint32

const (
	// set which actions the milter wants to perform
	OptAddHeader    OptAction = 0x01
	OptChangeBody   OptAction = 0x02
	OptAddRcpt      OptAction = 0x04
	OptRemoveRcpt   OptAction = 0x08
	OptChangeHeader OptAction = 0x10
	OptQuarantine   OptAction = 0x20
	OptChangeFrom   OptAction = 0x40
	// OptReplyCode is SMFIF_REPLYCODE (libmilter): allows SMFIR_REPLYCODE ('y') / CustomReply.
	// Not 0x08 - that bit is SMFIF_DELRCPT (OptRemoveRcpt).
	OptReplyCode OptAction = 0x80

	// mask out unwanted parts of the SMTP transaction
	OptNoConnect  OptProtocol = 0x01
	OptNoHelo     OptProtocol = 0x02
	OptNoMailFrom OptProtocol = 0x04
	OptNoRcptTo   OptProtocol = 0x08
	OptNoBody     OptProtocol = 0x10
	OptNoHeaders  OptProtocol = 0x20
	OptNoEOH      OptProtocol = 0x40
)

// milterSession keeps session state during MTA communication
type milterSession struct {
	actions  OptAction
	protocol OptProtocol
	sock     io.ReadWriteCloser
	bw       *bufio.Writer
	headers  textproto.MIMEHeader
	macros   map[string]string
	milter   Milter
	log      *easylog.Logger
}

// ReadPacket reads incoming milter packet
func (s *milterSession) ReadPacket() (*Message, error) {
	// read packet length
	var length uint32
	if err := binary.Read(s.sock, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// read packet data
	data := make([]byte, length)
	if _, err := io.ReadFull(s.sock, data); err != nil {
		return nil, err
	}

	// prepare response data
	message := Message{
		Code: data[0],
		Data: data[1:],
	}

	return &message, nil
}

// WritePacket sends a milter response packet to socket stream
func (s *milterSession) WritePacket(msg *Message) error {
	// apply write deadline on the underlying socket
	if c, ok := s.sock.(net.Conn); ok {
		if err := c.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}
	}

	// calculate and write response length
	length := uint32(len(msg.Data) + 1)
	if err := binary.Write(s.bw, binary.BigEndian, length); err != nil {
		return err
	}

	// write response code
	if err := s.bw.WriteByte(msg.Code); err != nil {
		return err
	}

	// write response data
	if _, err := s.bw.Write(msg.Data); err != nil {
		return err
	}

	// flush data to network socket stream
	if err := s.bw.Flush(); err != nil {
		return err
	}

	return nil
}

// Process processes incoming milter commands
func (s *milterSession) Process(msg *Message) (Response, error) {
	switch msg.Code {
	case 'A':
		// abort current message and start over
		s.headers = nil
		s.macros = nil
		// do not send response
		return nil, nil

	case 'B':
		// body chunk
		return s.milter.BodyChunk(msg.Data, newModifier(s))

	case 'C':
		// new connection, get hostname
		Hostname := readCString(msg.Data)
		msg.Data = msg.Data[len(Hostname)+1:]
		// get protocol family
		protocolFamily := msg.Data[0]
		msg.Data = msg.Data[1:]
		// get port
		var Port uint16
		if protocolFamily == '4' || protocolFamily == '6' {
			if len(msg.Data) < 2 {
				return RespTempFail, nil
			}
			Port = binary.BigEndian.Uint16(msg.Data)
			msg.Data = msg.Data[2:]
			if protocolFamily == '6' {
				// trim IPv6 prefix when necessary
				msg.Data, _ = bytes.CutPrefix(msg.Data, []byte("IPv6:"))
			}
		}
		// get address
		Address := readCString(msg.Data)
		// convert address and port to human readable string
		family := map[byte]string{
			'U': "unknown",
			'L': "unix",
			'4': "tcp4",
			'6': "tcp6",
		}
		// run handler and return
		return s.milter.Connect(
			Hostname,
			family[protocolFamily],
			Port,
			net.ParseIP(Address),
			newModifier(s))

	case 'D':
		// define macros
		s.macros = make(map[string]string)
		// convert data to Go strings
		data := decodeCStrings(msg.Data[1:])
		if len(data) != 0 {
			// store data in a map
			for i := 0; i < len(data); i += 2 {
				s.macros[data[i]] = data[i+1]
			}
		}
		// do not send response
		return nil, nil

	case 'E':
		// call and return milter handler
		return s.milter.Body(newModifier(s))

	case 'H':
		// helo command
		name := strings.TrimSuffix(string(msg.Data), null)
		return s.milter.Helo(name, newModifier(s))

	case 'L':
		// make sure headers is initialized
		if s.headers == nil {
			s.headers = make(textproto.MIMEHeader)
		}
		// add new header to headers map
		HeaderData := decodeCStrings(msg.Data)
		if len(HeaderData) == 2 {
			s.headers.Add(HeaderData[0], HeaderData[1])
			// call and return milter handler
			return s.milter.Header(HeaderData[0], HeaderData[1], newModifier(s))
		}

	case 'M':
		// envelope from address
		envfrom := readCString(msg.Data)
		return s.milter.MailFrom(strings.Trim(envfrom, "<>"), newModifier(s))

	case 'N':
		// end of headers
		return s.milter.Headers(s.headers, newModifier(s))

	case 'O':
		// Negotiate actions bitmask back to MTA: union of caller-declared flags + SMFIF_REPLYCODE
		// so CustomReply (SMFIR_REPLYCODE 'y') is permitted (Postfix/sendmail expect 0x80, not 0x08).
		negotiatedActions := s.actions | OptReplyCode

		buffer := new(bytes.Buffer)

		// protocol version 2 + actions + protocol steps mask
		for _, value := range []uint32{2, uint32(negotiatedActions), uint32(s.protocol)} {
			if err := binary.Write(buffer, binary.BigEndian, value); err != nil {
				return nil, err
			}
		}
		// build and send packet
		return NewResponse('O', buffer.Bytes()), nil

	case 'Q':
		// client requested session close
		return nil, errCloseSession

	case 'R':
		// envelope to address
		envto := readCString(msg.Data)
		return s.milter.RcptTo(strings.Trim(envto, "<>"), newModifier(s))

	case 'T':
		// data, ignore

	default:
		// print error and close session
		if s.log != nil {
			s.log.Warnf("Unrecognized command code: %c", msg.Code)
		}
		return nil, errCloseSession
	}

	// by default continue with next milter message
	return RespContinue, nil
}

// HandleMilterCommands processes all milter commands in the same connection
func (s *milterSession) HandleMilterCommands() {
	// close session socket on exit
	defer s.sock.Close()

	for {
		// ReadPacket
		msg, err := s.ReadPacket()
		if err != nil {
			if err != io.EOF && s.log != nil {
				s.log.Warnf("Error reading milter command: %v", err)
			}
			return
		}

		// process command
		resp, err := s.Process(msg)
		if err != nil {
			if err != errCloseSession && s.log != nil {
				s.log.Warnf("Error performing milter command: %v", err)
			}
			return
		}

		// ignore empty responses
		if resp != nil {
			// send back response message
			if err = s.WritePacket(resp.Response()); err != nil {
				if s.log != nil {
					s.log.Warnf("Error writing packet: %v", err)
				}
				return
			}

			if !resp.Continue() {
				return
			}

		}
	}
}
