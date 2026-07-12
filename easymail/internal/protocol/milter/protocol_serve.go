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
	"context"
	"net"

	"easymail/pkg/logger/easylog"
)

// ProtocolServe accepts Postfix milter connections on ln; each connection gets a fresh handler from newMilter().
func ProtocolServe(ctx context.Context, ln net.Listener, milterHandle func() Milter, actions OptAction, protocol OptProtocol, log *easylog.Logger) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			log.Errorf("milter accept error: %v", err)
			continue
		}
		go func(c net.Conn) {
			defer func() {
				if r := recover(); r != nil {
					log.Warnf("milter session panic: %v", r)
				}
				c.Close()
			}()
			bw := bufio.NewWriter(c)
			s := &milterSession{
				actions:  actions,
				protocol: protocol,
				sock:     c,
				bw:       bw,
				milter:   milterHandle(),
				log:      log,
			}
			if log != nil {
				log.Debugf("milter session started remote=%s", c.RemoteAddr().String())
			}
			s.HandleMilterCommands()
		}(conn)
	}
}
