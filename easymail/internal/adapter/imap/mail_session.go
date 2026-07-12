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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	managementSvc "easymail/internal/app/management"
	"easymail/internal/domain/messaging"
	mailservice "easymail/internal/domain/messaging/service"
	"easymail/internal/domain/shared"
	"easymail/pkg/constants"
)

var errAuthFailed = errors.New("imap: authentication failed")

type SessionDeps struct {
	Auth managementSvc.MailUserAuthService
	Mail mailservice.Backend
	Ctx  context.Context
	TLS  *tls.Config
}

type MailSession struct {
	deps SessionDeps

	mailUserID shared.GlobalID

	folderID        int64
	folderIMAP      string
	msgs            []int64
	uidValidity     uint32
	selectTime      time.Time
	selectMsgCnt    int
	cachedFolders   []mailservice.FolderDTO
	cachedFoldersAt time.Time

	notifyConn *protoConn

	// Tracks last-known message count per folder, used to compute RECENT
	// in SELECT and STATUS responses.
	folderMsgCounts map[int64]int
}

func NewMailSession(deps SessionDeps) *MailSession {
	return &MailSession{
		deps:            deps,
		folderMsgCounts: make(map[int64]int),
	}
}

func (s *MailSession) TLSConfig() *tls.Config      { return s.deps.TLS }
func (s *MailSession) SetNotifyConn(pc *protoConn) { s.notifyConn = pc }
func (s *MailSession) MailUserID() shared.GlobalID { return s.mailUserID }
func (s *MailSession) UIDValidity() uint32         { return s.uidValidity }
func (s *MailSession) SelectedMailbox() string     { return s.folderIMAP }

func (s *MailSession) ensureAuth() error {
	if s.mailUserID == "" {
		return errors.New("not authenticated")
	}
	return nil
}

func (s *MailSession) Login(username, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return errAuthFailed
	}
	acc, err := s.deps.Auth.Authenticate(s.deps.Ctx, username, password)
	if err != nil {
		return errAuthFailed
	}
	s.mailUserID = acc.ID
	return nil
}

func (s *MailSession) loadFolders() ([]mailservice.FolderDTO, error) {
	if s.cachedFolders != nil && time.Since(s.cachedFoldersAt) < 30*time.Second {
		return s.cachedFolders, nil
	}
	folders, err := s.deps.Mail.ListFolders(s.deps.Ctx, s.mailUserID)
	if err != nil {
		return nil, err
	}
	s.cachedFolders = folders
	s.cachedFoldersAt = time.Now()
	return folders, nil
}

func (s *MailSession) resolveFolderName(dest string) (int64, error) {
	folders, err := s.loadFolders()
	if err != nil {
		return 0, err
	}
	fid, _, found := findFolderID(folders, dest)
	if !found {
		return 0, fmt.Errorf("mailbox %q does not exist", dest)
	}
	return fid, nil
}

// ---- LIST ----

func (s *MailSession) ListMailboxes(ref string, patterns []string) ([]MailboxEntry, error) {
	if err := s.ensureAuth(); err != nil {
		return nil, err
	}
	folders, err := s.loadFolders()
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}
	delim := rune('/')
	var out []MailboxEntry
	for _, pat := range patterns {
		if pat == "" {
			pat = "*"
		}
		for _, f := range folders {
			name := folderDisplayName(f)
			if !matchList(name, delim, ref, pat) {
				continue
			}
			out = append(out, MailboxEntry{Delim: delim, Mailbox: name})
		}
	}
	return out, nil
}

var kindToSpecialUse = map[constants.FolderID]string{
	constants.Inbox: `\Inbox`,
	constants.Sent:  `\Sent`,
	constants.Draft: `\Drafts`,
	constants.Trash: `\Trash`,
	constants.Spam:  `\Junk`,
}

