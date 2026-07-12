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
	"path/filepath"
	"strings"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
)

type attachmentRiskPlugin struct{}

func (attachmentRiskPlugin) Key() string         { return "attachment_risk" }
func (attachmentRiskPlugin) Stage() filter.Stage { return filter.StageBody }

var riskyExt = map[string]struct{}{
	".exe": {}, ".dll": {}, ".scr": {}, ".bat": {}, ".cmd": {}, ".ps1": {}, ".vbs": {}, ".js": {}, ".jar": {}, ".lnk": {}, ".iso": {},
	".hta": {}, ".msi": {}, ".reg": {}, ".wsf": {},
}

func (attachmentRiskPlugin) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	if fc == nil {
		return nil, nil
	}
	_ = ctx

	total := len(fc.Attachments)
	if total == 0 {
		total = len(fc.AttachmentNames)
	}
	if total == 0 {
		return attachmentRiskZeros(), nil
	}

	risky := 0
	doubleExt := 0
	for _, raw := range fc.AttachmentNames {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if _, ok := riskyExt[ext]; ok {
			risky++
		}
		if strings.Count(name, ".") >= 2 {
			doubleExt++
		}
	}

	ratio := 0.0
	if total > 0 {
		ratio = float64(risky) / float64(total)
	}
	return filter.FeatureBatch{
		"attachment_count":               float64(total),
		"attachment_risky_ext_count":     float64(risky),
		"attachment_risky_ext_ratio":     ratio,
		"attachment_name_double_ext_cnt": float64(doubleExt),
	}, nil
}

func attachmentRiskZeros() filter.FeatureBatch {
	return filter.FeatureBatch{
		"attachment_count":               0,
		"attachment_risky_ext_count":     0,
		"attachment_risky_ext_ratio":     0,
		"attachment_name_double_ext_cnt": 0,
	}
}

func init() {
	rule.Register(attachmentRiskPlugin{})
}
