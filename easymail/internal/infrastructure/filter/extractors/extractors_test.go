package extractors

import (
	"bytes"
	"testing"
)

// --- dkim.go pure functions ---

func TestParseDKIMHeaderParams(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		vals map[string]string
	}{
		{"empty", "", map[string]string{}},
		{"single", "v=1; a=rsa-sha256", map[string]string{"v": "1", "a": "rsa-sha256"}},
		{"with spaces", " v = 1 ; a = rsa-sha256 ", map[string]string{"v": "1", "a": "rsa-sha256"}},
		{"selector+domain", "s=20230601; d=example.com; h=from:to", map[string]string{
			"s": "20230601", "d": "example.com", "h": "from:to",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDKIMHeaderParams(tt.raw)
			for k, v := range tt.vals {
				if got[k] != v {
					t.Errorf("param[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestCanonicalizeDKIMBody(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"hello\r\n", "hello\r\n"},
		{"hello\r\n\r\n\r\n", "hello\r\n"},
		{"hello", "hello\r\n"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := canonicalizeDKIMBody(bytes.NewReader([]byte(tt.body)))
			if string(got) != tt.want {
				t.Errorf("canonicalizeDKIMBody(%q) = %q, want %q", tt.body, string(got), tt.want)
			}
		})
	}
}

func TestExtractDKIMPublicKey(t *testing.T) {
	tests := []struct {
		records []string
		want    string
	}{
		{nil, ""},
		{[]string{""}, ""},
		{[]string{"v=DKIM1; p=abc123"}, "abc123"},
		{[]string{"v=DKIM1; k=rsa; p=xyz789"}, "xyz789"},
		{[]string{"v=DKIM1"}, ""},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := extractDKIMPublicKey(tt.records)
			if got != tt.want {
				t.Errorf("extractDKIMPublicKey(%v) = %q, want %q", tt.records, got, tt.want)
			}
		})
	}
}

// --- dmarc.go pure functions ---

func TestParseDMARCPolicy(t *testing.T) {
	tests := []struct {
		records []string
		wantP   string
	}{
		{[]string{"v=DMARC1; p=reject; rua=mailto:dmarc@ex.com"}, "reject"},
		{[]string{"v=DMARC1; p=quarantine"}, "quarantine"},
		{[]string{"v=DMARC1; p=none"}, "none"},
		{[]string{"v=dmarc1; p=reject; sp=quarantine"}, "reject"},
		{[]string{"not dmarc"}, ""},
		{nil, ""},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := parseDMARCPolicy(tt.records)
			if tt.wantP == "" {
				if got != nil {
					t.Errorf("parseDMARCPolicy(%v) = %v, want nil", tt.records, got)
				}
				return
			}
			if got == nil || got["p"] != tt.wantP {
				t.Errorf("parseDMARCPolicy(%v) p = %q, want %q", tt.records, got["p"], tt.wantP)
			}
		})
	}
}

func TestDomainsAlign(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"example.com", "example.com", true},
		{"mail.example.com", "example.com", true},
		{"b.example.com", "a.example.com", true},
		{"example.com", "mail.example.com", true},
		{"example.com", "other.com", false},
		{"", "example.com", false},
		{"example.com", "", false},
		{"a.b.example.com", "c.d.example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := domainsAlign(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("domainsAlign(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestRelaxedOrgDomain(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"example.com", "example.com"},
		{"mail.example.com", "example.com"},
		{"a.b.example.com", "example.com"},
		{"com", "com"},
		{"single", "single"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := relaxedOrgDomain(tt.in)
			if got != tt.want {
				t.Errorf("relaxedOrgDomain(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDmarcPolicyToCode(t *testing.T) {
	tests := []struct {
		p    string
		want float64
	}{
		{"none", 0}, {"quarantine", 1}, {"reject", 2},
		{"NONE", 0}, {"Quarantine", 1},
		{"unknown", 0},
	}
	for _, tt := range tests {
		t.Run(tt.p, func(t *testing.T) {
			got := dmarcPolicyToCode(tt.p)
			if got != tt.want {
				t.Errorf("dmarcPolicyToCode(%q) = %v, want %v", tt.p, got, tt.want)
			}
		})
	}
}

// --- dns.go helpers ---

func TestBoolToFloat64(t *testing.T) {
	if got := boolToFloat64(true); got != 1 {
		t.Errorf("boolToFloat64(true) = %v, want 1", got)
	}
	if got := boolToFloat64(false); got != 0 {
		t.Errorf("boolToFloat64(false) = %v, want 0", got)
	}
}

func TestExtractDomainFromAddr(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"user@example.com", "example.com"},
		{"user@sub.example.com", "sub.example.com"},
		{"noatsign", ""},
		{"", ""},
		{"@", ""},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := extractDomainFromAddr(tt.addr)
			if got != tt.want {
				t.Errorf("extractDomainFromAddr(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

// --- spf.go constants ---

func TestSPFCodeConstants(t *testing.T) {
	if spfCodeNone != 0 {
		t.Errorf("spfCodeNone = %v", spfCodeNone)
	}
	if spfCodePass != 2 {
		t.Errorf("spfCodePass = %v", spfCodePass)
	}
	if spfCodeFail != 4 {
		t.Errorf("spfCodeFail = %v", spfCodeFail)
	}
}

func TestSpfSkipped(t *testing.T) {
	batch := spfSkipped("test_reason")
	if batch["spf_result"] != spfCodeNone {
		t.Errorf("spfSkipped spf_result = %v, want %v", batch["spf_result"], spfCodeNone)
	}
	if batch["spf_skipped"] != 1 {
		t.Errorf("spfSkipped spf_skipped = %v, want 1", batch["spf_skipped"])
	}
}

// --- dmarc constants ---

func TestDmarcCodeConstants(t *testing.T) {
	if dmarcCodeNone != 0 {
		t.Errorf("dmarcCodeNone = %v", dmarcCodeNone)
	}
	if dmarcCodePass != 1 {
		t.Errorf("dmarcCodePass = %v", dmarcCodePass)
	}
	if dmarcCodeFail != 2 {
		t.Errorf("dmarcCodeFail = %v", dmarcCodeFail)
	}
}

func TestDmarcNone(t *testing.T) {
	batch := dmarcNone()
	if batch["dmarc_result"] != dmarcCodeNone {
		t.Errorf("dmarcNone result = %v, want %v", batch["dmarc_result"], dmarcCodeNone)
	}
	if batch["dmarc_has_policy"] != 0 {
		t.Errorf("dmarcNone has_policy = %v, want 0", batch["dmarc_has_policy"])
	}
}

// --- dkim constants ---

func TestDkimCodeConstants(t *testing.T) {
	if dkimCodeNone != 0 {
		t.Errorf("dkimCodeNone = %v", dkimCodeNone)
	}
	if dkimCodePass != 1 {
		t.Errorf("dkimCodePass = %v", dkimCodePass)
	}
	if dkimCodeFail != 2 {
		t.Errorf("dkimCodeFail = %v", dkimCodeFail)
	}
}
