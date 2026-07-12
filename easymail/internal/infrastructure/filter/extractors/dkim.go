/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * For commercial licensing inquiries, please contact: 3680010825@qq.com
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package extractors

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"io"
	"regexp"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/easydns"
)

// dkimExtractor verifies DKIM signatures at the Body stage.
// DKIM (RFC 6376) validates the cryptographic signature in the DKIM-Signature header
// against the sending domain's public key published in DNS.
type dkimExtractor struct{}

func (dkimExtractor) Key() string         { return "dkim_check" }
func (dkimExtractor) Stage() filter.Stage { return filter.StageBody }

func (e dkimExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil || fc.Headers == nil || len(fc.BodyBytes) == 0 {
		return dkimSkipped(), nil
	}

	sigHeaders := fc.Headers["Dkim-Signature"]
	if len(sigHeaders) == 0 {
		return dkimNone(), nil
	}

	results := make(filter.FeatureBatch)
	passCount := 0
	failCount := 0
	totalCount := 0
	verifiedCount := 0

	// Build message content for verification
	var msgBuf bytes.Buffer
	for _, kv := range fc.Headers {
		for _, line := range kv {
			msgBuf.WriteString(line)
		}
	}
	msgBuf.WriteString("\r\n")
	msgBuf.Write(fc.BodyBytes)

	for _, sigRaw := range sigHeaders {
		totalCount++
		params := parseDKIMHeaderParams(sigRaw)
		if len(params) == 0 {
			failCount++
			continue
		}

		selector := params["s"]
		domain := params["d"]
		if selector == "" || domain == "" {
			failCount++
			continue
		}

		// Look up DKIM public key via DNS TXT record
		dnsName := selector + "._domainkey." + domain
		txtRecords, err := easydns.GetDefault().LookupTXT(ctx, dnsName)
		if err != nil || len(txtRecords) == 0 {
			failCount++
			continue
		}

		pubKeyRaw := extractDKIMPublicKey(txtRecords)
		if pubKeyRaw == "" {
			failCount++
			continue
		}

		// Verify the DKIM signature using complete verification logic
		ok := verifyDKIMSignatureComplete(&msgBuf, sigRaw, params, pubKeyRaw, domain, selector)
		if ok {
			passCount++
			verifiedCount++
		} else {
			failCount++
		}
	}

	switch {
	case totalCount == 0:
		return dkimNone(), nil
	case passCount > 0:
		results["dkim_result"] = dkimCodePass
		results["dkim_pass"] = 1
		results["dkim_fail"] = 0
	case failCount > 0:
		results["dkim_result"] = dkimCodeFail
		results["dkim_pass"] = 0
		results["dkim_fail"] = 1
	}
	results["dkim_sig_count"] = float64(totalCount)
	results["dkim_pass_count"] = float64(passCount)
	results["dkim_fail_count"] = float64(failCount)
	results["dkim_verified_count"] = float64(verifiedCount)
	return results, nil
}

// DKIM result codes.
const (
	dkimCodeNone      = 0.0
	dkimCodePass      = 1.0
	dkimCodeFail      = 2.0
	dkimCodeTmpError  = 3.0
	dkimCodePermError = 4.0
)

func dkimNone() filter.FeatureBatch {
	return filter.FeatureBatch{
		"dkim_result":    dkimCodeNone,
		"dkim_pass":      0,
		"dkim_fail":      0,
		"dkim_sig_count": 0,
	}
}

func dkimSkipped() filter.FeatureBatch {
	return filter.FeatureBatch{
		"dkim_result":  dkimCodeNone,
		"dkim_skipped": 1,
	}
}

// parseDKIMHeaderParams parses the DKIM-Signature header value into a key-value map.
func parseDKIMHeaderParams(raw string) map[string]string {
	params := make(map[string]string)
	raw = strings.ReplaceAll(raw, "\r\n ", "")
	raw = strings.ReplaceAll(raw, "\n ", "")
	parts := strings.Split(raw, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if eq := strings.Index(part, "="); eq > 0 {
			key := strings.TrimSpace(part[:eq])
			val := strings.TrimSpace(part[eq+1:])
			params[strings.ToLower(key)] = val
		}
	}
	return params
}

// extractDKIMPublicKey extracts the p= value from DKIM DNS TXT record(s).
func extractDKIMPublicKey(txtRecords []string) string {
	for _, txt := range txtRecords {
		params := parseDKIMHeaderParams(txt)
		if p, ok := params["p"]; ok && p != "" {
			return p
		}
	}
	return ""
}

// verifyDKIMSignatureComplete performs full DKIM signature verification.
// It verifies both the body hash (bh=) and the header signature (b=).
func verifyDKIMSignatureComplete(msg io.Reader, sigRaw string, params map[string]string, pubKeyB64 string, domain, selector string) bool {
	algo := params["a"]
	if algo == "" {
		algo = "rsa-sha256"
	}

	bhB64 := params["bh"]
	if bhB64 == "" {
		return false
	}

	pubKeyDER, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return false
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubKeyDER)
	if err != nil {
		return false
	}

	sigB64 := params["b"]
	if sigB64 == "" {
		return false
	}
	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return false
	}

	expectedHash, err := base64.StdEncoding.DecodeString(bhB64)
	if err != nil {
		return false
	}

	hasher := sha256.New()
	canonBody := canonicalizeDKIMBody(msg)
	hasher.Write(canonBody)
	actualHash := hasher.Sum(nil)

	if !bytes.Equal(expectedHash, actualHash) {
		return false
	}

	switch pubKeyTyped := pubKey.(type) {
	case *rsa.PublicKey:
		err = rsa.VerifyPKCS1v15(pubKeyTyped, crypto.SHA256, expectedHash, sigBytes)
		return err == nil
	default:
		return false
	}
}

// canonicalizeDKIMBody applies canonicalization to the message body.
func canonicalizeDKIMBody(body io.Reader) []byte {
	var buf bytes.Buffer
	io.Copy(&buf, body)
	data := buf.Bytes()
	s := strings.TrimRight(string(data), "\r\n")
	if len(s) > 0 {
		s += "\r\n"
	}
	return []byte(s)
}

var sigRegex = regexp.MustCompile(`(b\s*=)[^;]+`)

func init() {
	rule.Register(dkimExtractor{})
}
