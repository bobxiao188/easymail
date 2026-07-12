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

	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

// PublicSampleService manages global training samples.
type PublicSampleService interface {
	ListSamples(ctx context.Context, categoryID uint, tag, keyword string, page, pageSize int) ([]mysql.PublicSampleRow, int64, error)
	ListTags(ctx context.Context, categoryID uint) ([]string, error)
	CreateSample(ctx context.Context, categoryID uint, tag, text string) error
	CreateSamplesBatch(ctx context.Context, items []mysql.PublicSamplePO) error
	UpdateSample(ctx context.Context, id uint, categoryID uint, tag, text string) error
	DeleteSample(ctx context.Context, id uint) error
	DeleteSamplesBatch(ctx context.Context, ids []uint) error
	UpdateSamplesBatch(ctx context.Context, ids []uint, categoryID uint, tag string) error
	DescribeSamples(ctx context.Context, categoryID uint) ([]struct {
		CategoryID uint   `json:"categoryId"`
		Category   string `json:"category"`
		Tag        string `json:"tag"`
		Count      int64  `json:"count"`
	}, error)
}

// PublicSampleCategoryService manages sample categories.
type PublicSampleCategoryService interface {
	ListCategories(ctx context.Context, keyword string, page, pageSize int) ([]mysql.CategoryRow, int64, error)
	GetCategory(ctx context.Context, id uint) (*mysql.CategoryRow, error)
	CreateCategory(ctx context.Context, name, description string) (*mysql.CategoryRow, error)
	UpdateCategory(ctx context.Context, id uint, name, description string) (*mysql.CategoryRow, error)
	DeleteCategory(ctx context.Context, id uint) error
	SyncSampleCounts(ctx context.Context) error
}

type publicSampleService struct {
	db           *gorm.DB
	sampleRepo   *mysql.PublicSampleRepository
	categoryRepo *mysql.PublicSampleCategoryRepository
}

func NewPublicSampleService(db *gorm.DB) (PublicSampleService, PublicSampleCategoryService) {
	svc := &publicSampleService{
		db:           db,
		sampleRepo:   mysql.NewPublicSampleRepository(db),
		categoryRepo: mysql.NewPublicSampleCategoryRepository(db),
	}
	return svc, svc
}

func (s *publicSampleService) ListSamples(ctx context.Context, categoryID uint, tag, keyword string, page, pageSize int) ([]mysql.PublicSampleRow, int64, error) {
	return s.sampleRepo.List(ctx, categoryID, tag, keyword, page, pageSize)
}

func (s *publicSampleService) ListTags(ctx context.Context, categoryID uint) ([]string, error) {
	return s.sampleRepo.ListTags(ctx, categoryID)
}

func (s *publicSampleService) CreateSample(ctx context.Context, categoryID uint, tag, text string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.sampleRepo.Create(ctx, categoryID, tag, text); err != nil {
			return err
		}
		// Increment sample count for the category
		return tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", categoryID).Update("sample_count", gorm.Expr("sample_count + ?", 1)).Error
	})
}

func (s *publicSampleService) CreateSamplesBatch(ctx context.Context, items []mysql.PublicSamplePO) error {
	if len(items) == 0 {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.sampleRepo.CreateBatch(ctx, items); err != nil {
			return err
		}
		// Group by category_id and update counts
		categoryCounts := make(map[uint]int)
		for _, item := range items {
			categoryCounts[item.CategoryID]++
		}
		for categoryId, count := range categoryCounts {
			if err := tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", categoryId).Update("sample_count", gorm.Expr("sample_count + ?", count)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *publicSampleService) UpdateSample(ctx context.Context, id uint, categoryID uint, tag, text string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get the sample to find its current category_id
		var sample mysql.PublicSamplePO
		if err := tx.Where("id = ?", id).First(&sample).Error; err != nil {
			return err
		}
		oldCategoryID := sample.CategoryID

		// Update the sample
		if err := s.sampleRepo.Update(ctx, id, categoryID, tag, text); err != nil {
			return err
		}

		// If category changed, update both categories' counts
		if categoryID > 0 && categoryID != oldCategoryID {
			// Decrement old category
			if err := tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", oldCategoryID).Update("sample_count", gorm.Expr("sample_count - ?", 1)).Error; err != nil {
				return err
			}
			// Increment new category
			if err := tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", categoryID).Update("sample_count", gorm.Expr("sample_count + ?", 1)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *publicSampleService) DeleteSample(ctx context.Context, id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Get the sample to find its category_id
		var sample mysql.PublicSamplePO
		if err := tx.Where("id = ?", id).First(&sample).Error; err != nil {
			return err
		}
		// Delete the sample
		if err := s.sampleRepo.Delete(ctx, id); err != nil {
			return err
		}
		// Decrement sample count for the category
		return tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", sample.CategoryID).Update("sample_count", gorm.Expr("sample_count - ?", 1)).Error
	})
}

