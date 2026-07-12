package dkim

import (
	"fmt"
	"strings"
	"time"
)

// DKIMHeader represents a DKIM-Signature header field.
type DKIMHeader struct {
	// Version (tag v)
	Version string

	// Algorithm (tag a) - "rsa-sha1" or "rsa-sha256"
	Algorithm string

	// Signature data base64 (tag b)
	SignatureData string

	// Body hash base64 (tag bh)
	BodyHash string

	// Message canonicalization (tag c) - e.g. "relaxed/simple"
	MessageCanonicalization string

	// Domain (tag d)
	Domain string

	// Signed header fields (tag h)
	Headers []string

	// Agent or User Identifier (tag i)
	Auid string

	// Body length (tag l)
	BodyLength uint

	// Query methods (tag q)
	QueryMethods []string

	// Selector (tag s)
	Selector string

	// Signature timestamp (tag t)
	SignatureTimestamp time.Time

	// Signature expiration (tag x)
	SignatureExpiration time.Time

	// Copied header fields (tag z)
	CopiedHeaderFields []string
}

// newDkimHeaderBySigOptions returns a new DKIMHeader initialized with SigOptions.
func newDkimHeaderBySigOptions(options SigOptions) *DKIMHeader {
	h := new(DKIMHeader)
	h.Version = "1"
	h.Algorithm = options.Algo
	h.MessageCanonicalization = options.Canonicalization
	h.Domain = options.Domain
	h.Headers = options.Headers
	h.Auid = options.Auid
	h.BodyLength = options.BodyLength
	h.QueryMethods = options.QueryMethods
	h.Selector = options.Selector
	if options.AddSignatureTimestamp {
		h.SignatureTimestamp = time.Now()
	}
	if options.SignatureExpireIn > 0 {
		h.SignatureExpiration = time.Now().Add(time.Duration(options.SignatureExpireIn) * time.Second)
	}
	h.CopiedHeaderFields = options.CopiedHeaderFields
	return h
}

// getHeaderBaseForSigning returns the base DKIM-Signature header string for signing.
func (d *DKIMHeader) getHeaderBaseForSigning(bodyHash string) string {
	h := "DKIM-Signature: v=" + d.Version + "; a=" + d.Algorithm + "; q=" + strings.Join(d.QueryMethods, ":") + "; c=" + d.MessageCanonicalization + ";" + CRLF + TAB
	subh := "s=" + d.Selector + ";"
	if len(subh)+len(d.Domain)+4 > MaxHeaderLineLength {
		h += subh + FWS
		subh = ""
	}
	subh += " d=" + d.Domain + ";"

	// Auid
	if len(d.Auid) != 0 {
		if len(subh)+len(d.Auid)+4 > MaxHeaderLineLength {
			h += subh + FWS
			subh = ""
		}
		subh += " i=" + d.Auid + ";"
	}

	// Signature timestamp
	if !d.SignatureTimestamp.IsZero() {
		ts := d.SignatureTimestamp.Unix()
		if len(subh)+14 > MaxHeaderLineLength {
			h += subh + FWS
			subh = ""
		}
		subh += " t=" + fmt.Sprintf("%d", ts) + ";"
	}
	if len(subh)+len(d.Domain)+4 > MaxHeaderLineLength {
		h += subh + FWS
		subh = ""
	}

	// Expiration
	if !d.SignatureExpiration.IsZero() {
		ts := d.SignatureExpiration.Unix()
		if len(subh)+14 > MaxHeaderLineLength {
			h += subh + FWS
			subh = ""
		}
		subh += " x=" + fmt.Sprintf("%d", ts) + ";"
	}

	// Body length
	if d.BodyLength != 0 {
		bodyLengthStr := fmt.Sprintf("%d", d.BodyLength)
		if len(subh)+len(bodyLengthStr)+4 > MaxHeaderLineLength {
			h += subh + FWS
			subh = ""
		}
		subh += " l=" + bodyLengthStr + ";"
	}

	// Headers
	if len(subh)+len(d.Headers)+4 > MaxHeaderLineLength {
		h += subh + FWS
		subh = ""
	}
	subh += " h="
	for _, header := range d.Headers {
		if len(subh)+len(header)+1 > MaxHeaderLineLength {
			h += subh + FWS
			subh = ""
		}
		subh += header + ":"
	}
	subh = subh[:len(subh)-1] + ";"

	// Body hash
	if len(subh)+5+len(bodyHash) > MaxHeaderLineLength {
		h += subh + FWS
		subh = ""
	} else {
		subh += " "
	}
	subh += "bh="
	l := len(subh)
	for _, c := range bodyHash {
		subh += string(c)
		l++
		if l >= MaxHeaderLineLength {
			h += subh + FWS
			subh = ""
			l = 0
		}
	}
	h += subh + ";" + FWS + "b="
	return h
}
