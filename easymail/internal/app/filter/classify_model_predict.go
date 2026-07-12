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

	"easymail/internal/domain/filter/classifier"
	modelcache "easymail/internal/infrastructure/filter/classifier/modelcache"
)

func (s *classifyModelService) Predict(ctx context.Context, id int64, text string, languageCodes []string) (classifier.Prediction, error) {
	if strings.TrimSpace(text) == "" {
		return classifier.Prediction{}, ErrClassifyModelPredictEmptyText
	}
	m, err := s.GetByID(ctx, id)
	if err != nil {
		return classifier.Prediction{}, err
	}
	if !classifyModelAssetsReady(m) {
		return classifier.Prediction{}, ErrClassifyModelActivationNotReady
	}
	mc := modelcache.MilterCache()
	if mc == nil {
		mc = modelcache.New()
	}
	in := classifier.BuildPredictorInput(text, languageCodes)
	out, err := mc.PredictForModel(ctx, in, *m)
	if err != nil {
		return classifier.Prediction{}, err
	}
	return out, nil
}
