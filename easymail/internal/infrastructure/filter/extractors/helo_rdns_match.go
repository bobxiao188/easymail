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

// heloRDNSMatchExtractor compares HELO name with rDNS (PTR) names (best-effort).
type heloRDNSMatchExtractor struct{}

func (heloRDNSMatchExtractor) Key() string         { return "helo_rdns_match" }
func (heloRDNSMatchExtractor) Stage() filter.Stage { return filter.StageHelo }
func (heloRDNSMatchExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil {
		return nil, nil
	}
	helo := normalizeDNSName(fc.HeloName)
	if helo == "" {
		return filter.FeatureBatch{"helo_present": 0}, nil
	}

	// If connect stage already stored rDNS names, prefer them.
	rdns := fc.RDNSNames
	if len(rdns) == 0 && fc.ConnectIP != nil {
		ptrs, _ := dnsLookupPTR(ctx, fc.ConnectIP.String())
		for i := 0; i < len(ptrs) && i < 3; i++ {
			rdns = append(rdns, normalizeDNSName(ptrs[i]))
		}
	}

	match := 0.0
	for _, n := range rdns {
		n = normalizeDNSName(n)
		if n == "" {
			continue
		}
		if n == helo || strings.HasSuffix(helo, "."+n) || strings.HasSuffix(n, "."+helo) {
			match = 1
			break
		}
	}
	return filter.FeatureBatch{
		"helo_present":         1,
		"helo_rdns_match":      match,
		"helo_rdns_name_count": float64(len(rdns)),
	}, nil
}

func init() {
	rule.Register(heloRDNSMatchExtractor{})
}
