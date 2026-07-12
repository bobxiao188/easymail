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
	"context"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/easydns"

	"github.com/weppos/publicsuffix-go/publicsuffix"
)

// dmarcExtractor validates DMARC policy at the Body stage.
// DMARC (RFC 7489) builds on SPF and DKIM results to determine whether the
// sending domain's policy is aligned (pass) or not (fail).
//
// The check:
// 1. Extract the From: header domain (RFC 5322 From)
// 2. Look up _dmarc.<domain> TXT record
// 3. Parse the policy (p=) and subdomain policy (sp=)
// 4. Check SPF alignment (domain in RFC 5321 MAIL FROM must align with From domain)
// 5. Check DKIM alignment (d= domain in DKIM must align with From domain)
// 6. Report dmarc_pass/dmarc_fail/dmarc_none based on policy evaluation
type dmarcExtractor struct{}

func (dmarcExtractor) Key() string         { return "dmarc_check" }
func (dmarcExtractor) Stage() filter.Stage { return filter.StageBody }

func (e dmarcExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	feat := filter.FeatureBatch{}

	// 1. Extract the From domain
	fromDomain := extractFromDomainFromHeaders(fc)
	if fromDomain == "" {
		return dmarcNone(), nil
	}

	// 2. Look up DMARC policy
	dmarcDomain := "_dmarc." + fromDomain
	txtRecords, err := easydns.GetDefault().LookupTXT(ctx, dmarcDomain)
	if err != nil || len(txtRecords) == 0 {
		return dmarcNone(), nil
	}

	dmarcPolicy := parseDMARCPolicy(txtRecords)
	if dmarcPolicy == nil {
		return dmarcNone(), nil
	}

	// 3. Check SPF alignment
	spfAligned := checkSPFAlignment(fc, fromDomain)

	// 4. Check DKIM alignment
	dkimAligned := checkDKIMAlignment(fc, fromDomain)

	// 5. Evaluate DMARC result
	dmarcPass := spfAligned || dkimAligned

	// 6. Build features
	pVal := dmarcPolicy["p"]
	if pVal == "" {
		pVal = "none"
	}
	policyCode := dmarcPolicyToCode(pVal)
	spVal := dmarcPolicy["sp"]
	if spVal == "" {
		spVal = pVal
	}

	feat["dmarc_domain"] = 0
	feat["dmarc_policy_none"] = boolToFloat64(pVal == "none")
	feat["dmarc_policy_quarantine"] = boolToFloat64(pVal == "quarantine")
	feat["dmarc_policy_reject"] = boolToFloat64(pVal == "reject")
	feat["dmarc_policy_code"] = policyCode
	feat["dmarc_spf_aligned"] = boolToFloat64(spfAligned)
	feat["dmarc_dkim_aligned"] = boolToFloat64(dkimAligned)
	feat["dmarc_spf_domain_match"] = boolToFloat64(spfAligned)
	feat["dmarc_dkim_domain_match"] = boolToFloat64(dkimAligned)

	if dmarcPass {
		feat["dmarc_result"] = dmarcCodePass
		feat["dmarc_pass"] = 1
		feat["dmarc_fail"] = 0
	} else {
		feat["dmarc_result"] = dmarcCodeFail
		feat["dmarc_pass"] = 0
		feat["dmarc_fail"] = 1
	}
	feat["dmarc_has_policy"] = 1

	return feat, nil
}

// DMARC result codes.
const (
	dmarcCodeNone = 0.0
	dmarcCodePass = 1.0
	dmarcCodeFail = 2.0
)

func dmarcNone() filter.FeatureBatch {
	return filter.FeatureBatch{
		"dmarc_result":     dmarcCodeNone,
		"dmarc_has_policy": 0,
		"dmarc_pass":       0,
		"dmarc_fail":       0,
	}
}

// extractFromDomainFromHeaders gets the domain from the From: header.
func extractFromDomainFromHeaders(fc *filter.MilterContext) string {
	if fc == nil || fc.Headers == nil {
		return ""
	}
	fromVals := fc.Headers["From"]
	if len(fromVals) == 0 {
		return ""
	}
	return extractDomainFromHeaderAddr(fromVals[0])
}

