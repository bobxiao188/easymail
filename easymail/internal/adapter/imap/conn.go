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
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"easymail/internal/domain/shared"
)

const (
	imapReadDeadline    = 30 * time.Minute
	idleNewMsgCheckFreq = 2 * time.Second
	idlePingInterval    = 10 * time.Minute
)

const (
	stateNotAuth = iota
	stateAuth
	stateSelected
)

type protoConn struct {
	netConn      net.Conn
	remote       string
	br           *bufio.Reader
	ms           *MailSession
	log          Logger
	debug        bool
	state        int
	startTLSCaps bool
	tlsEnabled   bool

	idleMu       sync.Mutex
	idleStop     chan struct{}
	idleStopOnce sync.Once
}

func handleConn(c net.Conn, ms *MailSession, log Logger, startTLS bool, debug bool) {
	defer c.Close()
	remote := ""
	if c != nil && c.RemoteAddr() != nil {
		remote = c.RemoteAddr().String()
	}
	_, isTLS := c.(*tls.Conn)
	pc := &protoConn{
		netConn:      c,
		remote:       remote,
		br:           bufio.NewReader(c),
		ms:           ms,
		log:          log,
		debug:        debug,
		startTLSCaps: startTLS,
		tlsEnabled:   isTLS,
	}
	pc.infof("imap: connected remote=%s tls=%v", remote, isTLS)
	_ = pc.netConn.SetReadDeadline(time.Now().Add(imapReadDeadline))
	pc.writeGreeting()
	for {
		line, err := readLine(pc.br)
		if err != nil {
			pc.infof("imap: connection closed remote=%s account_id=%s err=%v", remote, pc.MailUserID(), err)
			return
		}
		if line == "" {
			continue
		}
		// Debug: log incoming command (mask LOGIN password)
		pc.logIncoming(line)
		if err := pc.handleLine(line); err != nil {
			return
		}
		_ = pc.netConn.SetReadDeadline(time.Now().Add(imapReadDeadline))
	}
}

func (c *protoConn) writeGreeting() {
	caps := "IMAP4rev1"
	if !c.tlsEnabled && c.startTLSCaps {
		caps += " STARTTLS"
	}
	c.writeUntagged("OK [" + caps + "] easymail ready")
}

func (c *protoConn) capabilityList() string {
	caps := "IMAP4rev1 UIDPLUS MOVE IDLE ENABLE SPECIAL-USE UTF8=ACCEPT NAMESPACE UNSELECT CHILDREN"
	if !c.tlsEnabled && c.startTLSCaps {
		caps += " STARTTLS LOGINDISABLED"
	}
	caps += " AUTH=PLAIN"
	return caps
}

func (c *protoConn) infof(format string, args ...interface{}) {
	if c == nil || c.log == nil {
		return
	}
	c.log.Printf(format, args...)
}

func (c *protoConn) debugf(format string, args ...interface{}) {
	if c == nil || !c.debug || c.log == nil {
		return
	}
	c.log.Printf("[IMAP-DEBUG] "+format, args...)
}

func (c *protoConn) logIncoming(line string) {
	if c == nil || !c.debug || c.log == nil {
		return
	}
	// Mask password in LOGIN commands for security
	s := line
	if strings.HasPrefix(strings.ToUpper(s), "LOGIN ") || strings.Contains(strings.ToUpper(line), " LOGIN ") {
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			parts[len(parts)-1] = "***"
			s = strings.Join(parts, " ")
		}
	}
	c.debugf("C: %s", s)
}

func (c *protoConn) logOutgoing(direction string, msg string) {
	if c == nil || !c.debug || c.log == nil {
		return
	}
	// Truncate very long lines (e.g. literal data)
	const maxLen = 500
	display := msg
	if len(display) > maxLen {
		display = display[:maxLen] + "...[truncated]"
	}
	c.debugf("%s: %s", direction, display)
}

