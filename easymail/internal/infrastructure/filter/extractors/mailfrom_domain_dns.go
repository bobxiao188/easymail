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
)

type mailFromDomainDNSExtractor struct{}

func (mailFromDomainDNSExtractor) Key() string         { return "mailfrom_domain_dns" }
func (mailFromDomainDNSExtractor) Stage() filter.Stage { return filter.StageMailFrom }
func (mailFromDomainDNSExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil {
		return nil, nil
	}
	s := strings.TrimSpace(fc.MailFrom)
	if s == "" {
		return nil, nil
	}
	at := strings.LastIndex(s, "@")
	if at < 0 || at == len(s)-1 {
		return filter.FeatureBatch{
			"sender_domain_has_mx": 0,
			"sender_domain_has_a":  0,
		}, nil
	}
	domain := strings.ToLower(strings.TrimSpace(s[at+1:]))
	if domain == "" {
		return nil, nil
	}

	mxs, _ := dnsLookupMX(ctx, domain)
	hasMX := 0.0
	mxCount := float64(len(mxs))
	mxPrefMin := 0.0
	mxNull := 0.0
	if len(mxs) > 0 {
		hasMX = 1
		minPref := int64(-1)
		for _, mx := range mxs {
			if mx == nil {
				continue
			}
			host := normalizeDNSName(mx.Host)
			if host == "" || host == "." {
				mxNull = 1
			}
			if minPref < 0 || int64(mx.Pref) < minPref {
				minPref = int64(mx.Pref)
			}
		}
		if minPref >= 0 {
			mxPrefMin = float64(minPref)
		}
	}

	ips, _ := dnsLookupIP(ctx, domain)
	hasA := 0.0
	aCount := float64(len(ips))
	if len(ips) > 0 {
		hasA = 1
	}
	return filter.FeatureBatch{
		"sender_domain_has_mx":      hasMX,
		"sender_domain_has_a":       hasA,
		"sender_domain_mx_count":    mxCount,
		"sender_domain_mx_pref_min": mxPrefMin,
		"sender_domain_mx_is_null":  mxNull,
		"sender_domain_a_count":     aCount,
	}, nil
}

func init() {
	rule.Register(mailFromDomainDNSExtractor{})
}