// parseDMARCPolicy extracts DMARC policy key-values from DNS TXT records.
func parseDMARCPolicy(txtRecords []string) map[string]string {
	for _, txt := range txtRecords {
		txt = strings.TrimSpace(txt)
		if !strings.Contains(strings.ToLower(txt), "v=dmarc1") {
			continue
		}
		params := make(map[string]string)
		parts := strings.Split(txt, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if eq := strings.Index(part, "="); eq > 0 {
				key := strings.TrimSpace(strings.ToLower(part[:eq]))
				val := strings.TrimSpace(part[eq+1:])
				params[key] = val
			}
		}
		return params
	}
	return nil
}

// checkSPFAlignment checks if the SPF-authenticated domain aligns with the From domain.
// Two modes: strict (exact match) and relaxed (same organizational domain).
// This implementation uses relaxed alignment (same organizational domain).
func checkSPFAlignment(fc *filter.MilterContext, fromDomain string) bool {
	// Get the SPF-authenticated domain from the MAIL FROM (envelope sender)
	sender := strings.TrimSpace(fc.MailFrom)
	if sender == "" {
		return false
	}
	// Extract domain from email address (e.g., "user@example.com" -> "example.com")
	mailFromDomain := extractDomainFromAddr(sender)
	return domainsAlign(mailFromDomain, fromDomain)
}

// checkDKIMAlignment checks if any DKIM signature domain aligns with the From domain.
func checkDKIMAlignment(fc *filter.MilterContext, fromDomain string) bool {
	if fc.Headers == nil {
		return false
	}
	sigHeaders := fc.Headers["Dkim-Signature"]
	for _, sigRaw := range sigHeaders {
		params := parseDKIMHeaderParams(sigRaw)
		if d, ok := params["d"]; ok && d != "" {
			if domainsAlign(strings.ToLower(d), fromDomain) {
				return true
			}
		}
	}
	return false
}

// domainsAlign checks DMARC alignment (relaxed mode: same organizational domain).
// Uses Public Suffix List (PSL) to correctly identify registered domains.
// For strict mode, just compare exactly.
func domainsAlign(authDomain, fromDomain string) bool {
	authDomain = normalizeDNSName(authDomain)
	fromDomain = normalizeDNSName(fromDomain)
	if authDomain == "" || fromDomain == "" {
		return false
	}
	// Exact match
	if authDomain == fromDomain {
		return true
	}
	// Relaxed: use PSL to get organizational domain
	authOrg := getOrganizationalDomain(authDomain)
	fromOrg := getOrganizationalDomain(fromDomain)
	return authOrg != "" && fromOrg != "" && authOrg == fromOrg
}

// getOrganizationalDomain returns the registered domain (effective TLD + 1) using PSL.
// For example: "mail.example.com" -> "example.com", "mail.example.co.uk" -> "example.co.uk"
func getOrganizationalDomain(domain string) string {
	if domain == "" {
		return ""
	}
	// Use PSL to get the registrable domain (e.g., example.com from mail.example.com)
	regDomain, err := publicsuffix.Domain(domain)
	if err != nil {
		return ""
	}
	return regDomain
}

// relaxedOrgDomain returns the organizational domain (effective TLD + 1) for the given domain.
// It uses the Public Suffix List (PSL) to correctly identify registered domains.
// For example: "mail.example.com" -> "example.com", "mail.example.co.uk" -> "example.co.uk"
func relaxedOrgDomain(domain string) string {
	return getOrganizationalDomain(domain)
}

// dmarcPolicyToCode maps policy string to numeric code.
func dmarcPolicyToCode(p string) float64 {
	switch strings.ToLower(p) {
	case "none":
		return 0
	case "quarantine":
		return 1
	case "reject":
		return 2
	default:
		return 0
	}
}

func init() {
	rule.Register(dmarcExtractor{})
}
