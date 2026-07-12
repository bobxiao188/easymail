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
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/persistence/mysql"
	"easymail/internal/pkg/rfc2047"

	"easymail/pkg/database"

	enmime "github.com/jhillyerd/enmime/v2"
	"gorm.io/gorm"
)

type customFeatureType string

const (
	customTypeMetaRegex customFeatureType = "meta_regex"
	customTypeComposite customFeatureType = "composite"
)

type metaRegexSpec struct {
	Sources []string `json:"sources"`
	Pattern string   `json:"pattern"`
	Flags   string   `json:"flags"` // i,m,s
	Mode    string   `json:"mode"`  // any|all
	Emit    string   `json:"emit"`  // bool_hit|count
}

type compositeSpec struct {
	ConditionJSON string `json:"condition_json"`
	Emit          string `json:"emit"` // bool
}

type compiledCustomFeature struct {
	Row rule.CustomFeature

	Type customFeatureType

	// Stage is the earliest milter phase at which this definition may be evaluated (max of dependency stages).
	Stage filter.Stage

	MetaSpec   *metaRegexSpec
	MetaRegexp *regexp.Regexp

	CompSpec *compositeSpec
}

const customFeatureDefsTTL = 10 * time.Second

var customDefsCache cache.MemoryTTL[[]compiledCustomFeature]

// InvalidateCustomFeatureDefsCache drops compiled custom-feature definitions and the infrastructure feature-key stage cache; call after admin custom-feature CUD.
func InvalidateCustomFeatureDefsCache() {
	cache.InvalidateFeatureKeyStagesCache()
	customDefsCache.Invalidate()
}

// normalizeMetaSource maps admin/legacy aliases to canonical source names used by sourceTexts / sourceReadyAtStage.
func normalizeMetaSource(src string) string {
	s := strings.ToLower(strings.TrimSpace(src))
	switch s {
	case "subject":
		return "subject"
	default:
		return s
	}
}

// reloadCustomDefsFromDB loads and compiles enabled custom features (no TTL; used inside loadCustomDefs).
func reloadCustomDefsFromDB(ctx context.Context, db *gorm.DB) ([]compiledCustomFeature, error) {
	var rows []rule.CustomFeature
	if err := db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]compiledCustomFeature, 0, len(rows))
	for _, r := range rows {
		t := customFeatureType(strings.ToLower(strings.TrimSpace(r.Type)))
		switch t {
		case customTypeMetaRegex:
			var s metaRegexSpec
			if err := json.Unmarshal([]byte(r.SpecJSON), &s); err != nil {
				continue
			}
			pat := strings.TrimSpace(s.Pattern)
			if pat == "" || len(s.Sources) == 0 {
				continue
			}
			flags := strings.ToLower(strings.TrimSpace(s.Flags))
			if strings.Contains(flags, "i") {
				pat = "(?i)" + pat
			}
			if strings.Contains(flags, "m") {
				pat = "(?m)" + pat
			}
			if strings.Contains(flags, "s") {
				pat = "(?s)" + pat
			}
			re, err := regexp.Compile(pat)
			if err != nil {
				continue
			}
			out = append(out, compiledCustomFeature{
				Row:        r,
				Type:       t,
				MetaSpec:   &s,
				MetaRegexp: re,
			})
		case customTypeComposite:
			var s compositeSpec
			if err := json.Unmarshal([]byte(r.SpecJSON), &s); err != nil {
				continue
			}
			if strings.TrimSpace(s.ConditionJSON) == "" {
				continue
			}
			if err := rule.ValidateConditionJSON(s.ConditionJSON); err != nil {
				continue
			}
			out = append(out, compiledCustomFeature{
				Row:      r,
				Type:     t,
				CompSpec: &s,
			})
		default:
			continue
		}
	}

	assignCustomStages(&out)
	return out, nil
}