func (c *protoConn) MailUserID() shared.GlobalID {
	if c == nil || c.ms == nil {
		return ""
	}
	return c.ms.MailUserID()
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func unquote(s string) string { return s }

func (c *protoConn) writeRaw(s string)             { c.logOutgoing("S", s); _, _ = io.WriteString(c.netConn, s) }
func (c *protoConn) writeUntagged(rest string)     { c.writeRaw("* " + rest + "\r\n") }
func (c *protoConn) writeContinuation(msg string)  { c.writeRaw("+ " + msg + "\r\n") }
func (c *protoConn) writeTaggedOK(tag, msg string) { c.writeRaw(fmt.Sprintf("%s OK %s\r\n", tag, msg)) }
func (c *protoConn) writeTaggedNO(tag, msg string) { c.writeRaw(fmt.Sprintf("%s NO %s\r\n", tag, msg)) }
func (c *protoConn) writeTaggedBAD(tag, msg string) {
	c.writeRaw(fmt.Sprintf("%s BAD %s\r\n", tag, msg))
}

// ---- Command dispatch ----

func (c *protoConn) handleLine(line string) error {
	toks, err := tokenize(line)
	if err != nil || len(toks) < 2 {
		c.writeTaggedBAD("*", "syntax error")
		return nil
	}
	tag := toks[0]
	cmd := strings.ToUpper(toks[1])
	args := toks[2:]

	if cmd == "UID" && len(args) >= 1 {
		return c.dispatchUID(tag, args)
	}
	if cmd == "IDLE" {
		return c.handleIDLE(tag)
	}
	return c.dispatchNormal(tag, cmd, args)
}

func (c *protoConn) dispatchUID(tag string, args []string) error {
	sub := strings.ToUpper(args[0])
	args = args[1:]
	switch sub {
	case "FETCH":
		return c.handleFetch(tag, args, true)
	case "STORE":
		return c.handleStore(tag, args, true)
	case "SEARCH":
		return c.handleSearch(tag, args, true)
	case "COPY":
		return c.handleCopy(tag, args, true)
	case "MOVE":
		return c.handleMove(tag, args, true)
	case "EXPUNGE":
		return c.handleExpunge(tag, args, true)
	default:
		c.writeTaggedBAD(tag, "UID command not supported")
		return nil
	}
}

func (c *protoConn) dispatchNormal(tag, cmd string, args []string) error {
	switch cmd {
	case "STARTTLS":
		return c.handleSTARTTLS(tag)
	case "CAPABILITY":
		c.handleCapability(tag)
		return nil
	case "NOOP":
		c.writeTaggedOK(tag, "NOOP completed")
		return nil
	case "CHECK":
		c.writeTaggedOK(tag, "CHECK completed")
		return nil
	case "SEARCH":
		return c.handleSearch(tag, args, false)
	case "LOGOUT":
		c.infof("imap: logout remote=%s account_id=%s", c.remote, c.MailUserID())
		c.writeUntagged("BYE easymail logging out")
		c.writeTaggedOK(tag, "LOGOUT completed")
		return io.EOF
	case "LOGIN":
		return c.handleLogin(tag, args)
	case "AUTHENTICATE":
		return c.handleAuthenticate(tag, args)
	case "ENABLE":
		return c.handleEnable(tag, args)
	case "NAMESPACE":
		return c.handleNamespace(tag)
	case "LIST":
		return c.handleList(tag, args, false)
	case "XLIST":
		return c.handleList(tag, args, true)
	case "LSUB":
		return c.handleLSub(tag, args)
	case "SELECT":
		return c.handleSelect(tag, args, false)
	case "EXAMINE":
		return c.handleSelect(tag, args, true)
	case "UNSELECT":
		return c.handleUnselect(tag)
	case "CLOSE":
		return c.handleClose(tag)
	case "STATUS":
		return c.handleStatus(tag, args)
	case "FETCH":
		return c.handleFetch(tag, args, false)
	case "STORE":
		return c.handleStore(tag, args, false)
	case "COPY":
		return c.handleCopy(tag, args, false)
	case "MOVE":
		return c.handleMove(tag, args, false)
	case "EXPUNGE":
		return c.handleExpunge(tag, args, false)
	case "APPEND":
		return c.handleAppend(tag, args)
	case "CREATE":
		return c.handleCreate(tag, args)
	case "DELETE":
		return c.handleDelete(tag, args)
	case "RENAME":
		return c.handleRename(tag, args)
	case "SUBSCRIBE":
		return c.handleSubscribe(tag, args)
	case "UNSUBSCRIBE":
		return c.handleUnsubscribe(tag, args)
	default:
		c.writeTaggedBAD(tag, "command not supported")
		return nil
	}
}

// ---- STARTTLS ----

func (c *protoConn) handleSTARTTLS(tag string) error {
	if c.tlsEnabled {
		c.writeTaggedBAD(tag, "TLS already active")
		return nil
	}
	if c.ms.TLSConfig() == nil {
		c.writeTaggedNO(tag, "STARTTLS not available")
		return nil
	}
	if c.state != stateNotAuth {
		c.writeTaggedBAD(tag, "STARTTLS only allowed in non-authenticated state")
		return nil
	}
	c.writeTaggedOK(tag, "Begin TLS negotiation now")
	if flusher, ok := c.netConn.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	tlsConn := tls.Server(c.netConn, c.ms.TLSConfig())
	if err := tlsConn.Handshake(); err != nil {
		c.infof("imap: STARTTLS handshake failed remote=%s err=%v", c.remote, err)
		return io.EOF
	}
	c.netConn = tlsConn
	c.br = bufio.NewReader(tlsConn)
	c.tlsEnabled = true
	return nil
}

// ---- CAPABILITY ----

func (c *protoConn) handleCapability(tag string) {
	caps := c.capabilityList()
	c.writeUntagged("CAPABILITY " + caps)
	c.writeTaggedOK(tag, "CAPABILITY completed")
}

// ---- LOGIN / AUTHENTICATE ----

func (c *protoConn) handleLogin(tag string, args []string) error {
	if c.state != stateNotAuth {
		c.writeTaggedBAD(tag, "already authenticated")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "LOGIN expects username password")
		return nil
	}
	user := unquote(args[0])
	pass := unquote(args[1])
	if err := c.ms.Login(user, pass); err != nil {
		c.writeTaggedNO(tag, "authentication failed")
		return nil
	}
	c.state = stateAuth
	c.writeTaggedOK(tag, "LOGIN completed")
	return nil
}

