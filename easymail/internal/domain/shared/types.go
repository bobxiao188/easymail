package shared

import "errors"

// EmailAddress is a value object for email addresses.
type EmailAddress struct {
	address string
}

func NewEmailAddress(addr string) EmailAddress {
	return EmailAddress{address: addr}
}

func (e EmailAddress) String() string {
	return e.address
}

func (e EmailAddress) IsValid() bool {
	return e.address != ""
}

// Sentinel errors used across bounded contexts.
// These should be migrated to each BC's own errors.go over time.
var (
	ErrAccountInactive    = errors.New("account is inactive or disabled")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrResourceNotFound   = errors.New("resource not found for the given ID")
)

// IsAny checks if any of the provided errors match a specific domain error type.
func IsAny(errs ...error) bool {
	for _, err := range errs {
		if err != nil && (errors.Is(err, ErrAccountInactive) || errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrResourceNotFound)) {
			return true
		}
	}
	return false
}

// LabelDTO is the read model for email labels, shared across layers.
type LabelDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	IsBuiltin  bool   `json:"is_builtin"`
	EmailCount int64  `json:"email_count,omitempty"`
}
