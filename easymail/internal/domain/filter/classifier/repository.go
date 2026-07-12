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

package classifier

import "context"

// ModelConfigRepository loads enabled model definitions for the classifier registry / pools.
type ModelConfigRepository interface {
	AllModels(ctx context.Context) ([]Model, error)
}

// ModelAssetRepository checks on-disk model asset availability.
type ModelAssetRepository interface {
	AssetsReady(m Model) bool
	DistilBERTAssetPaths(m Model) (string, error)
}

// ModelRepository is the full CRUD port for classify model definitions.
type ModelRepository interface {
	GetByID(ctx context.Context, id int64) (*Model, error)
	List(ctx context.Context, keyword, algorithm string, status *int, page, pageSize int) ([]Model, int64, error)
	Create(ctx context.Context, m *Model) error
	Update(ctx context.Context, m *Model) error
	Delete(ctx context.Context, id int64) error
}

// SampleRepository is the CRUD port for model training samples.
type SampleRepository interface {
	List(ctx context.Context, modelID int64, keyword, labelFilter string, page, pageSize int) ([]Sample, int64, error)
	ListLabels(ctx context.Context, modelID int64) ([]string, error)
	Create(ctx context.Context, samples []Sample) error
	Update(ctx context.Context, sample *Sample) error
	Delete(ctx context.Context, modelID, sampleID int64) error
	ExportTrainTxt(ctx context.Context, modelID int64) ([]byte, error)
}
