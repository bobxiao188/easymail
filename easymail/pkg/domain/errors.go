// pkg/domain/errors.go
package domain

import "errors"

var ErrAccountInactive = errors.New("account is inactive or disabled")
var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrResourceNotFound = errors.New("resource not found for the given ID")

// IsAny checks if any of the provided errors match a specific domain error type.
func IsAny(errs ...error) bool {
	for _, err := range errs {
		if err != nil && (errors.Is(err, ErrAccountInactive) || errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrResourceNotFound)) {
			return true
		}
	}
	return false
}
