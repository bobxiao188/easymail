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

package distilbert

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/filter/assets"
	inframodel "easymail/internal/infrastructure/model"
)

// Predictor is a concrete DistilBERT ONNX classifier engine.
type Predictor struct {
	eng *inframodel.DistilBERTONNXEngine
}

// NewPredictor returns an unloaded predictor instance (pool will call Open).
func NewPredictor() *Predictor {
	return &Predictor{}
}

func (p *Predictor) Algorithm() classifier.Algorithm {
	return classifier.AlgorithmDistilBERT
}

func (p *Predictor) Open(ctx context.Context, spec classifier.ModelRuntime) error {
	_ = ctx
	layout := classifier.Model{SavePath: spec.SavePath, Params: spec.Params}
	onnxPath, err := assets.DistilBERTAssetPaths(layout)
	if err != nil {
		return fmt.Errorf("distilbert paths: %w", err)
	}
	if err := ensureONNXRuntime(); err != nil {
		return fmt.Errorf("distilbert onnx init: %w", err)
	}
	eng, err := getOrCreateBERTEngine(onnxPath, spec.MaxTextLength)
	if err != nil {
		return fmt.Errorf("distilbert load: %w", err)
	}
	p.eng = eng
	return nil
}

func (p *Predictor) Predict(ctx context.Context, in classifier.PredictorInput) (classifier.Prediction, error) {
	if p == nil || p.eng == nil {
		return classifier.Prediction{}, errors.New("distilbert: not opened or nil engine")
	}
	label, prob, labs, probs, err := predictDistilBERTWithEngine(ctx, p.eng, strings.TrimSpace(in.Text))
	if err != nil {
		return classifier.Prediction{}, err
	}
	dist := make([]classifier.LabelScore, 0, len(labs))
	for i := range labs {
		dist = append(dist, classifier.LabelScore{Label: labs[i], Probability: probs[i]})
	}
	return classifier.Prediction{
		TopLabel:       label,
		TopProbability: prob,
		Distribution:   dist,
	}, nil
}

func (p *Predictor) Close() error {
	p.eng = nil
	return nil
}

var bertEngineCache sync.Map // string -> *inframodel.DistilBERTONNXEngine

func bertEngineCacheKey(onnxPath string, seqLen int) string {
	return fmt.Sprintf("%s|%d", onnxPath, seqLen)
}

func clampSeqLen(n int) int {
	if n <= 0 {
		return 128
	}
	if n < 8 {
		return 8
	}
	if n > 512 {
		return 512
	}
	return n
}

func getOrCreateBERTEngine(onnxPath string, maxTextLen int) (*inframodel.DistilBERTONNXEngine, error) {
	seqLen := clampSeqLen(maxTextLen)
	key := bertEngineCacheKey(onnxPath, seqLen)
	if v, ok := bertEngineCache.Load(key); ok {
		return v.(*inframodel.DistilBERTONNXEngine), nil
	}
	eng, err := inframodel.NewDistilBERTONNXEngine(onnxPath, seqLen)
	if err != nil {
		return nil, err
	}
	if actual, loaded := bertEngineCache.LoadOrStore(key, eng); loaded {
		eng.Close()
		return actual.(*inframodel.DistilBERTONNXEngine), nil
	}
	return eng, nil
}

func argmaxFloat64(p []float64) int {
	if len(p) == 0 {
		return 0
	}
	best := 0
	for i := 1; i < len(p); i++ {
		if p[i] > p[best] {
			best = i
		}
	}
	return best
}

func predictDistilBERTWithEngine(ctx context.Context, eng *inframodel.DistilBERTONNXEngine, text string) (label string, prob float64, allLabels []string, allProbs []float64, err error) {
	if eng == nil {
		return "", 0, nil, nil, errors.New("nil engine")
	}
	done := make(chan struct{})
	var predLabel string
	var predProb float64
	var labs []string
	var probs []float64
	var predErr error
	go func() {
		defer close(done)
		labs, probs, predErr = eng.PredictProbs(text)
		if predErr != nil {
			return
		}
		idx := argmaxFloat64(probs)
		predLabel = labs[idx]
		predProb = probs[idx]
	}()
	select {
	case <-ctx.Done():
		return "", 0, nil, nil, ctx.Err()
	case <-done:
		if predErr != nil {
			return "", 0, nil, nil, predErr
		}
		outLabs := make([]string, len(labs))
		copy(outLabs, labs)
		outProbs := make([]float64, len(probs))
		copy(outProbs, probs)
		return predLabel, predProb, outLabs, outProbs, nil
	}
}