func (s *publicSampleService) DeleteSamplesBatch(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Query samples to get their category_ids
		var samples []mysql.PublicSamplePO
		if err := tx.Where("id IN ?", ids).Find(&samples).Error; err != nil {
			return err
		}
		if len(samples) == 0 {
			return nil
		}

		// Group by category_id to track count changes
		categoryCounts := make(map[uint]int)
		for _, sample := range samples {
			categoryCounts[sample.CategoryID]++
		}

		// Delete samples
		if err := s.sampleRepo.DeleteBatch(ctx, ids); err != nil {
			return err
		}

		// Decrement sample count for each affected category
		for categoryId, count := range categoryCounts {
			if err := tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", categoryId).Update("sample_count", gorm.Expr("sample_count - ?", count)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *publicSampleService) UpdateSamplesBatch(ctx context.Context, ids []uint, categoryID uint, tag string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Query samples to get their current category_ids
		var samples []mysql.PublicSamplePO
		if err := tx.Where("id IN ?", ids).Find(&samples).Error; err != nil {
			return err
		}
		if len(samples) == 0 {
			return nil
		}

		// Group by old category_id to track count changes
		oldCategoryCounts := make(map[uint]int)
		for _, sample := range samples {
			oldCategoryCounts[sample.CategoryID]++
		}

		// Update samples
		if err := s.sampleRepo.UpdateBatch(ctx, ids, categoryID, tag); err != nil {
			return err
		}

		// If category changed, adjust counts
		if categoryID > 0 {
			// Decrement old categories
			for oldCatId, count := range oldCategoryCounts {
				if oldCatId != categoryID {
					if err := tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", oldCatId).Update("sample_count", gorm.Expr("sample_count - ?", count)).Error; err != nil {
						return err
					}
				}
			}
			// Increment new category
			totalMoved := 0
			for oldCatId, count := range oldCategoryCounts {
				if oldCatId != categoryID {
					totalMoved += count
				}
			}
			if totalMoved > 0 {
				if err := tx.Model(&mysql.PublicSampleCategoryPO{}).Where("id = ?", categoryID).Update("sample_count", gorm.Expr("sample_count + ?", totalMoved)).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *publicSampleService) DescribeSamples(ctx context.Context, categoryID uint) ([]struct {
	CategoryID uint   `json:"categoryId"`
	Category   string `json:"category"`
	Tag        string `json:"tag"`
	Count      int64  `json:"count"`
}, error) {
	return s.sampleRepo.DescribeSamples(ctx, categoryID)
}

// Category management methods
func (s *publicSampleService) ListCategories(ctx context.Context, keyword string, page, pageSize int) ([]mysql.CategoryRow, int64, error) {
	return s.categoryRepo.List(ctx, keyword, page, pageSize)
}

func (s *publicSampleService) GetCategory(ctx context.Context, id uint) (*mysql.CategoryRow, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *publicSampleService) CreateCategory(ctx context.Context, name, description string) (*mysql.CategoryRow, error) {
	return s.categoryRepo.Create(ctx, name, description)
}

func (s *publicSampleService) UpdateCategory(ctx context.Context, id uint, name, description string) (*mysql.CategoryRow, error) {
	return s.categoryRepo.Update(ctx, id, name, description)
}

func (s *publicSampleService) DeleteCategory(ctx context.Context, id uint) error {
	return s.categoryRepo.Delete(ctx, id)
}

func (s *publicSampleService) SyncSampleCounts(ctx context.Context) error {
	return s.categoryRepo.SyncSampleCounts(ctx)
}