func (s *MailSession) SpecialUseForMailbox(mbox string) string {
	folders, _ := s.loadFolders()
	for _, f := range folders {
		if strings.EqualFold(folderDisplayName(f), mbox) {
			if su, ok := kindToSpecialUse[f.Kind]; ok {
				return su
			}
		}
	}
	return ""
}

func (s *MailSession) MailboxHasChildren(mbox string) bool {
	folders, _ := s.loadFolders()
	prefix := strings.ToLower(mbox) + "/"
	for _, f := range folders {
		dn := strings.ToLower(folderDisplayName(f))
		if strings.HasPrefix(dn, prefix) {
			return true
		}
	}
	return false
}

func (s *MailSession) ListSubscribedMailboxes(ref string, patterns []string) ([]MailboxEntry, error) {
	return s.ListMailboxes(ref, patterns)
}

// ---- SELECT / EXAMINE ----

type SelectResult struct {
	Flags             []string
	PermanentFlags    []string
	NumMessages       uint32
	NumRecent         uint32
	FirstUnseenSeqNum uint32
	UIDNext           uint32
	UIDValidity       uint32
}

func (s *MailSession) Select(mailbox string) (*SelectResult, error) {
	if err := s.ensureAuth(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(mailbox)
	folders, err := s.loadFolders()
	if err != nil {
		return nil, err
	}
	fid, uv, found := findFolderID(folders, name)
	if !found {
		return nil, errors.New("mailbox does not exist")
	}
	msgs, err := loadAllMailIDs(s.deps.Ctx, s.deps.Mail, s.mailUserID, fid)
	if err != nil {
		return nil, err
	}
	s.folderID = fid
	s.folderIMAP = name
	s.msgs = msgs
	// Prefer the persisted UIDVALIDITY; fall back to the deterministic value
	// only for folders that have not been migrated yet.
	if uv != 0 {
		s.uidValidity = uv
	} else {
		s.uidValidity = uidValidityForFolder(fid)
	}
	s.selectTime = time.Now()
	s.selectMsgCnt = len(msgs)

	var maxUID uint32
	for _, id := range msgs {
		u := uint32FromMailID(id)
		if u > maxUID {
			maxUID = u
		}
	}
	var recentCount uint32
	prevCount := s.folderMsgCounts[fid]
	if len(msgs) > prevCount {
		recentCount = uint32(len(msgs) - prevCount)
	}
	s.folderMsgCounts[fid] = len(msgs)

	var firstUnseen uint32
	for seq, id := range msgs {
		email, err := s.deps.Mail.GetMessage(s.deps.Ctx, s.mailUserID, id)
		if err == nil && email.ReadStatus == constants.UnRead {
			firstUnseen = uint32(seq + 1)
			break
		}
	}

	return &SelectResult{
		Flags:             []string{`\Answered`, `\Flagged`, `\Deleted`, `\Seen`, `\Draft`},
		PermanentFlags:    []string{`\Answered`, `\Flagged`, `\Deleted`, `\Seen`, `\Draft`, `\*`},
		NumMessages:       uint32(len(msgs)),
		NumRecent:         recentCount,
		FirstUnseenSeqNum: firstUnseen,
		UIDNext:           maxUID + 1,
		UIDValidity:       s.uidValidity,
	}, nil
}

// ---- UNSELECT / CLOSE ----

func (s *MailSession) Unselect() error {
	s.folderID = 0
	s.folderIMAP = ""
	s.msgs = nil
	return nil
}

func (s *MailSession) Close() error {
	_, _ = s.Expunge()
	s.folderID = 0
	s.folderIMAP = ""
	s.msgs = nil
	return nil
}

// ---- STATUS ----

func (s *MailSession) Status(mbox string, items []string) (*StatusResult, error) {
	if err := s.ensureAuth(); err != nil {
		return nil, err
	}
	folders, err := s.loadFolders()
	if err != nil {
		return nil, err
	}
	fid, uv, found := findFolderID(folders, mbox)
	if !found {
		return nil, errors.New("mailbox does not exist")
	}
	msgs, err := loadAllMailIDs(s.deps.Ctx, s.deps.Mail, s.mailUserID, fid)
	if err != nil {
		return nil, err
	}
	statusUIDValidity := uv
	if statusUIDValidity == 0 {
		statusUIDValidity = uidValidityForFolder(fid)
	}
	res := &StatusResult{}
	for _, item := range items {
		switch item {
		case "MESSAGES":
			v := uint32(len(msgs))
			res.Messages = &v
		case "RECENT":
			prevCount := s.folderMsgCounts[fid]
			v := uint32(0)
			if len(msgs) > prevCount {
				v = uint32(len(msgs) - prevCount)
			}
			res.Recent = &v
		case "UIDNEXT":
			var maxUID uint32
			for _, id := range msgs {
				u := uint32FromMailID(id)
				if u > maxUID {
					maxUID = u
				}
			}
			v := maxUID + 1
			res.UIDNext = &v
		case "UIDVALIDITY":
			v := statusUIDValidity
			res.UIDValidity = &v
		case "UNSEEN":
			var unseen uint32
			for _, id := range msgs {
				email, err := s.deps.Mail.GetMessage(s.deps.Ctx, s.mailUserID, id)
				if err == nil && email.ReadStatus == constants.UnRead {
					unseen++
				}
			}
			res.Unseen = &unseen
		}
	}
	return res, nil
}

// ---- SEARCH ----

// Search returns the matching message sequence numbers. When uidCmd is true
// (i.e. the client issued UID SEARCH), it returns UIDs instead, as required by
// RFC 3501: a UID SEARCH response MUST contain UIDs, not sequence numbers.
func (s *MailSession) Search(criteria string, uidCmd bool) ([]uint32, error) {
	if err := s.ensureAuth(); err != nil {
		return nil, err
	}
	if s.folderID == 0 {
		return nil, errors.New("no mailbox selected")
	}
	criteria = strings.TrimSpace(strings.ToLower(criteria))
	var seqs []uint32
	switch {
	case criteria == "all" || criteria == "":
		for i := range s.msgs {
			seqs = append(seqs, uint32(i+1))
		}
	case criteria == "unseen":
		for i, id := range s.msgs {
			email, err := s.deps.Mail.GetMessage(s.deps.Ctx, s.mailUserID, id)
			if err == nil && email.ReadStatus == constants.UnRead {
				seqs = append(seqs, uint32(i+1))
			}
		}
	case criteria == "seen":
		for i, id := range s.msgs {
			email, err := s.deps.Mail.GetMessage(s.deps.Ctx, s.mailUserID, id)
			if err == nil && email.ReadStatus != constants.UnRead {
				seqs = append(seqs, uint32(i+1))
			}
		}
	default:
		ss, err := parseSeqSet(criteria)
		if err == nil {
			ss = normalizeSeqSet(ss, uint32(len(s.msgs)))
			for _, r := range ss {
				for seq := r.Start; seq <= r.Stop && int(seq) <= len(s.msgs); seq++ {
					seqs = append(seqs, seq)
				}
			}
		}
	}
	if uidCmd {
		uids := make([]uint32, 0, len(seqs))
		for _, seq := range seqs {
			if int(seq) >= 1 && int(seq) <= len(s.msgs) {
				uids = append(uids, uint32FromMailID(s.msgs[seq-1]))
			}
		}
		return uids, nil
	}
	return seqs, nil
}

// ---- FETCH ----

func (s *MailSession) Fetch(kind NumKind, seqSet SeqSet, uidSet UIDSet, opts FetchOptions, fn func(seq uint32, mailID int64) error) error {
	if err := s.ensureAuth(); err != nil {
		return err
	}
	if kind == NumKindSeq {
		seqSet = normalizeSeqSet(seqSet, uint32(len(s.msgs)))
		for i, mailID := range s.msgs {
			seq := uint32(i + 1)
			if !SeqSetContains(seqSet, seq) {
				continue
			}
			if err := fn(seq, mailID); err != nil {
				return err
			}
		}
	} else {
		uidSet = normalizeUIDSet(uidSet, uint32FromMailID(s.msgs[len(s.msgs)-1]))
		for i, mailID := range s.msgs {
			uid := uint32FromMailID(mailID)
			seq := uint32(i + 1)
			if !UIDSetContains(uidSet, uid) {
				continue
			}
			if err := fn(seq, mailID); err != nil {
				return err
			}
		}
	}
	return nil
}

type FetchMessageData struct {
	Seq          uint32
	UID          uint32
	Flags        []string
	EnvelopeWire string
	RFC822Size   int64
	InternalDate time.Time
	OpenRaw      func() (io.ReadCloser, int64, error)
}

func (s *MailSession) LoadFetchMessageData(seq uint32, mailID int64, opts FetchOptions) (*FetchMessageData, error) {
	email, err := s.deps.Mail.GetMessage(s.deps.Ctx, s.mailUserID, mailID)
	if err != nil {
		return nil, err
	}
	fd := &FetchMessageData{
		Seq:   seq,
		UID:   uint32FromMailID(mailID),
		Flags: messageFlagStrings(email),
	}
	if opts.Envelope {
		fd.EnvelopeWire = formatEnvelope(email)
	}
	if opts.RFC822Size {
		fd.RFC822Size = email.MailSize
	}
	if opts.InternalDate {
		t, _ := time.Parse(time.RFC3339, email.MailTime)
		if t.IsZero() {
			t = time.Now()
		}
		fd.InternalDate = t
	}
	if opts.BodyPeek || opts.BodyNotPeek {
		fd.OpenRaw = func() (io.ReadCloser, int64, error) {
			rc, sz, err := s.deps.Mail.OpenMessageRaw(s.deps.Ctx, s.mailUserID, mailID)
			return rc, sz, err
		}
	}
	return fd, nil
}

// ---- STORE ----

func (s *MailSession) Store(kind NumKind, seqSet SeqSet, uidSet UIDSet, sf *StoreFlags) error {
	if err := s.ensureAuth(); err != nil {
		return err
	}

	// Determine which standard flags are requested
	seenWanted := containsFlag(sf.Flags, `\Seen`)
	flaggedWanted := containsFlag(sf.Flags, `\Flagged`)
	deletedWanted := containsFlag(sf.Flags, `\Deleted`)

	affect := func(mailID int64) error {
		switch sf.Op {
		case StoreOpAdd:
			if seenWanted {
				if err := s.deps.Mail.MarkRead(s.deps.Ctx, s.mailUserID, mailID, constants.WebRead); err != nil {
					return err
				}
			}
			if flaggedWanted {
				if err := s.deps.Mail.SetMessageFlagged(s.deps.Ctx, s.mailUserID, mailID, true); err != nil {
					return err
				}
			}
			if deletedWanted {
				if err := s.deps.Mail.SetMessageDeleted(s.deps.Ctx, s.mailUserID, mailID, true); err != nil {
					return err
				}
			}

		case StoreOpRemove:
			if seenWanted {
				if err := s.deps.Mail.MarkRead(s.deps.Ctx, s.mailUserID, mailID, constants.UnRead); err != nil {
					return err
				}
			}
			if flaggedWanted {
				if err := s.deps.Mail.SetMessageFlagged(s.deps.Ctx, s.mailUserID, mailID, false); err != nil {
					return err
				}
			}
			if deletedWanted {
				if err := s.deps.Mail.SetMessageDeleted(s.deps.Ctx, s.mailUserID, mailID, false); err != nil {
					return err
				}
			}

		case StoreOpReplace:
			// Replace: set specified flags, clear others
			rs := constants.WebRead
			if !seenWanted {
				rs = constants.UnRead
			}
			if err := s.deps.Mail.MarkRead(s.deps.Ctx, s.mailUserID, mailID, rs); err != nil {
				return err
			}
			if err := s.deps.Mail.SetMessageFlagged(s.deps.Ctx, s.mailUserID, mailID, flaggedWanted); err != nil {
				return err
			}
			if err := s.deps.Mail.SetMessageDeleted(s.deps.Ctx, s.mailUserID, mailID, deletedWanted); err != nil {
				return err
			}
		}
		return nil
	}

	if kind == NumKindSeq {
		seqSet = normalizeSeqSet(seqSet, uint32(len(s.msgs)))
		for i, mailID := range s.msgs {
			seq := uint32(i + 1)
			if !SeqSetContains(seqSet, seq) {
				continue
			}
			if err := affect(mailID); err != nil {
				return err
			}
		}
	} else {
		uidSet = normalizeUIDSet(uidSet, uint32FromMailID(s.msgs[len(s.msgs)-1]))
		for _, mailID := range s.msgs {
			uid := uint32FromMailID(mailID)
			if !UIDSetContains(uidSet, uid) {
				continue
			}
			if err := affect(mailID); err != nil {
				return err
			}
		}
	}
	return nil
}

func containsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}

