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
	"strings"
	"time"
	"unicode/utf8"

	filtermodelv1 "easymail/internal/api/filtermodel/v1"
	"easymail/internal/domain/filter/classifier"
	modelcache "easymail/internal/infrastructure/filter/classifier/modelcache"
	"easymail/pkg/logger/easylog"
)

// GRPCService implements FilterModelServiceServer (classify-model Infer).
type GRPCService struct {
	filtermodelv1.UnimplementedFilterModelServiceServer
	mc            *modelcache.ModelCache
	log            *easylog.Logger
	maxConcurrent  int
	inferTimeout   time.Duration
}

// NewGRPCService builds a server-side handler; Infer delegates parallel prediction to ModelPool.PredictAll.
func NewGRPCService(log *easylog.Logger, mc *modelcache.ModelCache, maxConcurrent int, inferTimeout time.Duration) *GRPCService {
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	if inferTimeout <= 0 {
		inferTimeout = 30 * time.Second
	}
	return &GRPCService{
		mc:            mc,
		log:           log,
		maxConcurrent: maxConcurrent,
		inferTimeout:  inferTimeout,
	}
}

func (s *GRPCService) svcInfof(format string, args ...interface{}) {
	if s != nil && s.log != nil {
		s.log.Infof("[classify_model_service] "+format, args...)
	}
}

func (s *GRPCService) svcWarnf(format string, args ...interface{}) {
	if s != nil && s.log != nil {
		s.log.Warnf("[classify_model_service] "+format, args...)
	}
}

// Infer runs every predictor in the pool (parallelism bounded by maxConcurrent).
func (s *GRPCService) Infer(ctx context.Context, req *filtermodelv1.InferRequest) (*filtermodelv1.InferResponse, error) {
	if req == nil {
		s.svcInfof("Infer received nil request")
		return &filtermodelv1.InferResponse{}, nil
	}
	langs := req.GetLanguageCodes()
	in := classifier.BuildPredictorInput(req.GetText(), langs)
	textRunes := utf8.RuneCountInString(in.Text)

	preds := s.mc.PredictAll(ctx, in, nil)
	s.svcInfof("Infer text_runes=%d lang_codes=%v pool_models=%d", textRunes, langs, len(preds))

	if len(preds) == 0 {
		s.svcInfof("Infer pool empty (admin: enable non-FastText models with valid assets, or wait for pool_refresh)")
		return &filtermodelv1.InferResponse{}, nil
	}

	out := make([]*filtermodelv1.ModelPrediction, 0, len(preds))
	okN, errN := 0, 0
	for _, cp := range preds {
		if strings.TrimSpace(cp.Err) != "" {
			s.svcWarnf("model id=%s name=%q infer_err=%v", cp.ModelID, cp.ModelName, cp.Err)
			errN++
		} else {
			okN++
		}
		out = append(out, PredictionToProto(cp))
	}
	s.svcInfof("Infer done predictions=%d ok=%d err=%d", len(out), okN, errN)
	return &filtermodelv1.InferResponse{Predictions: out}, nil
}

// PredictionToProto converts domain Prediction to protobuf ModelPrediction.
func PredictionToProto(p classifier.Prediction) *filtermodelv1.ModelPrediction {
	out := &filtermodelv1.ModelPrediction{
		ModelId:     p.ModelID,
		ModelName:   p.ModelName,
		Label:       p.TopLabel,
		Probability: p.TopProbability,
		Error:       p.Err,
	}
	for _, ls := range p.Distribution {
		out.LabelProbabilities = append(out.LabelProbabilities, &filtermodelv1.LabelProbability{
			Label:       ls.Label,
			Probability: ls.Probability,
		})
	}
	return out
}