func (c *protoConn) handleAuthenticate(tag string, args []string) error {
	if c.state != stateNotAuth {
		c.writeTaggedBAD(tag, "already authenticated")
		return nil
	}
	if len(args) < 1 {
		c.writeTaggedBAD(tag, "AUTHENTICATE expects mechanism")
		return nil
	}
	mech := strings.ToUpper(args[0])
	if mech == "PLAIN" {
		return c.handleAuthPlain(tag, args)
	}
	c.writeTaggedNO(tag, "unsupported mechanism")
	return nil
}

func (c *protoConn) handleAuthPlain(tag string, args []string) error {
	if len(args) > 1 {
		return c.doPlainAuth(tag, args[1])
	}
	c.writeRaw("+ \r\n")
	if flusher, ok := c.netConn.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	resp, err := readLine(c.br)
	if err != nil {
		return io.EOF
	}
	if resp == "*" {
		c.writeTaggedBAD(tag, "authentication cancelled")
		return nil
	}
	return c.doPlainAuth(tag, resp)
}

func (c *protoConn) doPlainAuth(tag, resp string) error {
	parts := strings.Split(resp, "\x00")
	if len(parts) < 3 {
		c.writeTaggedBAD(tag, "invalid PLAIN response")
		return nil
	}
	if err := c.ms.Login(parts[1], parts[2]); err != nil {
		c.writeTaggedNO(tag, "authentication failed")
		return nil
	}
	c.state = stateAuth
	c.writeTaggedOK(tag, "AUTHENTICATE completed")
	return nil
}

// ---- ENABLE ----

func (c *protoConn) handleEnable(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	var enabled []string
	for _, cap := range args {
		if strings.ToUpper(cap) == "UTF8=ACCEPT" {
			enabled = append(enabled, "UTF8=ACCEPT")
		}
	}
	c.writeUntagged("ENABLED " + strings.Join(enabled, " "))
	c.writeTaggedOK(tag, "ENABLE completed")
	return nil
}

// ---- NAMESPACE ----

func (c *protoConn) handleNamespace(tag string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	c.writeUntagged(`NAMESPACE (("" "/")) NIL NIL`)
	c.writeTaggedOK(tag, "NAMESPACE completed")
	return nil
}

// ---- LIST / LSUB ----

