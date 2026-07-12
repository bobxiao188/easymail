// pkg/constants.go
package domain

// Defines common, system-wide constants used across multiple bounded contexts.

type ReadStatus int

const (
	Unread   ReadStatus = iota // Default state for new messages
	Read                       // Marked as read
	Archived                   // Explicitly archived by user action
)

// IsRead checks if the status is 'Read'
func (rs ReadStatus) IsRead() bool {
	return rs == Read
}

// String implements the stringer interface for readability.
func (rs ReadStatus) String() string {
	switch rs {
	case Unread:
		return "UNREAD"
	case Read:
		return "READ"
	case Archived:
		return "ARCHIVED"
	default:
		return "UNKNOWN_STATUS"
	}
}
