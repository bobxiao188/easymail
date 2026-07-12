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
package rule

import (
	"context"
	"easymail/internal/domain/filter"
)

var registry []FeatureExtractor

// TestOnlySwapRegistry swaps the global extractor registry for tests.
func TestOnlySwapRegistry(next []FeatureExtractor) []FeatureExtractor {
	prev := registry
	registry = next
	return prev
}

// Register adds a built-in stage extractor.
func Register(e FeatureExtractor) {
	if e == nil {
		return
	}
	registry = append(registry, e)
}

// BuiltinStageByKeyWithPlugins maps built-in extractor keys and content-plugin keys to their pipeline stage.
func BuiltinStageByKeyWithPlugins() map[string]filter.Stage {
	out := make(map[string]filter.Stage)
	for _, e := range registry {
		if e == nil {
			continue
		}
		out[e.Key()] = e.Stage()
	}
	for _, p := range plugins {
		if p == nil {
			continue
		}
		out["plugin:"+p.Key()] = filter.StageBody
	}
	return out
}

// ForStage returns registered extractors for s.
func ForStage(s filter.Stage) []FeatureExtractor {
	if len(registry) == 0 {
		return nil
	}
	out := make([]FeatureExtractor, 0, 8)
	for _, e := range registry {
		if e != nil && e.Stage() == s {
			out = append(out, e)
		}
	}
	return out
}

var plugins []ContentPlugin

// RegisterContentPlugin registers a body-stage content plugin.
func RegisterContentPlugin(p ContentPlugin) {
	if p == nil {
		return
	}
	plugins = append(plugins, p)
}

// RunPlugins runs registered content plugins concurrently and merges numeric features into fc.
func RunPlugins(ctx context.Context, fc *filter.MilterContext) {
	if fc == nil || len(plugins) == 0 {
		return
	}
	ch := make(chan struct {
		m   filter.FeatureBatch
		err error
	}, len(plugins))
	for _, p := range plugins {
		p := p
		go func() {
			m, err := p.Run(ctx, fc)
			ch <- struct {
				m   filter.FeatureBatch
				err error
			}{m, err}
		}()
	}
	for i := 0; i < len(plugins); i++ {
		select {
		case r := <-ch:
			if len(r.m) > 0 {
				fc.Merge(r.m)
			}
		case <-ctx.Done():
			return
		}
	}
}
