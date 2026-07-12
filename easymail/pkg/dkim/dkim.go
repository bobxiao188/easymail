// Package dkim provides tools for signing email according to RFC 6376.
package dkim

import (
	"bytes"
	"container/list"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"hash"
	"regexp"
	"strings"
)

const (
	CRLF                = "\r\n"
	TAB                 = " "
	FWS                 = CRLF + TAB
	MaxHeaderLineLength = 70
)

// SigOptions represents signing options.
type SigOptions struct {
	// DKIM version (default 1)
	Version uint

	// Private key used for signing (required)
	PrivateKey []byte

	// Domain (required)
	Domain string

	// Selector (required)
	Selector string

	// The Agent or User IDentifier
	Auid string

	// Message canonicalization (plain-text; OPTIONAL, default is
	// "simple/simple").
	Canonicalization string

	// The algorithm used to generate the signature: "rsa-sha1" or "rsa-sha256"
	Algo string

	// Signed header fields
	Headers []string

	// Body length count (if set to 0 this tag is omitted in DKIM header)
	BodyLength uint

	// Query Methods used to retrieve the public key
	QueryMethods []string

	// Add a signature timestamp
	AddSignatureTimestamp bool

	// Time validity of the signature (0=never)
	SignatureExpireIn uint64

	// Copied header fields
	CopiedHeaderFields []string
}

// NewSigOptions returns new sigoption with some default values.
func NewSigOptions() SigOptions {
	return SigOptions{
		Version:               1,
		Canonicalization:      "simple/simple",
		Algo:                  "rsa-sha256",
		Headers:               []string{"from"},
		BodyLength:            0,
		QueryMethods:          []string{"dns/txt"},
		AddSignatureTimestamp: true,
		SignatureExpireIn:     0,
	}
}

// Sign signs an email according to RFC 6376.
// The email is modified in place with the DKIM-Signature header prepended.
func Sign(email *[]byte, options SigOptions) error {
	var privateKey *rsa.PrivateKey
	var err error

	// PrivateKey
	if len(options.PrivateKey) == 0 {
		return ErrSignPrivateKeyRequired
	}
	d, _ := pem.Decode(options.PrivateKey)
	if d == nil {
		return ErrCandNotParsePrivateKey
	}

	// try to parse it as PKCS1 otherwise try PKCS8
	if key, err := x509.ParsePKCS1PrivateKey(d.Bytes); err != nil {
		if key, err := x509.ParsePKCS8PrivateKey(d.Bytes); err != nil {
			return ErrCandNotParsePrivateKey
		} else {
			privateKey = key.(*rsa.PrivateKey)
		}
	} else {
		privateKey = key
	}

	// Domain required
	if options.Domain == "" {
		return ErrSignDomainRequired
	}

	// Selector required
	if options.Selector == "" {
		return ErrSignSelectorRequired
	}

	// Canonicalization
	options.Canonicalization, err = validateCanonicalization(strings.ToLower(options.Canonicalization))
	if err != nil {
		return err
	}

	// Algo
	options.Algo = strings.ToLower(options.Algo)
	if options.Algo != "rsa-sha1" && options.Algo != "rsa-sha256" {
		return ErrSignBadAlgo
	}

	// Header must contain "from"
	hasFrom := false
	for i, h := range options.Headers {
		h = strings.ToLower(h)
		options.Headers[i] = h
		if h == "from" {
			hasFrom = true
		}
	}
	if !hasFrom {
		return ErrSignHeaderShouldContainsFrom
	}

	// Reorder headers to match the email's actual header order.
	// This makes h= order consistent with the message, preventing
	// verification failures with verifiers that check header ordering.
	options.Headers = reorderHeadersByEmail(email, options.Headers)

	// Normalize
	headers, body, err := canonicalize(email, options.Canonicalization, options.Headers)
	if err != nil {
		return err
	}

	signHash := strings.Split(options.Algo, "-")

	// hash body
	bodyHash, err := getBodyHash(&body, signHash[1], options.BodyLength)
	if err != nil {
		return err
	}

	// Get DKIM header base
	dkimHeader := newDkimHeaderBySigOptions(options)
	dHeader := dkimHeader.getHeaderBaseForSigning(bodyHash)

	canonicalizations := strings.Split(options.Canonicalization, "/")
	dHeaderCanonicalized, err := canonicalizeHeader(dHeader, canonicalizations[0])
	if err != nil {
		return err
	}
	headers = append(headers, []byte(dHeaderCanonicalized)...)
	headers = bytes.TrimRight(headers, " \r\n")

	// sign
	sig, err := getSignature(&headers, privateKey, signHash[1])
	if err != nil {
		return err
	}

	// add to DKIM header
	subh := ""
	l := len(subh)
	for _, c := range sig {
		subh += string(c)
		l++
		if l >= MaxHeaderLineLength {
			dHeader += subh + FWS
			subh = ""
			l = 0
		}
	}
	dHeader += subh + CRLF
	*email = append([]byte(dHeader), *email...)
	return nil
}