// ---- COPY ----

func (s *MailSession) CopyUID(uidSet UIDSet, dest string) (uint32, error) {
	destID, err := s.resolveFolderName(dest)
	if err != nil {
		return 0, err
	}
	var count uint32
	err = EachUID(uidSet, func(uid uint32) error {
		if merr := s.deps.Mail.MoveMessage(s.deps.Ctx, s.mailUserID, int64(uid), destID); merr != nil {
			return merr
		}
		count++
		return nil
	})
	return count, err
}

func (s *MailSession) CopySeq(seqSet SeqSet, dest string) (uint32, error) {
	destID, err := s.resolveFolderName(dest)
	if err != nil {
		return 0, err
	}
	var count uint32
	seqSet = normalizeSeqSet(seqSet, uint32(len(s.msgs)))
	for i, mailID := range s.msgs {
		seq := uint32(i + 1)
		if !SeqSetContains(seqSet, seq) {
			continue
		}
		if merr := s.deps.Mail.MoveMessage(s.deps.Ctx, s.mailUserID, mailID, destID); merr != nil {
			return 0, merr
		}
		count++
	}
	return count, err
}

// ---- MOVE ----

func (s *MailSession) MoveUID(uidSet UIDSet, dest string) ([]uint32, uint32, error) {
	destID, err := s.resolveFolderName(dest)
	if err != nil {
		return nil, 0, err
	}
	oldMsgs := append([]int64(nil), s.msgs...)
	var kept []int64
	var expunged []uint32
	for _, id := range oldMsgs {
		uid := uint32FromMailID(id)
		moved := false
		_ = EachUID(uidSet, func(u uint32) error {
			if uid == u {
				moved = true
			}
			return nil
		})
		if moved {
			_ = s.deps.Mail.MoveMessage(s.deps.Ctx, s.mailUserID, id, destID)
			_ = s.deps.Mail.PurgeMessage(s.deps.Ctx, s.mailUserID, id)
		} else {
			kept = append(kept, id)
		}
	}
	// Compute expunged sequences
	for i, id := range oldMsgs {
		uid := uint32FromMailID(id)
		moved := false
		_ = EachUID(uidSet, func(u uint32) error {
			if uid == u {
				moved = true
			}
			return nil
		})
		if moved {
			expunged = append(expunged, uint32(i+1-len(expunged)))
		}
	}
	s.msgs = kept
	s.folderMsgCounts[s.folderID] = len(kept)
	s.cachedFolders = nil
	return expunged, 0, nil
}