// loadCustomDefs loads custom features from database with a short in-memory TTL.
func loadCustomDefs(ctx context.Context, db *gorm.DB) ([]compiledCustomFeature, error) {
	if db == nil {
		return nil, nil
	}
	return customDefsCache.Get(time.Now(), customFeatureDefsTTL, func() ([]compiledCustomFeature, error) {
		return reloadCustomDefsFromDB(ctx, db)
	})
}

// assignCustomStages assigns custom feature stages to the features.
func assignCustomStages(defs *[]compiledCustomFeature) {
	stages := rule.BuiltinStageByKeyWithPlugins()
	for i := range *defs {
		d := &(*defs)[i]
		if d.Type != customTypeMetaRegex || d.MetaSpec == nil {
			continue
		}
		mx := filter.StageConnect
		for _, src := range d.MetaSpec.Sources {
			mx = filter.MaxStage(mx, sourceMinStage(normalizeMetaSource(src)))
		}
		d.Stage = mx
		if k := strings.TrimSpace(d.Row.FeatureKey); k != "" {
			stages[k] = mx
		}
	}
	for round := 0; round < len(*defs)+3; round++ {
		changed := false
		for i := range *defs {
			d := &(*defs)[i]
			if d.Type != customTypeComposite || d.CompSpec == nil {
				continue
			}
			k := strings.TrimSpace(d.Row.FeatureKey)
			keys, err := rule.CollectConditionFeatureKeys(d.CompSpec.ConditionJSON)
			if err != nil || len(keys) == 0 {
				if d.Stage != filter.StageBody {
					d.Stage = filter.StageBody
					changed = true
				}
				if k != "" {
					stages[k] = d.Stage
				}
				continue
			}
			mx := filter.StageConnect
			for _, fk := range keys {
				st := filter.StageBody
				if s, ok := stages[fk]; ok {
					st = s
				}
				mx = filter.MaxStage(mx, st)
			}
			if d.Stage != mx {
				d.Stage = mx
				changed = true
			}
			if k != "" {
				stages[k] = d.Stage
			}
		}
		if !changed {
			break
		}
	}
}

func featureKeyStagesIntsUncached(ctx context.Context, db *gorm.DB) map[string]int {
	builtin := rule.BuiltinStageByKeyWithPlugins()
	m := make(map[string]int, len(builtin)+8)
	for k, v := range builtin {
		m[k] = int(v)
	}
	defs, err := loadCustomDefs(ctx, db)
	if err != nil || len(defs) == 0 {
		return m
	}
	for _, d := range defs {
		k := strings.TrimSpace(d.Row.FeatureKey)
		if k == "" {
			continue
		}
		m[k] = int(d.Stage)
	}
	var pos []mysql.ClassifyModelPO
	if err := db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).Where("enabled = ? AND is_deleted = ?", true, false).Order("id ASC").Find(&pos).Error; err == nil {
		body := int(filter.StageBody)
		for i := range pos {
			cm := mysql.PoToClassifyModel(&pos[i])
			for _, ent := range rule.RuleFeatureEntries(*cm) {
				if ent.Key != "" {
					m[ent.Key] = body
				}
			}
		}
	}
	return m
}

// FeatureKeyStages returns resolved pipeline stages for feature keys (builtin, plugins, custom).
func FeatureKeyStages(ctx context.Context, db *gorm.DB) map[string]filter.Stage {
	if db == nil {
		return rule.BuiltinStageByKeyWithPlugins()
	}
	ints := cache.CachedFeatureKeyStages(ctx, func(ctx context.Context) map[string]int {
		return featureKeyStagesIntsUncached(ctx, db)
	})
	out := make(map[string]filter.Stage, len(ints))
	for k, v := range ints {
		out[k] = filter.Stage(v)
	}
	return out
}

