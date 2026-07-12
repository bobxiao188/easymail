package imap

import (
	"testing"
)

// --- tokenize ---

func TestTokenize_Simple(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		{"", nil},
		{"CAPABILITY", []string{"CAPABILITY"}},
		{"A001 LOGIN alice password", []string{"A001", "LOGIN", "alice", "password"}},
		{"A002 SELECT INBOX", []string{"A002", "SELECT", "INBOX"}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := tokenize(tt.line)
			if err != nil {
				t.Fatalf("tokenize(%q) error = %v", tt.line, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("tokenize(%q) = %v, want %v", tt.line, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("token[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestTokenize_Quoted(t *testing.T) {
	got, err := tokenize(`A001 LOGIN "hello world" x`)
	if err != nil {
		t.Fatalf("tokenize() error = %v", err)
	}
	if len(got) != 4 || got[2] != "hello world" {
		t.Errorf("tokenize() = %v, want ... 'hello world' ...", got)
	}
}

func TestTokenize_ParenList(t *testing.T) {
	got, err := tokenize(`A001 FETCH 1:* (FLAGS INTERNALDATE)`)
	if err != nil {
		t.Fatalf("tokenize() error = %v", err)
	}
	if len(got) != 4 || got[3] != "(FLAGS INTERNALDATE)" {
		t.Errorf("tokenize() = %v", got)
	}
}

func TestTokenize_NestedParens(t *testing.T) {
	got, err := tokenize(`A001 SEARCH HEADER Subject (hello world)`)
	if err != nil {
		t.Fatalf("tokenize() error = %v", err)
	}
	if len(got) < 5 {
		t.Errorf("tokenize() = %v, want at least 5 tokens", got)
	}
}

func TestTokenize_UnclosedQuote(t *testing.T) {
	_, err := tokenize(`A001 LOGIN "hello`)
	if err == nil {
		t.Fatal("tokenize() with unclosed quote should error")
	}
}

func TestTokenize_UnbalancedParen(t *testing.T) {
	_, err := tokenize(`A001 FETCH 1:* (FLAGS`)
	if err == nil {
		t.Fatal("tokenize() with unbalanced paren should error")
	}
}

// --- parseSeqSet ---

func TestParseSeqSet(t *testing.T) {
	tests := []struct {
		s    string
		want []SeqRange
	}{
		{"1", []SeqRange{{1, 1}}},
		{"1,3,5", []SeqRange{{1, 1}, {3, 3}, {5, 5}}},
		{"1:5", []SeqRange{{1, 5}}},
		{"1:3,5:7", []SeqRange{{1, 3}, {5, 7}}},
		
		{"1:*", []SeqRange{{1, 0}}},
		{"*:100", []SeqRange{{0, 100}}},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got, err := parseSeqSet(tt.s)
			if err != nil {
				t.Fatalf("parseSeqSet(%q) error = %v", tt.s, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseSeqSet(%q) = %v, want %v", tt.s, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("range[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseSeqSet_Empty(t *testing.T) {
	_, err := parseSeqSet("")
	if err == nil {
		t.Fatal("parseSeqSet() should error")
	}
}

func TestParseSeqSet_Invalid(t *testing.T) {
	_, err := parseSeqSet("abc")
	if err == nil {
		t.Fatal("parseSeqSet(abc) should error")
	}
}

// --- parseUIDSet ---

func TestParseUIDSet(t *testing.T) {
	got, err := parseUIDSet("10:20")
	if err != nil {
		t.Fatalf("parseUIDSet() error = %v", err)
	}
	if len(got) != 1 || got[0].Start != 10 || got[0].Stop != 20 {
		t.Errorf("parseUIDSet() = %v, want [(10,20)]", got)
	}
}

// --- parseRangeToken ---

func TestParseRangeToken(t *testing.T) {
	tests := []struct {
		tok    string
		start  uint32
		stop   uint32
		errNil bool
	}{
		{"42", 42, 42, true},
		{"1:5", 1, 5, true},
		
		{"1:*", 1, 0, true},
		{"*:100", 0, 100, true},
		{"abc", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.tok, func(t *testing.T) {
			s, e, err := parseRangeToken(tt.tok)
			if tt.errNil && err != nil {
				t.Fatalf("parseRangeToken(%q) error = %v", tt.tok, err)
			}
			if !tt.errNil && err == nil {
				t.Fatalf("parseRangeToken(%q) should error", tt.tok)
			}
			if s != tt.start || e != tt.stop {
				t.Errorf("parseRangeToken(%q) = (%d,%d), want (%d,%d)", tt.tok, s, e, tt.start, tt.stop)
			}
		})
	}
}

// --- parseFetchList ---

func TestParseFetchList(t *testing.T) {
	opts, err := parseFetchList("(FLAGS INTERNALDATE RFC822.SIZE)")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if !opts.Flags || !opts.InternalDate || !opts.RFC822Size {
		t.Errorf("parseFetchList() = %+v, want all true", opts)
	}
}

func TestParseFetchList_UID(t *testing.T) {
	opts, err := parseFetchList("(UID FLAGS)")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if !opts.UID || !opts.Flags {
		t.Errorf("parseFetchList() = %+v, want UID+FLAGS", opts)
	}
}

func TestParseFetchList_BodyPeek(t *testing.T) {
	opts, err := parseFetchList("(BODY.PEEK[])")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if !opts.BodyPeek {
		t.Errorf("parseFetchList() BodyPeek = %v, want true", opts.BodyPeek)
	}
}

func TestParseFetchList_Fast(t *testing.T) {
	opts, err := parseFetchList("(FAST)")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if !opts.Flags || !opts.InternalDate || !opts.RFC822Size {
		t.Errorf("FAST should set Flags+InternalDate+RFC822Size: %+v", opts)
	}
}

func TestParseFetchList_All(t *testing.T) {
	opts, err := parseFetchList("(ALL)")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if !opts.Flags || !opts.InternalDate || !opts.RFC822Size || !opts.Envelope {
		t.Errorf("ALL should set Flags+InternalDate+RFC822Size+Envelope: %+v", opts)
	}
}

func TestParseFetchList_Full(t *testing.T) {
	opts, err := parseFetchList("(FULL)")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if !opts.Flags || !opts.InternalDate || !opts.RFC822Size || !opts.Envelope {
		t.Errorf("FULL should set Flags+InternalDate+RFC822Size+Envelope: %+v", opts)
	}
}

func TestParseFetchList_NoParen(t *testing.T) {
	_, err := parseFetchList("FLAGS")
	if err == nil {
		t.Fatal("parseFetchList without paren should error")
	}
}

func TestParseFetchList_Empty(t *testing.T) {
	opts, err := parseFetchList("()")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if opts.UID || opts.Flags {
		t.Errorf("empty fetch list should have no opts: %+v", opts)
	}
}

// --- parseStoreItem ---

func TestParseStoreItem(t *testing.T) {
	tests := []struct {
		item   string
		op     StoreOp
		silent bool
	}{
		{"+FLAGS", StoreOpAdd, false},
		{"-FLAGS", StoreOpRemove, false},
		{"FLAGS", StoreOpReplace, false},
		{"+FLAGS.SILENT", StoreOpAdd, true},
		{"-FLAGS.SILENT", StoreOpRemove, true},
		{"FLAGS.SILENT", StoreOpReplace, true},
		{"+flags", StoreOpAdd, false},
		{"unknown", StoreOpReplace, false},
	}
	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			op, silent := parseStoreItem(tt.item)
			if op != tt.op || silent != tt.silent {
				t.Errorf("parseStoreItem(%q) = (%v,%v), want (%v,%v)", tt.item, op, silent, tt.op, tt.silent)
			}
		})
	}
}

// --- parseFlagList ---

func TestParseFlagList(t *testing.T) {
	tests := []struct {
		s    string
		want []string
	}{
		{`(\Seen)`, []string{`\Seen`}},
		{`(\Seen \Answered)`, []string{`\Seen`, `\Answered`}},
		{`()`, nil},
		{`not_parens`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := parseFlagList(tt.s)
			if len(got) != len(tt.want) {
				t.Fatalf("parseFlagList(%q) = %v, want %v", tt.s, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("flag[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// --- parseFetchItems ---

func TestParseFetchItems(t *testing.T) {
	got := parseFetchItems("FLAGS INTERNALDATE BODY.PEEK[]")
	if len(got) != 3 {
		t.Fatalf("parseFetchItems() = %v, want 3 items", got)
	}
	if got[0] != "FLAGS" || got[1] != "INTERNALDATE" || got[2] != "BODY.PEEK[]" {
		t.Errorf("parseFetchItems() = %v", got)
	}
}

func TestParseFetchItems_HeaderFields(t *testing.T) {
	// BODY.PEEK[HEADER.FIELDS (From Subject)] must be a single token
	got := parseFetchItems("BODY.PEEK[HEADER.FIELDS (From Subject)] BODY[]")
	if len(got) != 2 {
		t.Fatalf("parseFetchItems() = %v, want 2 items (got %d)", got, len(got))
	}
	if got[0] != "BODY.PEEK[HEADER.FIELDS (From Subject)]" {
		t.Errorf("parseFetchItems()[0] = %q, want BODY.PEEK[HEADER.FIELDS (From Subject)]", got[0])
	}
	if got[1] != "BODY[]" {
		t.Errorf("parseFetchItems()[1] = %q, want BODY[]", got[1])
	}
}

func TestParseFetchItems_HeaderFieldsNot(t *testing.T) {
	// HEADER.FIELDS.NOT with multiple fields
	got := parseFetchItems("BODY.PEEK[HEADER.FIELDS.NOT (X-Bogosity X-Spam)]")
	if len(got) != 1 {
		t.Fatalf("parseFetchItems() = %v, want 1 item", got)
	}
	want := "BODY.PEEK[HEADER.FIELDS.NOT (X-Bogosity X-Spam)]"
	if got[0] != want {
		t.Errorf("parseFetchItems()[0] = %q, want %q", got[0], want)
	}
}

func TestParseFetchList_HeaderFields(t *testing.T) {
	// Full FETCH list with HEADER.FIELDS — BodyItem must be the complete item
	opts, err := parseFetchList("(UID RFC822.SIZE FLAGS BODY.PEEK[HEADER.FIELDS (From To Cc Subject Date Message-ID)] FLAGS)")
	if err != nil {
		t.Fatalf("parseFetchList() error = %v", err)
	}
	if !opts.UID {
		t.Errorf("UID should be true")
	}
	if !opts.Flags {
		t.Errorf("Flags should be true")
	}
	if !opts.RFC822Size {
		t.Errorf("RFC822Size should be true")
	}
	if !opts.BodyPeek {
		t.Errorf("BodyPeek should be true")
	}
	wantBody := "BODY.PEEK[HEADER.FIELDS (From To Cc Subject Date Message-ID)]"
	if opts.BodyItem != wantBody {
		t.Errorf("BodyItem = %q, want %q", opts.BodyItem, wantBody)
	}
}