func (s *MailSession) MoveSeq(seqSet SeqSet, dest string) ([]uint32, error) {
	destID, err := s.resolveFolderName(dest)
	if err != nil {
		return nil, err
	}
	oldMsgs := append([]int64(nil), s.msgs...)
	seqSet = normalizeSeqSet(seqSet, uint32(len(s.msgs)))
	var kept []int64
	var expunged []uint32
	for i, mailID := range oldMsgs {
		seq := uint32(i + 1)
		if !SeqSetContains(seqSet, seq) {
			kept = append(kept, mailID)
			continue
		}
		if merr := s.deps.Mail.MoveMessage(s.deps.Ctx, s.mailUserID, mailID, destID); merr != nil {
			return nil, merr
		}
		_ = s.deps.Mail.PurgeMessage(s.deps.Ctx, s.mailUserID, mailID)
		expunged = append(expunged, seq-uint32(len(expunged)))
	}
	s.msgs = kept
	s.folderMsgCounts[s.folderID] = len(kept)
	s.cachedFolders = nil
	return expunged, nil
}

// ---- EXPUNGE ----

func (s *MailSession) Expunge() ([]uint32, error) {
	if err := s.ensureAuth(); err != nil {
		return nil, err
	}
	var kept []int64
	var expunged []uint32
	for i, mailID := range s.msgs {
		email, err := s.deps.Mail.GetMessage(s.deps.Ctx, s.mailUserID, mailID)
		if err == nil && email.IsDeleted {
			_ = s.deps.Mail.PurgeMessage(s.deps.Ctx, s.mailUserID, mailID)
			expunged = append(expunged, uint32(i+1-len(expunged)))
		} else {
			kept = append(kept, mailID)
		}
	}
	s.msgs = kept
	s.folderMsgCounts[s.folderID] = len(kept)
	return expunged, nil
}

