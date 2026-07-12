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

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
)

// fcrdnsExtractor checks forward-confirmed rDNS: PTR -> A/AAAA includes original IP.
type fcrdnsExtractor struct{}

func (fcrdnsExtractor) Key() string         { return "connect_fcrdns" }
func (fcrdnsExtractor) Stage() filter.Stage { return filter.StageConnect }
func (fcrdnsExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil || fc.ConnectIP == nil {
		return nil, nil
	}

	ptrs, err := dnsLookupPTR(ctx, fc.ConnectIP.String())
	if err != nil || len(ptrs) == 0 {
		return filter.FeatureBatch{
			"ip_fcrdns_ok":               0,
			"ip_ptr_forward_match_count": 0,
		}, nil
	}

	match := 0
	for i := 0; i < len(ptrs) && i < 3; i++ {
		host := normalizeDNSName(ptrs[i])
		if host == "" {
			continue
		}
		ips, err := dnsLookupIP(ctx, host)
		if err != nil || len(ips) == 0 {
			continue
		}
		for _, ip := range ips {
			if ip.IP != nil && ip.IP.Equal(fc.ConnectIP) {
				match++
				break
			}
		}
	}

	ok := 0.0
	if match > 0 {
		ok = 1
	}
	return filter.FeatureBatch{
		"ip_fcrdns_ok":               ok,
		"ip_ptr_forward_match_count": float64(match),
	}, nil
}

func init() {
	rule.Register(fcrdnsExtractor{})
}