func applyCustomFeaturesForStage(ctx context.Context, stage filter.Stage, fc *filter.MilterContext) error {
	if fc == nil {
		return nil
	}
	db := database.GetDB()
	if db == nil {
		return nil
	}
	defs, err := loadCustomDefs(ctx, db)
	if err != nil || len(defs) == 0 {
		return err
	}

	// Snapshot once for composite feature evaluation; meta_regex reads raw fields directly.
	baseSnap := fc.Snapshot()

	for _, d := range defs {
		if d.Stage != stage {
			continue
		}
		key := strings.TrimSpace(d.Row.FeatureKey)
		if key == "" {
			continue
		}
		switch d.Type {
		case customTypeMetaRegex:
			v, ok := evalMetaRegexForStage(ctx, stage, fc, d.MetaSpec, d.MetaRegexp)
			if ok {
				fc.Set(key, v)
			}
		case customTypeComposite:
			ok, e := rule.EvalConditionJSON(d.CompSpec.ConditionJSON, baseSnap, nil)
			if e != nil {
				continue
			}
			if ok {
				fc.Set(key, 1)
			} else {
				fc.Set(key, 0)
			}
		}
	}
	return nil
}

func evalMetaRegexForStage(ctx context.Context, stage filter.Stage, fc *filter.MilterContext, spec *metaRegexSpec, re *regexp.Regexp) (val float64, decided bool) {
	_ = ctx
	if fc == nil || spec == nil || re == nil {
		return 0, false
	}
	mode := strings.ToLower(strings.TrimSpace(spec.Mode))
	if mode == "" {
		mode = "any"
	}
	emit := strings.ToLower(strings.TrimSpace(spec.Emit))
	if emit == "" {
		emit = "bool_hit"
	}

	// Evaluate per-source; then combine by mode.
	hits := make([]bool, 0, len(spec.Sources))
	hitCount := 0
	evaluatedSources := 0
	skippedSources := 0

	for _, src := range spec.Sources {
		src = normalizeMetaSource(src)
		if stage < sourceMinStage(src) {
			skippedSources++
			continue
		}
		texts := sourceTexts(fc, src)
		evaluatedSources++
		srcHit := false
		srcCount := 0
		for _, t := range texts {
			if t == "" {
				continue
			}
			locs := re.FindAllStringIndex(t, -1)
			if len(locs) > 0 {
				srcHit = true
				srcCount += len(locs)
				// short-circuit for bool_hit
				if emit == "bool_hit" {
					break
				}
			}
		}
		hits = append(hits, srcHit)
		hitCount += srcCount
	}

	if evaluatedSources == 0 {
		// Not enough data at this stage; don't set the feature yet.
		return 0, false
	}

	combined := false
	if len(hits) == 0 {
		combined = false
	} else if mode == "all" {
		// If any requested source isn't ready yet, we can't satisfy "all" early.
		if skippedSources > 0 {
			combined = false
		} else {
			combined = true
			for _, h := range hits {
				if !h {
					combined = false
					break
				}
			}
		}
	} else {
		combined = false
		for _, h := range hits {
			if h {
				combined = true
				break
			}
		}
	}

	switch emit {
	case "count":
		if combined {
			return float64(hitCount), true
		}
		return 0, true
	default:
		if combined {
			return 1, true
		}
		return 0, true
	}
}

