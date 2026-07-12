package modelcache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/filter/classifier/distilbert"
	"easymail/internal/infrastructure/filter/classifier/fasttext"
	"easymail/internal/infrastructure/filter/classifier/xgboost"
	"golang.org/x/sync/errgroup"
)

const (
	defaultMaxConcurrent = 4
	defaultPerModelTO    = 30 * time.Second
)

// inferenceEngine is the per-algorithm model client (open once, predict many).
type inferenceEngine interface {
	Predict(ctx context.Context, in classifier.PredictorInput) (classifier.Prediction, error)
	Close() error
}

type cachedModel struct {
	id   string
	name string
	eng  inferenceEngine
}

// ModelCache provides lazy-loaded model inference.
// Models are opened on first use and cached until Invalidate is called.
type ModelCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedModel
}

func New() *ModelCache {
	return &ModelCache{
		entries: make(map[string]*cachedModel),
	}
}

// Invalidate closes all cached engines. Call after InvalidateClassifyModelsCache.
func (c *ModelCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.entries {
		_ = e.eng.Close()
	}
	c.entries = make(map[string]*cachedModel)
}

// PredictAll runs inference on all eligible models (enabled + non-deleted + filter match).
func (c *ModelCache) PredictAll(ctx context.Context, in classifier.PredictorInput, include func(classifier.Model) bool) []classifier.Prediction {
	models, err := cache.CachedClassifyModels(ctx, nil)
	if err != nil || len(models) == 0 {
		return nil
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(defaultMaxConcurrent)

	var mu sync.Mutex
	var out []classifier.Prediction

	for i := range models {
		m := models[i]
		if !m.Enabled || m.IsDeleted {
			continue
		}
		if include != nil && !include(m) {
			continue
		}
		g.Go(func() error {
			cm := c.getOrOpen(gctx, m)
			if cm == nil {
				return nil
			}
			cctx, cancel := context.WithTimeout(gctx, defaultPerModelTO)
			defer cancel()
			cp, cpErr := cm.eng.Predict(cctx, in)
			if cpErr != nil {
				cp.Err = cpErr.Error()
			}
			cp.ModelID = cm.id
			cp.ModelName = cm.name
			mu.Lock()
			out = append(out, cp)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return out
}

// PredictForModel runs inference for a single model (admin try-predict).
func (c *ModelCache) PredictForModel(ctx context.Context, in classifier.PredictorInput, m classifier.Model) (classifier.Prediction, error) {
	cm := c.getOrOpen(ctx, m)
	if cm == nil {
		return classifier.Prediction{}, fmt.Errorf("failed to open model %q", m.Name)
	}
	cctx, cancel := context.WithTimeout(ctx, defaultPerModelTO)
	defer cancel()
	cp, cpErr := cm.eng.Predict(cctx, in)
	if cpErr != nil {
		cp.Err = cpErr.Error()
	}
	cp.ModelID = cm.id
	cp.ModelName = cm.name
	return cp, nil
}

func (c *ModelCache) getOrOpen(ctx context.Context, m classifier.Model) *cachedModel {
	id := strconv.FormatUint(uint64(m.ID), 10)

	c.mu.RLock()
	e := c.entries[id]
	c.mu.RUnlock()
	if e != nil {
		return e
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if e = c.entries[id]; e != nil {
		return e
	}

	spec := classifier.ModelRuntime{
		ID:            id,
		Name:          strings.TrimSpace(m.Name),
		Algorithm:     m.Algorithm,
		SavePath:      strings.TrimSpace(m.SavePath),
		MaxTextLength: m.MaxTextLength,
		Params:        m.Params,
	}
	eng, err := openEngine(ctx, spec)
	if err != nil {
		return nil
	}
	e = &cachedModel{id: id, name: strings.TrimSpace(m.Name), eng: eng}
	c.entries[id] = e
	return e
}

func openEngine(ctx context.Context, spec classifier.ModelRuntime) (inferenceEngine, error) {
	switch spec.Algorithm {
	case classifier.AlgorithmFastText:
		p := fasttext.NewPredictor()
		if err := p.Open(ctx, spec); err != nil {
			return nil, err
		}
		return p, nil
	case classifier.AlgorithmDistilBERT:
		p := distilbert.NewPredictor()
		if err := p.Open(ctx, spec); err != nil {
			return nil, err
		}
		return p, nil
	case classifier.AlgorithmXGBoost:
		p := xgboost.NewPredictor()
		if err := p.Open(ctx, spec); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", spec.Algorithm)
	}
}


// Global milter-process cache (single instance, set by milter launcher).
var milterCache *ModelCache

// SetMilterCache registers the milter-process model cache.
func SetMilterCache(c *ModelCache) {
	milterCache = c
}

// MilterCache returns the milter-process model cache (nil if not set).
func MilterCache() *ModelCache {
	return milterCache
}