func (s *MailSession) ExpungeUID(uidSet UIDSet) ([]uint32, error) {
	if err := s.ensureAuth(); err != nil {
		return nil, err
	}
	var kept []int64
	var expunged []uint32
	for i, mailID := range s.msgs {
		uid := uint32FromMailID(mailID)
		if UIDSetContains(uidSet, uid) {
			_ = s.deps.Mail.PurgeMessage(s.deps.Ctx, s.mailUserID, mailID)
			expunged = append(expunged, uint32(i+1-len(expunged)))
		} else {
			kept = append(kept, mailID)
		}
	}
	s.msgs = kept
	s.folderMsgCounts[s.folderID] = len(kept)
	return expunged, nil
}

// ---- APPEND ----

func (s *MailSession) Append(mbox string, flags []string, dateStr string, body []byte) (uint32, error) {
	if err := s.ensureAuth(); err != nil {
		return 0, err
	}
	destID, err := s.resolveFolderName(mbox)
	if err != nil {
		return 0, err
	}
	sender, subject := extractAppendHeaders(body)
	now := time.Now()
	if dateStr != "" {
		if t, perr := time.Parse("02-Jan-2006 15:04:05 -0700", dateStr); perr == nil {
			now = t
		}
	}
	hasSeen := false
	for _, f := range flags {
		if strings.EqualFold(f, `\Seen`) {
			hasSeen = true
		}
	}
	rs := constants.UnRead
	if hasSeen {
		rs = constants.WebRead
	}
	// Save via the MailBackend by constructing an email and saving
	ema := &messaging.Email{
		MailUserID: s.mailUserID,
		FolderID:   destID,
		Sender:     sender,
		Recipient:  "",
		Subject:    subject,
		Body:       string(body),
		MailSize:   int64(len(body)),
		MailTime:   now.Format(time.RFC3339),
		ReadStatus: rs,
	}
	// Use a type-assertion to access Save or use an intermediate approach
	if saver, ok := s.deps.Mail.(interface {
		SaveMessage(ctx context.Context, email *messaging.Email) (int64, error)
	}); ok {
		id, serr := saver.SaveMessage(s.deps.Ctx, ema)
		if serr != nil {
			return 0, serr
		}
		s.cachedFolders = nil
		return uint32FromMailID(id), nil
	}
	return 0, errors.New("APPEND not supported by backend")
}

