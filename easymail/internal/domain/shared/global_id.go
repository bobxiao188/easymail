package shared

import "fmt"

// GlobalID is a globally unique identifier for aggregate roots.
// Implemented as a UUID v4 string, used as the database primary key.
type GlobalID string

// NewGlobalID creates a GlobalID with a new UUID v4.
func NewGlobalID() GlobalID {
	return GlobalID(GenerateUUID())
}

// ParseGlobalID creates a GlobalID from a UUID string, validating the format.
func ParseGlobalID(s string) (GlobalID, error) {
	if !IsValidUUID(s) {
		return "", fmt.Errorf("invalid GlobalID format: %s", s)
	}
	return GlobalID(s), nil
}

// String returns the underlying UUID string.
func (g GlobalID) String() string {
	return string(g)
}

// DomainID represents an ID scoped to a specific bounded context.
// Use for cross-context references when GlobalID is not available.
type DomainID struct {
	Context string
	Value   string
}

func NewDomainID(context string, value string) *DomainID {
	return &DomainID{Context: context, Value: value}
}

func (d *DomainID) String() string {
	return fmt.Sprintf("%s:%s", d.Context, d.Value)
}
