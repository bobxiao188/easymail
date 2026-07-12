package filter

import (
	"context"
	"testing"
	"time"

	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/filter/extractors"
)

func TestRunStageExtractors_NilContext(t *testing.T) {
	old := rule.TestOnlySwapRegistry(nil)
	defer rule.TestOnlySwapRegistry(old)

	rule.Register(slowExtractor{stage: filter.StageConnect, key: "a", sleep: 10 * time.Millisecond})
	fc := filter.NewMilterContext()
	res := extractors.RunStage(context.Background(), filter.StageConnect, fc, 1*time.Second)
	if len(res) == 0 {
		t.Fatal("expected result from extractor")
	}
}

func TestRunStageExtractors_EmptyRegistry(t *testing.T) {
	old := rule.TestOnlySwapRegistry(nil)
	defer rule.TestOnlySwapRegistry(old)

	fc := filter.NewMilterContext()
	res := extractors.RunStage(context.Background(), filter.StageConnect, fc, 1*time.Second)
	if len(res) != 0 {
		t.Fatalf("expected empty result, got %v", res)
	}
}

func TestRunStageExtractors_CancelContext(t *testing.T) {
	old := rule.TestOnlySwapRegistry(nil)
	defer rule.TestOnlySwapRegistry(old)

	rule.Register(slowExtractor{stage: filter.StageConnect, key: "hang", sleep: 5 * time.Second})
	fc := filter.NewMilterContext()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	res := extractors.RunStage(ctx, filter.StageConnect, fc, 10*time.Second)
	if len(res) > 0 {
		t.Logf("got partial results: %v", res)
	}
}
