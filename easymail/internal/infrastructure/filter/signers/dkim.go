// Package signers provides DKIM signing infrastructure.
package signers

import (
	"context"
	"sync"

	"easymail/internal/domain/management"
	"easymail/internal/domain/messaging/service"
	"easymail/pkg/dkim"
)

// dkimCache holds the cached DKIM configuration for a domain to avoid repeated DB lookups.
type dkimCache struct {
	selector   string
	privateKey []byte
}

// DKIMSignerImpl implements service.DKIMSigner.
// It looks up DKIM configuration from the MailDomainRepository and signs emails using pkg/dkim.
type DKIMSignerImpl struct {
	repo management.MailDomainRepository
	mu   sync.RWMutex
	// Cache domain name -> DKIM config. Cleared on explicit invalidation.
	cache map[string]*dkimCache
}

// NewDKIMSigner creates a new DKIMSignerImpl that uses the given repository
// to look up domain DKIM configurations.
func NewDKIMSigner(repo management.MailDomainRepository) *DKIMSignerImpl {
	return &DKIMSignerImpl{
		repo:  repo,
		cache: make(map[string]*dkimCache),
	}
}

// Sign signs the given email bytes using the DKIM key for the specified domain.
// The email is modified in place with the DKIM-Signature header prepended.
// If the domain does not have DKIM enabled, the email is returned unchanged (no error).
func (s *DKIMSignerImpl) Sign(ctx context.Context, email *[]byte, domain string) error {
	// Look up cached config first
	s.mu.RLock()
	cached, found := s.cache[domain]
	s.mu.RUnlock()

	if !found {
		// Load from repository
		d, err := s.repo.FindValidatedByName(ctx, domain)
		if err != nil {
			// Domain not found or invalid — skip signing silently
			return nil
		}
		if !d.HasDKIM() {
			// DKIM not enabled for this domain — cache negative result
			s.mu.Lock()
			s.cache[domain] = nil
			s.mu.Unlock()
			return nil
		}
		cached = &dkimCache{
			selector:   d.DKIMSelector,
			privateKey: []byte(d.DKIMPrivateKey),
		}
		s.mu.Lock()
		s.cache[domain] = cached
		s.mu.Unlock()
	}

	if cached == nil {
		// Cached negative result — DKIM not enabled
		return nil
	}

	// Sign the email
	options := dkim.NewSigOptions()
	options.Domain = domain
	options.Selector = cached.selector
	options.PrivateKey = cached.privateKey
	options.Canonicalization = "relaxed/relaxed"
	options.Algo = "rsa-sha256"
	options.Headers = []string{"from", "date", "subject", "to", "message-id"}

	return dkim.Sign(email, options)
}

// Invalidate clears the cached DKIM configuration for the given domain.
// Call this when a domain's DKIM settings are updated.
func (s *DKIMSignerImpl) Invalidate(domain string) {
	s.mu.Lock()
	delete(s.cache, domain)
	s.mu.Unlock()
}

// Compile-time check
var _ service.DKIMSigner = (*DKIMSignerImpl)(nil)
