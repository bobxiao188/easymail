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

package filter

import (
	"context"
	"testing"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/filter/extractors"
)

type slowExtractor struct {
	stage filter.Stage
	key   string
	sleep time.Duration
}

func (s slowExtractor) Key() string { return s.key }
func (s slowExtractor) Stage() filter.Stage {
	return s.stage
}
func (s slowExtractor) Run(ctx context.Context, fc *filter.MilterContext) (filter.FeatureBatch, error) {
	select {
	case <-time.After(s.sleep):
		return filter.FeatureBatch{"x_" + s.key: 1}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestRunStageExtractors_TimeoutDoesNotBlock(t *testing.T) {
	// Save/restore global registry to avoid leaking across tests.
	old := rule.TestOnlySwapRegistry(nil)
	defer rule.TestOnlySwapRegistry(old)

	rule.Register(slowExtractor{stage: filter.StageConnect, key: "fast", sleep: 10 * time.Millisecond})
	rule.Register(slowExtractor{stage: filter.StageConnect, key: "slow", sleep: 200 * time.Millisecond})

	fc := filter.NewMilterContext()
	t0 := time.Now()
	res := extractors.RunStage(context.Background(), filter.StageConnect, fc, 50*time.Millisecond)
	if time.Since(t0) > 120*time.Millisecond {
		t.Fatalf("took too long: %v", time.Since(t0))
	}
	if len(res) == 0 {
		t.Fatalf("expected some results")
	}
	feat := fc.Snapshot()
	if feat["x_fast"] != 1 {
		t.Fatalf("expected fast extractor feature")
	}
}