func (c *protoConn) handleList(tag string, args []string, isXlist bool) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "LIST needs reference mailbox")
		return nil
	}
	ref := unquote(args[0])
	pat := unquote(args[1])
	returnSpecialUse := isXlist
	returnChildren := false
	extArgs := args[2:]
	for i := 0; i < len(extArgs); i++ {
		switch strings.ToUpper(extArgs[i]) {
		case "RETURN":
			if i+1 < len(extArgs) {
				subArgs := strings.TrimPrefix(extArgs[i+1], "(")
				subArgs = strings.TrimSuffix(subArgs, ")")
				for _, s := range strings.Fields(subArgs) {
					switch strings.ToUpper(s) {
					case "SPECIAL-USE":
						returnSpecialUse = true
					case "CHILDREN":
						returnChildren = true
					case "SUBSCRIBED":
					}
				}
				i++
			}
		case "SUBSCRIBED":
		}
	}

	entries, err := c.ms.ListMailboxes(ref, []string{pat})
	if err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	delim := "/"
	for _, e := range entries {
		attrs := "()"
		var attrParts []string
		if returnSpecialUse {
			special := c.ms.SpecialUseForMailbox(e.Mailbox)
			if special != "" {
				attrParts = append(attrParts, special)
			}
		}
		if returnChildren {
			if c.ms.MailboxHasChildren(e.Mailbox) {
				attrParts = append(attrParts, `\HasChildren`)
			} else {
				attrParts = append(attrParts, `\HasNoChildren`)
			}
		}
		if len(attrParts) > 0 {
			attrs = "(" + strings.Join(attrParts, " ") + ")"
		}
		c.writeUntagged(fmt.Sprintf(`LIST %s "%s" %s`, attrs, delim, quoteIMAPString(e.Mailbox)))
	}
	c.writeTaggedOK(tag, "LIST completed")
	return nil
}

func (c *protoConn) handleLSub(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "LSUB needs reference mailbox")
		return nil
	}
	ref := unquote(args[0])
	pat := unquote(args[1])
	entries, err := c.ms.ListSubscribedMailboxes(ref, []string{pat})
	if err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	delim := "/"
	for _, e := range entries {
		c.writeUntagged(fmt.Sprintf(`LSUB () "%s" %s`, delim, quoteIMAPString(e.Mailbox)))
	}
	c.writeTaggedOK(tag, "LSUB completed")
	return nil
}

// ---- SELECT / EXAMINE ----

func (c *protoConn) handleSelect(tag string, args []string, readOnly bool) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 1 {
		c.writeTaggedBAD(tag, "needs mailbox name")
		return nil
	}
	mbox := unquote(args[0])
	cmdName := "SELECT"
	if readOnly {
		cmdName = "EXAMINE"
	}
	res, err := c.ms.Select(mbox)
	if err != nil {
		c.debugf("%s failed: mbox=%q err=%v", cmdName, mbox, err)
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.debugf("%s OK: mbox=%q messages=%d recent=%d unseen=%d uidvalidity=%d uidnext=%d",
		cmdName, mbox, res.NumMessages, res.NumRecent, res.FirstUnseenSeqNum, res.UIDValidity, res.UIDNext)
	c.writeUntagged(fmt.Sprintf("%d EXISTS", res.NumMessages))
	c.writeUntagged(fmt.Sprintf("%d RECENT", res.NumRecent))
	c.writeUntagged(fmt.Sprintf("OK [UNSEEN %d] unseen", res.FirstUnseenSeqNum))
	c.writeUntagged(fmt.Sprintf("OK [UIDVALIDITY %d] UIDs valid", res.UIDValidity))
	c.writeUntagged(fmt.Sprintf("OK [UIDNEXT %d] next UID", res.UIDNext))
	c.writeUntagged(`FLAGS (\Answered \Flagged \Deleted \Seen \Draft)`)
	c.writeUntagged(`OK [PERMANENTFLAGS (\Answered \Flagged \Deleted \Seen \Draft \*)] Permanent flags`)
	if readOnly {
		c.writeTaggedOK(tag, "[READ-ONLY] EXAMINE completed")
	} else {
		c.state = stateSelected
		c.writeTaggedOK(tag, "[READ-WRITE] SELECT completed")
	}
	return nil
}

// ---- UNSELECT / CLOSE ----

func (c *protoConn) handleUnselect(tag string) error {
	if c.state != stateSelected {
		c.writeTaggedBAD(tag, "no mailbox selected")
		return nil
	}
	if err := c.ms.Unselect(); err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.state = stateAuth
	c.writeTaggedOK(tag, "UNSELECT completed")
	return nil
}

func (c *protoConn) handleClose(tag string) error {
	if c.state != stateSelected {
		c.writeTaggedBAD(tag, "no mailbox selected")
		return nil
	}
	if err := c.ms.Close(); err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.state = stateAuth
	c.writeTaggedOK(tag, "CLOSE completed")
	return nil
}

// ---- STATUS ----

