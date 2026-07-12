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

type connectRDNSExtractor struct{}

func (connectRDNSExtractor) Key() string         { return "connect_rdns" }
func (connectRDNSExtractor) Stage() filter.Stage { return filter.StageConnect }
func (connectRDNSExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil || fc.ConnectIP == nil {
		return nil, nil
	}
	ptrs, err := dnsLookupPTR(ctx, fc.ConnectIP.String())
	if err != nil || len(ptrs) == 0 {
		return filter.FeatureBatch{
			"ip_ptr_ok":      0,
			"ip_ptr_count":   0,
			"ip_ptr_has_dot": 0,
		}, nil
	}
	// Keep a few names for later stages (best-effort).
	fc.RDNSNames = fc.RDNSNames[:0]
	for i := 0; i < len(ptrs) && i < 3; i++ {
		fc.RDNSNames = append(fc.RDNSNames, normalizeDNSName(ptrs[i]))
	}
	hasDot := 0.0
	if strings.Contains(ptrs[0], ".") {
		hasDot = 1
	}
	return filter.FeatureBatch{
		"ip_ptr_ok":      1,
		"ip_ptr_count":   float64(len(ptrs)),
		"ip_ptr_has_dot": hasDot,
	}, nil
}

func init() {
	rule.Register(connectRDNSExtractor{})
}