func extractAppendHeaders(body []byte) (sender, subject string) {
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		t := strings.TrimRight(line, "\r")
		if t == "" {
			break
		}
		low := strings.ToLower(t)
		if strings.HasPrefix(low, "from:") {
			sender = strings.TrimSpace(t[5:])
		} else if strings.HasPrefix(low, "subject:") {
			subject = strings.TrimSpace(t[8:])
		}
	}
	// Decode MIME-encoded subject so DB stores plain text (searchable by INSTR/LIKE)
	if strings.Contains(subject, "=?") {
		dec := mime.WordDecoder{}
		if d, err := dec.DecodeHeader(subject); err == nil {
			subject = d
		}
	}
	if sender == "" {
		sender = "unknown@sender.local"
	}
	if subject == "" {
		subject = "(no subject)"
	}
	return
}

// ---- CREATE / DELETE / RENAME ----

func (s *MailSession) CreateMailbox(mbox string) error {
	if err := s.ensureAuth(); err != nil {
		return err
	}
	name := strings.TrimSpace(mbox)
	if name == "" || IsInboxName(name) {
		return errors.New("cannot create INBOX")
	}
	_, err := s.deps.Mail.CreateFolder(s.deps.Ctx, s.mailUserID, name)
	s.cachedFolders = nil
	return err
}