func (c *protoConn) handleStatus(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "STATUS needs mailbox items")
		return nil
	}
	mbox := unquote(args[0])
	itemsStr := strings.Join(args[1:], " ")
	res, err := c.ms.Status(mbox, parseStatusItems(itemsStr))
	if err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	var parts []string
	if res.Messages != nil {
		parts = append(parts, fmt.Sprintf("MESSAGES %d", *res.Messages))
	}
	if res.Recent != nil {
		parts = append(parts, fmt.Sprintf("RECENT %d", *res.Recent))
	}
	if res.UIDNext != nil {
		parts = append(parts, fmt.Sprintf("UIDNEXT %d", *res.UIDNext))
	}
	if res.UIDValidity != nil {
		parts = append(parts, fmt.Sprintf("UIDVALIDITY %d", *res.UIDValidity))
	}
	if res.Unseen != nil {
		parts = append(parts, fmt.Sprintf("UNSEEN %d", *res.Unseen))
	}
	c.writeUntagged(fmt.Sprintf("STATUS %s (%s)", quoteIMAPString(mbox), strings.Join(parts, " ")))
	c.writeTaggedOK(tag, "STATUS completed")
	return nil
}

// ---- SEARCH ----

func (c *protoConn) handleSearch(tag string, args []string, uidCmd bool) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if c.state != stateSelected {
		c.writeTaggedBAD(tag, "not selected")
		return nil
	}
	criteria := strings.Join(args, " ")
	ids, err := c.ms.Search(criteria, uidCmd)
	if err != nil {
		c.debugf("SEARCH failed: criteria=%q err=%v", criteria, err)
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	if len(ids) > 0 {
		c.debugf("SEARCH found %d results: %v", len(ids), ids)
		c.writeUntagged("SEARCH " + formatSeqNumbers(ids))
	} else {
		c.debugf("SEARCH found 0 results: criteria=%q", criteria)
		c.writeUntagged("SEARCH")
	}
	c.writeTaggedOK(tag, "SEARCH completed")
	return nil
}

// ---- FETCH ----

func (c *protoConn) handleFetch(tag string, args []string, uidCmd bool) error {
	c.debugf("[FETCH-DEBUG] tag=%s uidCmd=%v args=%v", tag, uidCmd, args)
	if c.state != stateSelected {
		c.writeTaggedBAD(tag, "no mailbox selected")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "FETCH needs set items")
		return nil
	}
	setStr := args[0]
	listStr := strings.Join(args[1:], " ")
	opts, err := parseFetchList(listStr)
	if err != nil {
		c.writeTaggedBAD(tag, err.Error())
		return nil
	}
	opts.UID = opts.UID || uidCmd
	c.debugf("FETCH opts: uid=%v flags=%v envelope=%v rfc822size=%v bodypeek=%v bodynotpeek=%v bodyitem=%q",
		opts.UID, opts.Flags, opts.Envelope, opts.RFC822Size, opts.BodyPeek, opts.BodyNotPeek, opts.BodyItem)
	var seqSet SeqSet
	var uidSet UIDSet
	if uidCmd {
		uidSet, err = parseUIDSet(setStr)
	} else {
		seqSet, err = parseSeqSet(setStr)
	}
	if err != nil {
		c.writeTaggedBAD(tag, err.Error())
		return nil
	}
	kind := NumKindSeq
	if uidCmd {
		kind = NumKindUID
	}
	c.infof("imap: fetch remote=%s set=%s", c.remote, setStr)
	var fetchedCount uint32
	err = c.ms.Fetch(kind, seqSet, uidSet, opts, func(seq uint32, mailID int64) error {
		fetchedCount++
		fd, fErr := c.ms.LoadFetchMessageData(seq, mailID, opts)
		if fErr != nil {
			return fErr
		}
		return c.writeFetchLine(fd, opts)
	})
	if err != nil {
		c.debugf("FETCH failed: set=%q err=%v", setStr, err)
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.debugf("FETCH OK: set=%s uidCmd=%v fetched=%d", setStr, uidCmd, fetchedCount)
	c.writeTaggedOK(tag, "FETCH completed")
	return nil
}

