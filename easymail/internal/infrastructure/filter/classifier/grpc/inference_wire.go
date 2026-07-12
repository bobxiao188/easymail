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

package grpc

import (
	"strings"

	filtermodelv1 "easymail/internal/api/filtermodel/v1"
	"easymail/internal/domain/filter/classifier"
)

// ClassifyPredictionFromProto maps gRPC ModelPrediction into the domain result (milter client path).
func PredictionFromProto(p *filtermodelv1.ModelPrediction) classifier.Prediction {
	if p == nil {
		return classifier.Prediction{}
	}
	out := classifier.Prediction{
		ModelID:        strings.TrimSpace(p.GetModelId()),
		ModelName:      strings.TrimSpace(p.GetModelName()),
		TopLabel:       strings.TrimSpace(p.GetLabel()),
		TopProbability: p.GetProbability(),
		Err:            strings.TrimSpace(p.GetError()),
	}
	for _, lp := range p.GetLabelProbabilities() {
		if lp == nil {
			continue
		}
		out.Distribution = append(out.Distribution, classifier.LabelScore{
			Label:       strings.TrimSpace(lp.GetLabel()),
			Probability: lp.GetProbability(),
		})
	}
	return out
}


