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

// NumKind distinguishes sequence numbers from UIDs.
type NumKind int

const (
	NumKindSeq NumKind = iota
	NumKindUID
)

// SeqRange and UIDRange are inclusive IMAP sequence-set intervals.
type SeqRange struct {
	Start, Stop uint32
}

// SeqSet and UIDSet are unions of intervals.
type SeqSet  []SeqRange
type UIDSet  []SeqRange

// FetchOptions lists FETCH items requested by the client.
type FetchOptions struct {
	UID          bool
	Flags        bool
	Envelope     bool
	RFC822Size   bool
	InternalDate bool
	BodyPeek     bool   // BODY.PEEK[] or equivalent
	BodyNotPeek  bool   // BODY[] (non-peek)
	BodyItem     string // exact item name from client request
}

// StoreOp is the STORE operation kind.
type StoreOp int

const (
	StoreOpReplace StoreOp = iota
	StoreOpAdd
	StoreOpRemove
)

// StoreFlags carries STORE / UID STORE flag updates.
type StoreFlags struct {
	Op     StoreOp
	Silent bool
	Flags  []string // \Seen, \Deleted, \Flagged, \Answered, \Draft
}

// StatusResult holds STATUS command results.
type StatusResult struct {
	Messages    *uint32
	Recent      *uint32
	UIDNext     *uint32
	UIDValidity *uint32
	Unseen      *uint32
}

// MailboxEntry is returned by LIST/LSUB.
type MailboxEntry struct {
	Delim   rune
	Mailbox string
}

// PollResult holds new message notification data from IDLE polling.
type PollResult struct {
	Exists uint32
	Recent uint32
}

// --------------------------------------------------------------------------
// Methods on SeqSet/UIDSet for iterating affected mail IDs.
// --------------------------------------------------------------------------

// SeqSetContains checks if a given seq number is in the set.
func SeqSetContains(ss SeqSet, seq uint32) bool {
	for i := range ss {
		if seq >= ss[i].Start && seq <= ss[i].Stop {
			return true
		}
	}
	return false
}

// UIDSetContains checks if a given uid is in the set.
func UIDSetContains(us UIDSet, uid uint32) bool {
	for i := range us {
		if uid >= us[i].Start && uid <= us[i].Stop {
			return true
		}
	}
	return false
}

// EachUID iterates over all UIDs in the set, calling fn for each one.
func EachUID(us UIDSet, fn func(uid uint32) error) error {
	for _, r := range us {
		for uid := r.Start; uid <= r.Stop; uid++ {
			if err := fn(uid); err != nil {
				return err
			}
		}
	}
	return nil
}

// EachSeq iterates over all sequence numbers in the set, calling fn for each one.
func EachSeq(ss SeqSet, fn func(seq uint32) error) error {
	for _, r := range ss {
		for seq := r.Start; seq <= r.Stop; seq++ {
			if err := fn(seq); err != nil {
				return err
			}
		}
	}
	return nil
}

// EachUIDPair iterates over all UIDs in the set, calling fn with (uid, newUID).
// newUID is returned from fn to enable COPYUID response.
func EachUIDPair(us UIDSet, fn func(uid uint32) (uint32, error)) ([]uint32, error) {
	var result []uint32
	for _, r := range us {
		for uid := r.Start; uid <= r.Stop; uid++ {
			newUID, err := fn(uid)
			if err != nil {
				return nil, err
			}
			result = append(result, newUID)
		}
	}
	return result, nil
}