func (c *protoConn) writeFetchLine(fd *FetchMessageData, opts FetchOptions) error {
	var items []string
	if opts.UID {
		items = append(items, "UID", uint32dec(fd.UID))
	}
	if opts.Flags {
		items = append(items, "FLAGS", formatFlagsList(fd.Flags))
	}
	if opts.Envelope {
		items = append(items, "ENVELOPE", fd.EnvelopeWire)
	}
	if opts.RFC822Size {
		items = append(items, "RFC822.SIZE", fmt.Sprintf("%d", fd.RFC822Size))
	}
	if opts.InternalDate {
		items = append(items, "INTERNALDATE", quoteIMAPString(fd.InternalDate.Format("02-Jan-2006 15:04:05 -0700")))
	}

	if !opts.BodyPeek && !opts.BodyNotPeek {
		c.debugf("FETCH no-body: seq=%d uid=%d items=%s", fd.Seq, fd.UID, strings.Join(items, " "))
		c.writeRaw(fmt.Sprintf("* %d FETCH (%s)\r\n", fd.Seq, strings.Join(items, " ")))
		return nil
	}
	if fd.OpenRaw == nil {
		return fmt.Errorf("missing body")
	}
	rc, sz, err := fd.OpenRaw()
	if err != nil {
		return err
	}
	defer rc.Close()
	body, err := io.ReadAll(io.LimitReader(rc, sz))
	if err != nil {
		return err
	}
	itemName := opts.BodyItem
	if itemName == "" {
		itemName = "BODY[]"
		if opts.BodyPeek {
			itemName = "BODY.PEEK[]"
		}
	}
	// RFC 3501 §6.4.5: .PEEK is a modifier, not part of the response data item name
	respItemName := strings.Replace(itemName, ".PEEK", "", 1)
	sendBody := body
	upItem := strings.ToUpper(itemName)

	// Handle HEADER, HEADER.FIELDS, HEADER.FIELDS.NOT
	if strings.Contains(upItem, "[HEADER") {
		// Locate end of header section
		hdrEnd := bytes.Index(sendBody, []byte("\r\n\r\n"))
		if hdrEnd < 0 {
			hdrEnd = bytes.Index(sendBody, []byte("\n\n"))
		}
		if hdrEnd >= 0 {
			sendBody = sendBody[:hdrEnd]
		}

		// HEADER.FIELDS or HEADER.FIELDS.NOT — filter specific header lines
		if strings.Contains(upItem, "HEADER.FIELDS") {
			// Parse the field list from the body item:
			//   BODY.PEEK[HEADER.FIELDS (From To Subject)]
			//   BODY.PEEK[HEADER.FIELDS.NOT (X-Bogosity)]
			fieldStart := strings.Index(upItem, "(")
			fieldEnd := strings.LastIndex(upItem, ")")
			var fieldNames []string
			if fieldStart >= 0 && fieldEnd > fieldStart {
				fieldStr := strings.TrimSpace(itemName[fieldStart+1 : fieldEnd])
				if fieldStr != "" {
					fieldNames = strings.Fields(fieldStr)
				}
			}

			negate := strings.Contains(upItem, "FIELDS.NOT")
			lookup := make(map[string]bool, len(fieldNames))
			for _, f := range fieldNames {
				lookup[strings.ToUpper(f)] = true
			}

			lines := bytes.Split(sendBody, []byte("\n"))
			var filtered bytes.Buffer
			for _, line := range lines {
				line = bytes.TrimRight(line, "\r")
				colonIdx := bytes.Index(line, []byte(":"))
				if colonIdx < 0 {
					// Continuation / folded header — keep if we already have output
					if filtered.Len() > 0 {
						filtered.Write(line)
						filtered.WriteString("\r\n")
					}
					continue
				}
				fn := strings.TrimSpace(string(line[:colonIdx]))
				matches := lookup[strings.ToUpper(fn)]
				if negate == matches {
					continue
				}
				filtered.Write(line)
				filtered.WriteString("\r\n")
			}
			sendBody = filtered.Bytes()
		}

		// Append the blank line separator after the header section
		sendBody = append(sendBody, []byte("\r\n")...)
	}
	c.debugf("FETCH body: seq=%d uid=%d item=%s rawSize=%d sendSize=%d",
		fd.Seq, fd.UID, respItemName, sz, len(sendBody))
	var head strings.Builder
	fmt.Fprintf(&head, "* %d FETCH (", fd.Seq)
	head.WriteString(strings.Join(items, " "))
	if len(items) > 0 {
		head.WriteString(" ")
	}
	head.WriteString(respItemName)
	fmt.Fprintf(&head, " {%d}\r\n", len(sendBody))
	c.writeRaw(head.String())
	if _, err := c.netConn.Write(sendBody); err != nil {
		return err
	}
	c.writeRaw(")\r\n")
	return nil
}

// ---- STORE ----

