package storagepath

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
)

// JoinMailboxRoot returns the absolute path for a user's storage directory.
func JoinMailboxRoot(root, dataPath string) string {
	return filepath.Join(root, dataPath)
}

// UserDBPath returns the path to the per-user SQLite database file.
func UserDBPath(absPath string) string {
	return filepath.Join(absPath, "user.db")
}

// HashShardPath returns a two-level hash sharded path for storing a mail file.
// For example, for an email with ID 12345, it produces a path like "a1/b2/12345.eml".
// The first two hex characters of SHA256 are used for the first two directory levels.
func HashShardPath(mailID int64) string {
	h := sha256.Sum256(fmt.Appendf(nil, "%d", mailID))
	shard1 := fmt.Sprintf("%02x", h[0])
	shard2 := fmt.Sprintf("%02x", h[1])
	return filepath.Join(shard1, shard2, fmt.Sprintf("%d.eml", mailID))
}

// DataPathHash produces a deterministic two-level hash sharded directory path for a user email.
// Uses hex characters of SHA256 for each level (0-f), formatted as hex, e.g. "a3/f7".
func DataPathHash(email string) string {
	h := sha256.Sum256([]byte(email))
	shard1 := fmt.Sprintf("%02x", h[0])
	shard2 := fmt.Sprintf("%02x", h[1])
	return filepath.Join(shard1, shard2)
}

// CleanUsername sanitizes the username to prevent path traversal attacks.
// It removes or replaces dangerous characters and ensures only safe characters are used.
func CleanUsername(username string) string {
	// Trim whitespace
	username = strings.TrimSpace(username)
	
	// Convert to lowercase for consistency
	username = strings.ToLower(username)
	
	// Replace unsafe characters with underscore
	var cleaned strings.Builder
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			cleaned.WriteRune(r)
		} else {
			cleaned.WriteRune('_')
		}
	}
	
	username = cleaned.String()
	
	// Remove consecutive dots to prevent confusion
	for strings.Contains(username, "..") {
		username = strings.ReplaceAll(username, "..", ".")
	}
	
	// Remove leading/trailing dots and whitespace
	username = strings.Trim(username, ".")
	
	// Limit length to prevent filesystem issues (max 64 chars)
	if len(username) > 64 {
		username = username[:64]
	}
	
	return username
}

// MailUserDataPath returns the relative storage directory path for a mail user.
// Format: {domain_name}/{hash_1}/{hash_2}/{username}/ where hash_1 and hash_2 are hex hashes.
// Example: example.com/a3/f7/john_doe/
func MailUserDataPath(domainName, email string) string {
	// Extract username from email
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return filepath.Join(domainName, DataPathHash(email), CleanUsername(email))
	}
	username := parts[0]
	
	// Clean username to prevent path traversal
	cleanUsername := CleanUsername(username)
	
	// Build path: {domain}/{hash1}/{hash2}/{username}
	return filepath.Join(domainName, DataPathHash(email), cleanUsername)
}

// UserStorageDir returns the absolute storage directory path for a mail user.
func UserStorageDir(root, domainName, email string) string {
	return filepath.Join(root, MailUserDataPath(domainName, email))
}

// HashShardPathStr produces a hash sharded path for a string-based UID.
func HashShardPathStr(uid string) string {
	h := sha256.Sum256([]byte(uid))
	shard1 := fmt.Sprintf("%02x", h[0])
	shard2 := fmt.Sprintf("%02x", h[1])
	return filepath.Join(shard1, shard2, uid)
}