// canonicalize returns canonicalized version of header and body.
func canonicalize(email *[]byte, cano string, h []string) (headers, body []byte, err error) {
	body = []byte{}
	rxReduceWS := regexp.MustCompile(`[ \t]+`)

	rawHeaders, rawBody, err := getHeadersBody(email)
	if err != nil {
		return nil, nil, err
	}

	canonicalizations := strings.Split(cano, "/")

	// canonicalize header
	headersList, err := getHeadersList(&rawHeaders)
	if err != nil {
		return nil, nil, err
	}

	// For each header to keep, traverse all available headers
	// If multiple instances of a field, keep them from the bottom to the top
	var match *list.Element
	headersToKeepList := list.New()

	for _, headerToKeep := range h {
		match = nil
		headerToKeepToLower := strings.ToLower(headerToKeep)
		for e := headersList.Front(); e != nil; e = e.Next() {
			t := strings.Split(e.Value.(string), ":")
			if strings.ToLower(t[0]) == headerToKeepToLower {
				match = e
			}
		}
		if match != nil {
			headersToKeepList.PushBack(match.Value.(string) + "\r\n")
			headersList.Remove(match)
		}
	}

	for e := headersToKeepList.Front(); e != nil; e = e.Next() {
		cHeader, err := canonicalizeHeader(e.Value.(string), canonicalizations[0])
		if err != nil {
			return headers, body, err
		}
		headers = append(headers, []byte(cHeader)...)
	}

	// canonicalize body
	if canonicalizations[1] == "simple" {
		// "simple" body canonicalization: ignore empty lines at end, ensure single CRLF
		body = bytes.TrimRight(rawBody, "\r\n")
		body = append(body, []byte{13, 10}...)
	} else {
		// "relaxed" body canonicalization:
		// - Ignore all whitespace at the end of lines
		// - Reduce all sequences of WSP within a line to a single SP
		// - Ignore all empty lines at the end of the message body
		rawBody = rxReduceWS.ReplaceAll(rawBody, []byte(" "))
		for _, line := range bytes.SplitAfter(rawBody, []byte{10}) {
			line = bytes.TrimRight(line, " \r\n")
			body = append(body, line...)
			body = append(body, []byte{13, 10}...)
		}
		body = bytes.TrimRight(body, "\r\n")
		body = append(body, []byte{13, 10}...)
	}
	return
}

// canonicalizeHeader returns canonicalized version of header.
func canonicalizeHeader(header string, algo string) (string, error) {
	if algo == "simple" {
		// "simple" header canonicalization: no changes
		return header, nil
	} else if algo == "relaxed" {
		// "relaxed" header canonicalization:
		// 1. Convert header field names to lowercase
		// 2. Unfold all continuation lines
		// 3. Convert WSP sequences to single SP
		// 4. Delete WSP at end of unfolded header field value
		// 5. Delete WSP before and after colon separator
		kv := strings.SplitN(header, ":", 2)
		if len(kv) != 2 {
			return header, ErrBadMailFormatHeaders
		}
		k := strings.ToLower(kv[0])
		k = strings.TrimSpace(k)
		v := removeFWS(kv[1])
		return k + ":" + v + CRLF, nil
	}
	return header, ErrSignBadCanonicalization
}