func (c *protoConn) handleStore(tag string, args []string, uidCmd bool) error {
	if c.state != stateSelected {
		c.writeTaggedNO(tag, "no mailbox selected")
		return nil
	}
	if len(args) < 3 {
		c.writeTaggedBAD(tag, "STORE needs set item flags")
		return nil
	}
	setStr := args[0]
	opItem := args[1]
	flagStr := strings.Join(args[2:], " ")
	op, silent := parseStoreItem(opItem)
	flags := parseFlagList(flagStr)
	sf := &StoreFlags{Op: op, Silent: silent, Flags: flags}
	var seqSet SeqSet
	var uidSet UIDSet
	var err error
	if uidCmd {
		uidSet, err = parseUIDSet(setStr)
	} else {
		seqSet, err = parseSeqSet(setStr)
	}
	if err != nil {
		c.writeTaggedBAD(tag, err.Error())
		return nil
	}
	kind := NumKindSeq
	if uidCmd {
		kind = NumKindUID
	}
	err = c.ms.Store(kind, seqSet, uidSet, sf)
	if err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	if !sf.Silent {
		_ = c.ms.Fetch(kind, seqSet, uidSet, FetchOptions{Flags: true, UID: uidCmd}, func(seq uint32, mailID int64) error {
			fd, fErr := c.ms.LoadFetchMessageData(seq, mailID, FetchOptions{Flags: true})
			if fErr != nil {
				return nil
			}
			return c.writeFetchLine(fd, FetchOptions{Flags: true, UID: uidCmd})
		})
	}
	c.writeTaggedOK(tag, "STORE completed")
	return nil
}

// ---- COPY ----

func (c *protoConn) handleCopy(tag string, args []string, uidCmd bool) error {
	if c.state != stateSelected {
		c.writeTaggedNO(tag, "no mailbox selected")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "COPY needs set destination")
		return nil
	}
	setStr := args[0]
	dest := unquote(args[1])
	var err error
	if uidCmd {
		uidSet, _ := parseUIDSet(setStr)
		_, err = c.ms.CopyUID(uidSet, dest)
	} else {
		seqSet, _ := parseSeqSet(setStr)
		_, err = c.ms.CopySeq(seqSet, dest)
	}
	if err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.writeTaggedOK(tag, "COPY completed")
	return nil
}

// ---- MOVE ----

func (c *protoConn) handleMove(tag string, args []string, uidCmd bool) error {
	if c.state != stateSelected {
		c.writeTaggedNO(tag, "no mailbox selected")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "MOVE needs set destination")
		return nil
	}
	setStr := args[0]
	dest := unquote(args[1])
	if uidCmd {
		uidSet, _ := parseUIDSet(setStr)
		expunged, _, err := c.ms.MoveUID(uidSet, dest)
		for _, seq := range expunged {
			c.writeUntagged(fmt.Sprintf("%d EXPUNGE", seq))
		}
		if err != nil {
			c.writeTaggedNO(tag, err.Error())
			return nil
		}
	} else {
		seqSet, _ := parseSeqSet(setStr)
		expunged, err := c.ms.MoveSeq(seqSet, dest)
		for _, seq := range expunged {
			c.writeUntagged(fmt.Sprintf("%d EXPUNGE", seq))
		}
		if err != nil {
			c.writeTaggedNO(tag, err.Error())
			return nil
		}
	}
	c.writeTaggedOK(tag, "MOVE completed")
	return nil
}

// ---- EXPUNGE ----

func (c *protoConn) handleExpunge(tag string, args []string, uidCmd bool) error {
	if c.state != stateSelected {
		c.writeTaggedNO(tag, "no mailbox selected")
		return nil
	}
	var expunged []uint32
	var err error
	if uidCmd {
		if len(args) < 1 {
			c.writeTaggedBAD(tag, "UID EXPUNGE needs set")
			return nil
		}
		uidSet, perr := parseUIDSet(args[0])
		if perr != nil {
			c.writeTaggedBAD(tag, perr.Error())
			return nil
		}
		expunged, err = c.ms.ExpungeUID(uidSet)
	} else {
		expunged, err = c.ms.Expunge()
	}
	if err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	for _, seq := range expunged {
		c.writeUntagged(fmt.Sprintf("%d EXPUNGE", seq))
	}
	c.writeTaggedOK(tag, "EXPUNGE completed")
	return nil
}

// ---- APPEND ----

func (c *protoConn) handleAppend(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "APPEND needs mailbox literal")
		return nil
	}
	mbox := unquote(args[0])
	var literalSize int
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "{") && strings.HasSuffix(a, "}") {
			fmt.Sscanf(a, "{%d}", &literalSize)
			break
		}
	}
	if literalSize <= 0 {
		c.writeTaggedBAD(tag, "APPEND needs literal size")
		return nil
	}
	c.writeRaw("+ Ready for literal data\r\n")
	buf := make([]byte, literalSize)
	n, err := io.ReadFull(c.br, buf)
	if err != nil {
		return io.EOF
	}
	uid, err := c.ms.Append(mbox, nil, "", buf[:n])
	if err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.writeTaggedOK(tag, fmt.Sprintf("[APPENDUID %d %d] APPEND completed", c.ms.UIDValidity(), uid))
	return nil
}