func (s *MailSession) DeleteMailbox(mbox string) error {
	if err := s.ensureAuth(); err != nil {
		return err
	}
	name := strings.TrimSpace(mbox)
	if IsInboxName(name) {
		return errors.New("cannot delete INBOX")
	}
	// Delete ALL folders matching this name (handles legacy duplicates)
	folders, err := s.loadFolders()
	if err != nil {
		return err
	}
	var lastErr error
	found := false
	for _, f := range folders {
		if strings.EqualFold(folderDisplayName(f), name) {
			found = true
			if err := s.deps.Mail.DeleteFolder(s.deps.Ctx, s.mailUserID, f.ID); err != nil {
				lastErr = err
				continue
			}
		}
	}
	if !found {
		return errors.New("mailbox does not exist")
	}
	s.cachedFolders = nil
	return lastErr
}

func (s *MailSession) RenameMailbox(oldName, newName string) error {
	if err := s.ensureAuth(); err != nil {
		return err
	}
	if IsInboxName(oldName) || IsInboxName(newName) {
		return errors.New("cannot rename INBOX")
	}
	fid, err := s.resolveFolderName(oldName)
	if err != nil {
		return err
	}
	err = s.deps.Mail.RenameFolder(s.deps.Ctx, s.mailUserID, fid, newName)
	s.cachedFolders = nil
	return err
}

// ---- SUBSCRIBE / UNSUBSCRIBE ----

func (s *MailSession) Subscribe(mbox string) error {
	if err := s.ensureAuth(); err != nil {
		return err
	}
	return nil
}

func (s *MailSession) Unsubscribe(mbox string) error {
	if err := s.ensureAuth(); err != nil {
		return err
	}
	return nil
}

// ---- IDLE ----

func (s *MailSession) PollNewMessages() ([]PollResult, error) {
	if err := s.ensureAuth(); err != nil {
		return nil, err
	}
	if s.folderID == 0 {
		return nil, nil
	}
	newMsgs, err := loadAllMailIDs(s.deps.Ctx, s.deps.Mail, s.mailUserID, s.folderID)
	if err != nil {
		return nil, err
	}
	if len(newMsgs) > len(s.msgs) {
		diff := len(newMsgs) - len(s.msgs)
		s.msgs = newMsgs
		s.folderMsgCounts[s.folderID] = len(newMsgs)
		return []PollResult{{Exists: uint32(len(s.msgs)), Recent: uint32(diff)}}, nil
	}
	return nil, nil
}

// ---- Helpers ----

func IsInboxName(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "INBOX")
}

