package extractors

import (
	"context"
	"log"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/feature"
	"easymail/internal/domain/filter/rule"
)

// FeatureEngine is the in-process filter sub-service for feature extraction.
// It runs stage extractors/plugins concurrently with timeout and merges results into Context.
type FeatureEngine struct {
	StageTimeout time.Duration
}

func (fe *FeatureEngine) timeoutOrDefault() time.Duration {
	if fe == nil || fe.StageTimeout <= 0 {
		return 200 * time.Millisecond
	}
	return fe.StageTimeout
}

func (fe *FeatureEngine) Extract(ctx context.Context, stage filter.Stage, fc *filter.MilterContext, timeout time.Duration) []feature.Result {
	if fc == nil {
		return nil
	}
	if timeout <= 0 {
		timeout = fe.timeoutOrDefault()
	}

	// Use a single parent context with the total timeout for all sub-steps.
	// This prevents timeout accumulation where sequential sub-steps each wait the full timeout.
	pctx, pcancel := context.WithTimeout(ctx, timeout)
	defer pcancel()

	// Allocate half the budget to RunStage, the other half to custom features + plugins.
	stageTimeout := timeout / 2
	if stageTimeout < 50*time.Millisecond {
		stageTimeout = 50 * time.Millisecond
	}
	customTimeout := timeout - stageTimeout
	if customTimeout < 50*time.Millisecond {
		customTimeout = 50 * time.Millisecond
	}

	e0 := time.Now()
	res := RunStage(pctx, stage, fc, stageTimeout)
	log.Printf("milter_trace_extract stage=%s runstage elapsed_ms=%d stage_timeout_ms=%d", stage, time.Since(e0).Milliseconds(), stageTimeout.Milliseconds())

	// Apply custom features as early as possible for early-reject.
	{
		ce0 := time.Now()
		_ = applyCustomFeaturesForStage(pctx, stage, fc)
		log.Printf("milter_trace_extract stage=%s custom_features elapsed_ms=%d", stage, time.Since(ce0).Milliseconds())
	}
	if stage == filter.StageBody {
		pe0 := time.Now()
		rule.RunPlugins(pctx, fc)
		log.Printf("milter_trace_extract stage=%s plugins elapsed_ms=%d", stage, time.Since(pe0).Milliseconds())
		_ = applyCustomFeaturesForStage(pctx, stage, fc)
	}
	return res
}
