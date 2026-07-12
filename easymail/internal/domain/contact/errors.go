// internal/domain/contact/errors.go - Common sentinel errors

package contact

import "errors"

var (
	ErrNotFound        = errors.New("contact: not found")
	ErrDuplicate       = errors.New("contact: duplicate")
	ErrInvalidEmail    = errors.New("contact: invalid email")
	ErrInvalidGroup    = errors.New("contact: invalid group")
	ErrInvalidArgument = errors.New("contact: invalid argument")
)
