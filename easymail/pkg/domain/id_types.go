// pkg/domain/id_types.go
package domain

import (
	"fmt"
)

// GlobalID is a standardized wrapper for all globally unique identifiers within the system.
type GlobalID struct {
	UUID string
}

// NewGlobalID creates a new instance from a UUID string.
func NewGlobalID(uuidStr string) (*GlobalID, error) {
	if !IsValidUUID(uuidStr) {
		return nil, fmt.Errorf("invalid uuid format: %s", uuidStr)
	}
	return &GlobalID{UUID: uuidStr}, nil
}

// String returns the underlying UUID as a string.
func (g *GlobalID) String() string {
	return g.UUID
}

// DomainID represents an ID scoped to a specific business domain, useful for clarity in large systems.
type DomainID struct {
	Context string // e.g., "USER", "DOMAIN", "MAILBOX"
	Value   string
}

// NewDomainID creates a structured domain ID.
func NewDomainID(context string, value string) *DomainID {
	return &DomainID{Context: context, Value: value}
}

// String provides a readable representation of the DomainID.
func (d *DomainID) String() string {
	return fmt.Sprintf("%s:%s", d.Context, d.Value)
}