// getBodyHash returns the hash (base64 encoded) of the body.
func getBodyHash(body *[]byte, algo string, bodyLength uint) (string, error) {
	var h hash.Hash
	if algo == "sha1" {
		h = sha1.New()
	} else {
		h = sha256.New()
	}
	toH := *body
	if bodyLength != 0 {
		if uint(len(toH)) < bodyLength {
			return "", ErrBadDKimTagLBodyTooShort
		}
		toH = toH[0:bodyLength]
	}

	h.Write(toH)
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// getSignature returns signature of toSign using private key.
func getSignature(toSign *[]byte, key *rsa.PrivateKey, algo string) (string, error) {
	var h1 hash.Hash
	var h2 crypto.Hash
	switch algo {
	case "sha1":
		h1 = sha1.New()
		h2 = crypto.SHA1
	case "sha256":
		h1 = sha256.New()
		h2 = crypto.SHA256
	default:
		return "", ErrVerifyInappropriateHashAlgo
	}

	h1.Write(*toSign)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, h2, h1.Sum(nil))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// reorderHeadersByEmail scans the email's actual header order and reorders
// the wanted header list to match. This ensures h= order and canonicalized
// hash input order are consistent with the email's header sequence,
// maximizing compatibility with verifiers that check header order.
func reorderHeadersByEmail(email *[]byte, wanted []string) []string {
	rawHeaders, _, err := getHeadersBody(email)
	if err != nil {
		return wanted
	}
	headersList, err := getHeadersList(&rawHeaders)
	if err != nil {
		return wanted
	}

	wantedSet := make(map[string]bool, len(wanted))
	for _, w := range wanted {
		wantedSet[strings.ToLower(w)] = true
	}

	var result []string
	added := make(map[string]bool)

	// Pass 1: collect wanted headers in the order they appear in the email
	for e := headersList.Front(); e != nil; e = e.Next() {
		t := strings.SplitN(e.Value.(string), ":", 2)
		if len(t) < 2 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(t[0]))
		if wantedSet[name] && !added[name] {
			result = append(result, name)
			added[name] = true
		}
	}

	// Pass 2: append any wanted headers not found in the email
	// (preserving original relative order for missing ones)
	for _, w := range wanted {
		wl := strings.ToLower(w)
		if !added[wl] {
			result = append(result, wl)
			added[wl] = true
		}
	}

	return result
}
func removeFWS(in string) string {
	rxReduceWS := regexp.MustCompile(`[ \t]+`)
	out := strings.Replace(in, "\n", "", -1)
	out = strings.Replace(out, "\r", "", -1)
	out = rxReduceWS.ReplaceAllString(out, " ")
	return strings.TrimSpace(out)
}

// validateCanonicalization validates canonicalization (c flag).
func validateCanonicalization(cano string) (string, error) {
	p := strings.Split(cano, "/")
	if len(p) > 2 {
		return "", ErrSignBadCanonicalization
	}
	if len(p) == 1 {
		cano = cano + "/simple"
	}
	for _, c := range p {
		if c != "simple" && c != "relaxed" {
			return "", ErrSignBadCanonicalization
		}
	}
	return cano, nil
}

// getHeadersList returns headers as list.
func getHeadersList(rawHeader *[]byte) (*list.List, error) {
	headersList := list.New()
	currentHeader := []byte{}
	for _, line := range bytes.SplitAfter(*rawHeader, []byte{10}) {
		if line[0] == 32 || line[0] == 9 {
			if len(currentHeader) == 0 {
				return headersList, ErrBadMailFormatHeaders
			}
			currentHeader = append(currentHeader, line...)
		} else {
			// New header, save current if exists
			if len(currentHeader) != 0 {
				headersList.PushBack(string(bytes.TrimRight(currentHeader, "\r\n")))
				currentHeader = []byte{}
			}
			currentHeader = append(currentHeader, line...)
		}
	}
	headersList.PushBack(string(currentHeader))
	return headersList, nil
}

// getHeadersBody returns headers and body parts.
func getHeadersBody(email *[]byte) ([]byte, []byte, error) {
	substitutedEmail := *email

	// only replace \n with \r\n when \r\n\r\n not exists
	if bytes.Index(*email, []byte{13, 10, 13, 10}) < 0 {
		substitutedEmail = bytes.Replace(*email, []byte{10}, []byte{13, 10}, -1)
	}

	parts := bytes.SplitN(substitutedEmail, []byte{13, 10, 13, 10}, 2)
	if len(parts) != 2 {
		return []byte{}, []byte{}, ErrBadMailFormat
	}
	// Empty body
	if len(parts[1]) == 0 {
		parts[1] = []byte{13, 10}
	}
	return parts[0], parts[1], nil
}
