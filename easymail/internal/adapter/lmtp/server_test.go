package lmtp

import (
	"path/filepath"
	"testing"
)

func TestCleanDataPath_Normal(t *testing.T) {
	tests := []string{
		"user@example.com",
		"domain/user",
		"a/b/c",
		"",
	}
	for _, dp := range tests {
		t.Run(dp, func(t *testing.T) {
			got := cleanDataPath(dp)
			want := filepath.Join("/", dp)
			if got != want {
				t.Errorf("cleanDataPath(%q) = %q, want %q", dp, got, want)
			}
		})
	}
}

func TestCleanDataPath_Traversal(t *testing.T) {
	tests := []struct {
		dp   string
		want string
	}{
		{"../../etc/passwd", filepath.Join("/", "etc", "passwd")},
		{"..\\..\\windows\\system32", filepath.Join("/", "windows", "system32")},
		{"./././user", filepath.Join("/", "user")},
		{"user/../../../root", filepath.Join("/", "root")},
	}
	for _, tt := range tests {
		t.Run(tt.dp, func(t *testing.T) {
			got := cleanDataPath(tt.dp)
			if got != tt.want {
				t.Errorf("cleanDataPath(%q) = %q, want %q", tt.dp, got, tt.want)
			}
		})
	}
}

func TestCleanDataPath_NoEscapingRoot(t *testing.T) {
	tests := []string{"..", "../..", "../../.."}
	for _, dp := range tests {
		t.Run(dp, func(t *testing.T) {
			got := cleanDataPath(dp)
			want := string(filepath.Separator)
			if got != want {
				t.Errorf("cleanDataPath(%q) = %q, want %q", dp, got, want)
			}
		})
	}
}

func TestCleanDataPath_PreventsEscape(t *testing.T) {
	// Verify that cleanDataPath never produces a path outside root
	root := filepath.Join("testdata", "root")
	attacks := []string{
		"..", "../..", "../../../etc/passwd",
		"./././user/..", "user\\..\\..",
	}
	for _, dp := range attacks {
		t.Run(dp, func(t *testing.T) {
			clean := cleanDataPath(dp)
			absPath := filepath.Join(root, clean)
			// absPath must be within root
			if !stringsHasPrefix(absPath, root+string(filepath.Separator)) && absPath != root {
				t.Errorf("PATH TRAVERSAL: %q + %q = %q", root, clean, absPath)
			}
		})
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
