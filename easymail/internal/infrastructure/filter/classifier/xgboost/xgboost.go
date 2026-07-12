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

// Package xgboost is reserved for future XGBoost-based classify models (training + inference).
package xgboost

import (
	"context"
	"errors"

	"easymail/internal/domain/filter/classifier"
)

// Predictor is a placeholder until XGBoost integration lands.
type Predictor struct{}

// NewPredictor returns a stub predictor.
func NewPredictor() *Predictor {
	return &Predictor{}
}

func (p *Predictor) Algorithm() classifier.Algorithm {
	return classifier.AlgorithmXGBoost
}

func (p *Predictor) Open(ctx context.Context, spec classifier.ModelRuntime) error {
	_, _ = ctx, spec
	return errors.New("xgboost: not implemented")
}

func (p *Predictor) Predict(ctx context.Context, in classifier.PredictorInput) (classifier.Prediction, error) {
	_, _ = ctx, in
	return classifier.Prediction{}, errors.New("xgboost: not implemented")
}

func (p *Predictor) Close() error {
	return nil
}