func loadAllMailIDs(ctx context.Context, b mailservice.Backend, uid shared.GlobalID, folderID int64) ([]int64, error) {
	return b.ListAllMailIDs(ctx, uid, folderID)
}

func uidValidityForFolder(folderID int64) uint32 {
	return uint32(folderID) ^ 0xa5a5a5a5
}

func uint32FromMailID(id int64) uint32 {
	if id < 0 || id > 0xffffffff {
		return 0
	}
	return uint32(id)
}

func messageFlagStrings(e *messaging.Email) []string {
	var fl []string
	if e.ReadStatus != constants.UnRead {
		fl = append(fl, `\Seen`)
	}
	if e.Flagged {
		fl = append(fl, `\Flagged`)
	}
	if e.IsDeleted {
		fl = append(fl, `\Deleted`)
	}
	return fl
}

func folderDisplayName(f mailservice.FolderDTO) string {
	if f.Name != "" {
		return f.Name
	}
	if f.IMAPName != "" {
		return f.IMAPName
	}
	return mailservice.IMAPNameForKind(f.Kind)
}

func findFolderID(folders []mailservice.FolderDTO, spec string) (int64, uint32, bool) {
	name := strings.TrimSpace(spec)
	for i := range folders {
		dn := folderDisplayName(folders[i])
		if strings.EqualFold(name, dn) {
			return folders[i].ID, folders[i].UIDValidity, true
		}
		if strings.EqualFold(name, mailservice.IMAPNameForKind(folders[i].Kind)) {
			return folders[i].ID, folders[i].UIDValidity, true
		}
	}
	if k, ok := mailservice.KindFromIMAPName(name); ok {
		for i := range folders {
			if folders[i].Kind == k {
				return folders[i].ID, folders[i].UIDValidity, true
			}
		}
	}
	return 0, 0, false
}

func normalizeSeqSet(ss SeqSet, maxSeq uint32) SeqSet {
	out := make(SeqSet, len(ss))
	copy(out, ss)
	for i := range out {
		staticNumRange(&out[i].Start, &out[i].Stop, maxSeq)
	}
	return out
}

func normalizeUIDSet(us UIDSet, maxUID uint32) UIDSet {
	out := make(UIDSet, len(us))
	copy(out, us)
	for i := range out {
		staticNumRange(&out[i].Start, &out[i].Stop, maxUID)
	}
	return out
}

func staticNumRange(start, stop *uint32, max uint32) {
	dyn := false
	if *start == 0 {
		*start = max
		dyn = true
	}
	if *stop == 0 {
		*stop = max
		dyn = true
	}
	if dyn && *start > *stop {
		*start, *stop = *stop, *start
	}
}

func matchList(name string, delim rune, reference, pattern string) bool {
	delimStr := ""
	if delim != 0 {
		delimStr = string(delim)
	}
	if delimStr != "" && strings.HasPrefix(pattern, delimStr) {
		reference = ""
		pattern = strings.TrimPrefix(pattern, delimStr)
	}
	if reference != "" {
		if delimStr != "" && !strings.HasSuffix(reference, delimStr) {
			reference += delimStr
		}
		if !strings.HasPrefix(name, reference) {
			return false
		}
		name = strings.TrimPrefix(name, reference)
	}
	return matchListPattern(name, delimStr, pattern)
}

func matchListPattern(name, delim, pattern string) bool {
	i := strings.IndexAny(pattern, "*%")
	if i == -1 {
		return name == pattern
	}
	chunk, wildcard, rest := pattern[0:i], pattern[i], pattern[i+1:]
	if len(chunk) > 0 && !strings.HasPrefix(name, chunk) {
		return false
	}
	name = strings.TrimPrefix(name, chunk)
	var j int
	for j = 0; j < len(name); j++ {
		if wildcard == '%' && string(name[j]) == delim {
			break
		}
		if matchListPattern(name[j:], delim, rest) {
			return true
		}
	}
	return matchListPattern(name[j:], delim, rest)
}
