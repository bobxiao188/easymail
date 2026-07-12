package imap

import "testing"

func TestQuoteIMAPString(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"", `""`},
		{"hello", `"hello"`},
		{"hello world", `"hello world"`},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := quoteIMAPString(tt.s)
			if got != tt.want {
				t.Errorf("quoteIMAPString(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestEscapeQuoted(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"hello", "hello"},
		{`has\slash`, `has\\slash`},
		{`has"quote`, `has"quote`},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := escapeQuoted(tt.s)
			if got != tt.want {
				t.Errorf("escapeQuoted(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestFormatFlagsList(t *testing.T) {
	tests := []struct {
		flags []string
		want  string
	}{
		{nil, "()"},
		{[]string{}, "()"},
		{[]string{`\Seen`}, `(\Seen)`},
		{[]string{`\Seen`, `\Answered`}, `(\Seen \Answered)`},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatFlagsList(tt.flags)
			if got != tt.want {
				t.Errorf("formatFlagsList(%v) = %q, want %q", tt.flags, got, tt.want)
			}
		})
	}
}

func TestUint32dec(t *testing.T) {
	if got := uint32dec(42); got != "42" {
		t.Errorf("uint32dec(42) = %q, want 42", got)
	}
	if got := uint32dec(0); got != "0" {
		t.Errorf("uint32dec(0) = %q, want 0", got)
	}
}

func TestParseOneAddr(t *testing.T) {
	tests := []struct {
		s       string
		mailbox string
		host    string
	}{
		{"user@example.com", "user", "example.com"},
		{"user", "user", "local"},
		{"", "", "local"},
		{"@domain", "@domain", "local"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			a := parseOneAddr(tt.s)
			if a.mailbox != tt.mailbox || a.host != tt.host {
				t.Errorf("parseOneAddr(%q) = (%q,%q), want (%q,%q)",
					tt.s, a.mailbox, a.host, tt.mailbox, tt.host)
			}
		})
	}
}

func TestEnvelopeAddrList(t *testing.T) {
	if got := envelopeAddrList(""); got != "NIL" {
		t.Errorf("envelopeAddrList() = %q, want NIL", got)
	}
	got := envelopeAddrList("user@example.com")
	if got == "" || got == "NIL" {
		t.Errorf("envelopeAddrList() = %q, should not be NIL", got)
	}
}

func TestSingleAddress(t *testing.T) {
	got := singleAddress("user@example.com")
	if got == "" {
		t.Error("singleAddress() returned empty")
	}
}
