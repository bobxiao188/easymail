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

package fasttext

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"easymail/internal/domain/filter/classifier"
)

// Predictor is a concrete FastText classifier engine (pure Go loader for .bin/.ftz models).
type Predictor struct {
	mu           sync.Mutex
	m            *Model
	maxTextRunes int // from ModelRuntime.MaxTextLength at Open
}

// NewPredictor returns an unloaded FastText predictor.
func NewPredictor() *Predictor {
	return &Predictor{}
}

func (p *Predictor) Algorithm() classifier.Algorithm {
	return classifier.AlgorithmFastText
}

func (p *Predictor) Open(ctx context.Context, spec classifier.ModelRuntime) error {
	_ = ctx
	path := strings.TrimSpace(spec.SavePath)
	if path == "" {
		return errors.New("fasttext: empty save_path")
	}
	m, err := LoadModel(path)
	if err != nil {
		return fmt.Errorf("fasttext: load %q: %w", path, err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.m != nil {
		_ = p.m.Close()
	}
	p.m = m
	p.maxTextRunes = spec.MaxTextLength
	return nil
}

func (p *Predictor) Predict(ctx context.Context, in classifier.PredictorInput) (classifier.Prediction, error) {
	if err := ctx.Err(); err != nil {
		return classifier.Prediction{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	m := p.m
	if m == nil {
		return classifier.Prediction{}, errors.New("fasttext: predictor not opened")
	}
	maxRunes := p.maxTextRunes
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return classifier.Prediction{}, errors.New("fasttext: empty text")
	}
	if maxRunes > 0 && utf8.RuneCountInString(text) > maxRunes {
		text = truncateRunes(text, maxRunes)
	}
	words := WordsForInference(text)
	if len(words) == 0 {
		return classifier.Prediction{}, errors.New("fasttext: no tokens after segmentation")
	}
	k := m.GetNLabels()
	if k <= 0 {
		return classifier.Prediction{}, errors.New("fasttext: model has no labels")
	}
	raw, err := m.Predict(words, k, 0)
	if err != nil {
		return classifier.Prediction{}, err
	}
	if len(raw) == 0 {
		return classifier.Prediction{}, errors.New("fasttext: empty prediction (OOV or unsupported loss)")
	}
	dist := make([]classifier.LabelScore, 0, len(raw))
	for i := range raw {
		lab := normalizeFastTextClassLabel(raw[i].Label)
		if lab == "" {
			continue
		}
		dist = append(dist, classifier.LabelScore{Label: lab, Probability: raw[i].Prob})
	}
	if len(dist) == 0 {
		return classifier.Prediction{}, errors.New("fasttext: no usable labels in output")
	}
	top := dist[0]
	return classifier.Prediction{
		TopLabel:       top.Label,
		TopProbability: top.Probability,
		Distribution:   dist,
	}, nil
}

func (p *Predictor) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.m != nil {
		_ = p.m.Close()
		p.m = nil
	}
	p.maxTextRunes = 0
	return nil
}

func normalizeFastTextClassLabel(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "__label__")
	return strings.TrimSpace(s)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}