// ---- CREATE / DELETE / RENAME ----

func (c *protoConn) handleCreate(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 1 {
		c.writeTaggedBAD(tag, "CREATE needs mailbox name")
		return nil
	}
	if err := c.ms.CreateMailbox(unquote(args[0])); err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.writeTaggedOK(tag, "CREATE completed")
	return nil
}

func (c *protoConn) handleDelete(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 1 {
		c.writeTaggedBAD(tag, "DELETE needs mailbox name")
		return nil
	}
	if err := c.ms.DeleteMailbox(unquote(args[0])); err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.writeTaggedOK(tag, "DELETE completed")
	return nil
}

func (c *protoConn) handleRename(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 2 {
		c.writeTaggedBAD(tag, "RENAME needs old new mailbox names")
		return nil
	}
	if err := c.ms.RenameMailbox(unquote(args[0]), unquote(args[1])); err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.writeTaggedOK(tag, "RENAME completed")
	return nil
}

// ---- SUBSCRIBE / UNSUBSCRIBE ----

func (c *protoConn) handleSubscribe(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 1 {
		c.writeTaggedBAD(tag, "SUBSCRIBE needs mailbox name")
		return nil
	}
	if err := c.ms.Subscribe(unquote(args[0])); err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.writeTaggedOK(tag, "SUBSCRIBE completed")
	return nil
}

func (c *protoConn) handleUnsubscribe(tag string, args []string) error {
	if c.state == stateNotAuth {
		c.writeTaggedBAD(tag, "not authenticated")
		return nil
	}
	if len(args) < 1 {
		c.writeTaggedBAD(tag, "UNSUBSCRIBE needs mailbox name")
		return nil
	}
	if err := c.ms.Unsubscribe(unquote(args[0])); err != nil {
		c.writeTaggedNO(tag, err.Error())
		return nil
	}
	c.writeTaggedOK(tag, "UNSUBSCRIBE completed")
	return nil
}

// ---- IDLE (RFC 2177) ----

func (c *protoConn) handleIDLE(tag string) error {
	if c.state != stateSelected {
		c.writeTaggedBAD(tag, "IDLE requires selected mailbox")
		return nil
	}
	c.writeContinuation("idling")
	c.ms.SetNotifyConn(c)
	defer c.ms.SetNotifyConn(nil)

	// Reset read deadline so IDLE can last beyond the previous command's deadline.
	_ = c.netConn.SetReadDeadline(time.Now().Add(imapReadDeadline))

	c.idleMu.Lock()
	c.idleStop = make(chan struct{})
	stopCh := c.idleStop
	c.idleMu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(idleNewMsgCheckFreq)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				// Keep the read deadline alive so IDLE isn't killed after 30 min.
				_ = c.netConn.SetReadDeadline(time.Now().Add(imapReadDeadline))
				newMsgs, err := c.ms.PollNewMessages()
				if err != nil {
					c.infof("imap: idle poll error err=%v", err)
					return
				}
				for _, nm := range newMsgs {
					c.writeUntagged(fmt.Sprintf("%d EXISTS", nm.Exists))
					c.writeUntagged(fmt.Sprintf("%d RECENT", nm.Recent))
				}
			}
		}
	}()

	// Wait for DONE line
	for {
		line, err := readLine(c.br)
		if err != nil {
			c.idleMu.Lock()
			if c.idleStop != nil {
				close(c.idleStop)
				c.idleStop = nil
			}
			c.idleMu.Unlock()
			<-done
			return io.EOF
		}
		if strings.ToUpper(strings.TrimSpace(line)) == "DONE" {
			c.idleMu.Lock()
			if c.idleStop != nil {
				close(c.idleStop)
				c.idleStop = nil
			}
			c.idleMu.Unlock()
			<-done
			c.writeTaggedOK(tag, "IDLE terminated")
			return nil
		}
	}
}

// ---- Helpers ----

func parseStatusItems(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimSuffix(s, ")"), "(")
	var items []string
	for _, it := range strings.Fields(s) {
		items = append(items, strings.ToUpper(it))
	}
	return items
}

func formatSeqNumbers(ids []uint32) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, " ")
}

func storeOpName(op StoreOp) string {
	switch op {
	case StoreOpAdd:
		return "add"
	case StoreOpRemove:
		return "remove"
	default:
		return "replace"
	}
}
