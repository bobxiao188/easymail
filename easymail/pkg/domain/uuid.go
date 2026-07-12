// pkg/domain/uuid.go
package domain

import "github.com/google/uuid"

// UUID generates a new universally unique identifier.
func GenerateUUID() string {
	return uuid.New().String()
}

// IsValidUUID checks if the given string is a valid UUID format.
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