func sourceTexts(fc *filter.MilterContext, src string) []string {
	if fc == nil {
		return nil
	}
	src = normalizeMetaSource(src)
	switch src {
	case "connect_ip":
		if fc.ConnectIP == nil {
			return nil
		}
		return []string{fc.ConnectIP.String()}
	case "mail_from":
		return []string{strings.TrimSpace(fc.MailFrom)}
	case "rcpt":
		out := make([]string, 0, len(fc.Rcpts))
		for _, r := range fc.Rcpts {
			r = strings.TrimSpace(r)
			if r != "" {
				out = append(out, r)
			}
		}
		return out
	case "subject":
		if fc.Headers == nil {
			return nil
		}
		raw := strings.TrimSpace(fc.Headers.Get("Subject"))
		return []string{rfc2047.DecodeHeader(raw)}
	case "header_from_email", "header_from_name":
		// Best-effort parse: "Name <a@b>" or "a@b"
		if fc.Headers == nil {
			return nil
		}
		from := strings.TrimSpace(fc.Headers.Get("From"))
		if from == "" {
			return nil
		}
		name, email := splitFromHeader(from)
		if src == "header_from_email" {
			return []string{email}
		}
		return []string{name}
	case "body":
		if s := strings.TrimSpace(fc.TextBody); s != "" {
			return []string{s}
		}
		if len(fc.BodyBytes) == 0 {
			return nil
		}
		// limit to avoid huge regex cost
		b := fc.BodyBytes
		if len(b) > 1<<20 {
			b = b[:1<<20]
		}
		return []string{string(b)}
	case "url_list":
		if len(fc.URLList) > 0 {
			u := append([]string(nil), fc.URLList...)
			sort.Strings(u)
			return []string{strings.Join(u, "\n")}
		}
		urls := ExtractURLsFromBody(fc.BodyBytes)
		if len(urls) == 0 {
			return nil
		}
		// stable order for determinism
		sort.Strings(urls)
		return []string{strings.Join(urls, "\n")}
	case "attachment_names":
		if len(fc.AttachmentNames) > 0 {
			n := append([]string(nil), fc.AttachmentNames...)
			sort.Strings(n)
			return []string{strings.Join(n, "\n")}
		}
		names := ExtractAttachmentNamesFromRFC822(fc.BodyBytes)
		if len(names) == 0 {
			return nil
		}
		sort.Strings(names)
		return []string{strings.Join(names, "\n")}
	default:
		return nil
	}
}

// sourceMinStage is the earliest pipeline phase at which the meta_regex source has data.
func sourceMinStage(src string) filter.Stage {
	src = normalizeMetaSource(src)
	switch src {
	case "connect_ip":
		return filter.StageConnect
	case "mail_from":
		return filter.StageMailFrom
	case "rcpt":
		return filter.StageRcptTo
	case "subject", "header_from_email", "header_from_name":
		return filter.StageHeaders
	case "body", "url_list", "attachment_names":
		return filter.StageBody
	default:
		return filter.StageBody
	}
}

func splitFromHeader(v string) (name string, email string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", ""
	}
	// crude parse, good enough for feature engineering
	if i := strings.LastIndex(v, "<"); i >= 0 {
		if j := strings.LastIndex(v, ">"); j > i {
			email = strings.TrimSpace(v[i+1 : j])
			name = strings.TrimSpace(strings.Trim(v[:i], "\""))
			return name, email
		}
	}
	// no angle brackets: assume email
	return "", v
}

var urlListRe = regexp.MustCompile(`https?://[^\s<>"']+`)

// ExtractURLsFromBody returns unique http(s) URLs found in raw body bytes (best-effort).
func ExtractURLsFromBody(body []byte) []string {
	if len(body) == 0 {
		return nil
	}
	b := body
	if len(b) > 1<<20 {
		b = b[:1<<20]
	}
	s := string(b)
	ms := urlListRe.FindAllString(s, -1)
	if len(ms) == 0 {
		return nil
	}
	uniq := make(map[string]struct{}, len(ms))
	out := make([]string, 0, len(ms))
	for _, u := range ms {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := uniq[u]; ok {
			continue
		}
		uniq[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

// ExtractAttachmentNamesFromRFC822 parses MIME and returns attachment filenames (best-effort).
func ExtractAttachmentNamesFromRFC822(rfc822 []byte) []string {
	if len(rfc822) == 0 {
		return nil
	}
	env, err := enmime.ReadEnvelope(bytes.NewReader(rfc822))
	if err != nil {
		return nil
	}
	if len(env.Attachments) == 0 {
		return nil
	}
	out := make([]string, 0, len(env.Attachments))
	for _, a := range env.Attachments {
		n := strings.TrimSpace(a.FileName)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}
