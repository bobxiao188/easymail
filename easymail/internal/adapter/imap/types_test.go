package imap

import "testing"

func TestSeqSetContains(t *testing.T) {
	ss := SeqSet{{1, 5}, {10, 15}}
	tests := []struct {
		seq  uint32
		want bool
	}{
		{1, true}, {3, true}, {5, true},
		{6, false}, {9, false},
		{10, true}, {12, true}, {15, true},
		{16, false}, {0, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := SeqSetContains(ss, tt.seq)
			if got != tt.want {
				t.Errorf("SeqSetContains(%v, %d) = %v, want %v", ss, tt.seq, got, tt.want)
			}
		})
	}
}

func TestUIDSetContains(t *testing.T) {
	us := UIDSet{{100, 200}}
	if !UIDSetContains(us, 150) {
		t.Error("UIDSetContains({100,200}, 150) should be true")
	}
	if UIDSetContains(us, 99) {
		t.Error("UIDSetContains({100,200}, 99) should be false")
	}
}

func TestEachUID(t *testing.T) {
	us := UIDSet{{1, 3}, {10, 11}}
	var got []uint32
	EachUID(us, func(uid uint32) error {
		got = append(got, uid)
		return nil
	})
	want := []uint32{1, 2, 3, 10, 11}
	if len(got) != len(want) {
		t.Fatalf("EachUID() = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("uid[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestEachSeq(t *testing.T) {
	ss := SeqSet{{5, 7}}
	var got []uint32
	EachSeq(ss, func(seq uint32) error {
		got = append(got, seq)
		return nil
	})
	want := []uint32{5, 6, 7}
	if len(got) != len(want) {
		t.Fatalf("EachSeq() = %v, want %v", got, want)
	}
}

func TestNumKind(t *testing.T) {
	if NumKindSeq != 0 {
		t.Error("NumKindSeq should be 0")
	}
	if NumKindUID != 1 {
		t.Error("NumKindUID should be 1")
	}
}

func TestStoreOp(t *testing.T) {
	if StoreOpReplace != 0 {
		t.Error("StoreOpReplace should be 0")
	}
	if StoreOpAdd != 1 {
		t.Error("StoreOpAdd should be 1")
	}
	if StoreOpRemove != 2 {
		t.Error("StoreOpRemove should be 2")
	}
}

func TestStatusResult(t *testing.T) {
	m := uint32(42)
	r := uint32(1)
	sr := StatusResult{Messages: &m, Recent: &r}
	if sr.Messages == nil || *sr.Messages != 42 {
		t.Error("StatusResult Messages not set")
	}
	if sr.Recent == nil || *sr.Recent != 1 {
		t.Error("StatusResult Recent not set")
	}
}

func TestMailboxEntry(t *testing.T) {
	e := MailboxEntry{Delim: '/', Mailbox: "INBOX"}
	if e.Delim != '/' || e.Mailbox != "INBOX" {
		t.Errorf("MailboxEntry = %+v", e)
	}
}
