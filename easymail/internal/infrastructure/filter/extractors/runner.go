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
	"log"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/feature"
	"easymail/internal/domain/filter/rule"
)

// RunStage runs all registered extractors for a stage concurrently.
// It waits up to timeout; results arriving after timeout are ignored.
func RunStage(parent context.Context, stage filter.Stage, fc *filter.MilterContext, timeout time.Duration) []feature.Result {
	es := rule.ForStage(stage)
	if len(es) == 0 {
		return nil
	}
	if timeout <= 0 {
		timeout = 200 * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	log.Printf("milter_trace_runstage stage=%d extractors=%d timeout_ms=%d start", stage, len(es), timeout.Milliseconds())
	ch := make(chan feature.Result, len(es))
	for _, ex := range es {
		ex := ex
		go func() {
			e0 := time.Now()
			feat, err := ex.Run(ctx, fc)
			log.Printf("milter_trace_runstage key=%s elapsed_ms=%d err=%v", ex.Key(), time.Since(e0).Milliseconds(), err)
			ch <- feature.Result{Key: ex.Key(), Features: feat, Err: err}
		}()
	}

	results := make([]feature.Result, 0, len(es))
	t0 := time.Now()
	for i := 0; i < len(es); i++ {
		select {
		case r := <-ch:
			if len(r.Features) > 0 {
				fc.Merge(r.Features)
			}
			results = append(results, r)
			log.Printf("milter_trace_runstage stage=%d received key=%s elapsed_ms=%d total_received=%d", stage, r.Key, time.Since(t0).Milliseconds(), len(results))
		case <-ctx.Done():
			log.Printf("milter_trace_runstage stage=%d timeout_at_ms=%d received=%d/%d", stage, time.Since(t0).Milliseconds(), len(results), len(es))
			return results
		}
	}
	log.Printf("milter_trace_runstage stage=%d all_done elapsed_ms=%d", stage, time.Since(t0).Milliseconds())
	return results
}
