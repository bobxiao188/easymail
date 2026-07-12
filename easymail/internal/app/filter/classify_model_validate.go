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
	"errors"
	"strings"
	"unicode"

	"easymail/internal/domain/filter/classifier"

	"gorm.io/gorm"
)

var (
	ErrClassifyModelNameInvalidFeatureKey = errors.New("classify_model_name_invalid_feature_key")
	ErrClassifyModelNameConflictsBuiltin  = errors.New("classify_model_name_conflicts_builtin_feature")
	ErrClassifyModelNameConflictsCustom   = errors.New("classify_model_name_conflicts_custom_feature")
	ErrClassifyModelNameDuplicate         = errors.New("classify_model_name_duplicate")
	ErrModelSampleNotFound                = errors.New("model_sample_not_found")
	ErrModelSampleInvalid                 = errors.New("model_sample_invalid")
	ErrModelSampleBatchTooLarge           = errors.New("model_sample_batch_too_large")
	ErrClassifyModelRemoveFiles           = errors.New("classify_model_remove_files")
	ErrClassifyModelActivationNotReady    = errors.New("classify_model_activation_not_ready")

	ErrFastTextExecutableNotConfigured     = errors.New("fasttext_executable_not_configured")
	ErrClassifyModelModelRootNotConfigured = errors.New("classify_model_model_root_not_configured")
	ErrClassifyModelTrainNotFastText       = errors.New("classify_model_train_not_fasttext")
	ErrClassifyModelTrainAlreadyRunning    = errors.New("classify_model_train_already_running")
	ErrClassifyModelTrainNoSamples         = errors.New("classify_model_train_no_samples")
	ErrClassifyModelPredictEmptyText       = errors.New("classify_model_predict_empty_text")

	ErrClassifyModelOnnxRequired    = errors.New("classify_model_onnx_required")
	ErrClassifyModelOnnxInvalidExt  = errors.New("classify_model_onnx_invalid_ext")
	ErrClassifyModelOnnxWriteFailed = errors.New("classify_model_onnx_write_failed")
	ErrClassifyModelOnnxLabelsParse = errors.New("classify_model_onnx_labels_parse")
	ErrClassifyModelDirRenameFailed = errors.New("classify_model_dir_rename_failed")
	ErrClassifyModelNotDistilBERT   = errors.New("classify_model_not_distilbert")

	ErrClassifyModelExportNoBin      = errors.New("classify_model_export_no_bin")
	ErrClassifyModelImportInvalidZip = errors.New("classify_model_import_invalid_zip")
	ErrClassifyModelImportMissingConf = errors.New("classify_model_import_missing_conf")
	ErrClassifyModelImportMissingBin  = errors.New("classify_model_import_missing_bin")
	ErrClassifyModelImportConfParse   = errors.New("classify_model_import_conf_parse")
	ErrClassifyModelImportNameConflict = errors.New("classify_model_import_name_conflict")
	ErrClassifyModelImportWriteFailed      = errors.New("classify_model_import_write_failed")
	ErrClassifyModelImportAlgorithmMismatch = errors.New("classify_model_import_algorithm_mismatch")
)

func classifyModelNameHasAlnum(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

// ValidateClassifyModelDisplayName ensures the model name does not collide with builtin/custom feature keys
// or another non-deleted classify model's feature key (same sanitization as runtime).
func ValidateClassifyModelDisplayName(ctx context.Context, db *gorm.DB, displayName string, excludeModelID uint) error {
	trim := strings.TrimSpace(displayName)
	if trim == "" || !classifyModelNameHasAlnum(trim) {
		return ErrClassifyModelNameInvalidFeatureKey
	}
	key := classifier.SanitizeFeatureKey(trim)
	if key == "" {
		return ErrClassifyModelNameInvalidFeatureKey
	}
	builtins, err := mysql.ListFeatureDefs(ctx, db)
	if err != nil {
		return err
	}
	for _, b := range builtins {
		if strings.EqualFold(strings.TrimSpace(b.FeatureKey), key) {
			return ErrClassifyModelNameConflictsBuiltin
		}
	}
	customs, err := mysql.ListCustomFeatureDefs(ctx, db)
	if err != nil {
		return err
	}
	for _, c := range customs {
		if strings.EqualFold(strings.TrimSpace(c.FeatureKey), key) {
			return ErrClassifyModelNameConflictsCustom
		}
	}
	q := db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).Where("is_deleted = ?", false)
	if excludeModelID > 0 {
		q = q.Where("id <> ?", excludeModelID)
	}
	var pos []mysql.ClassifyModelPO
	if err := q.Find(&pos).Error; err != nil {
		return err
	}
	for _, p := range pos {
		if classifier.SanitizeFeatureKey(p.Name) == key {
			return ErrClassifyModelNameDuplicate
		}
	}
	return nil
}

// FeatureKeyReservedByClassifyModel reports whether featureKey equals any non-deleted classify model's feature key.
func FeatureKeyReservedByClassifyModel(ctx context.Context, db *gorm.DB, featureKey string) (bool, error) {
	want := strings.TrimSpace(strings.ToLower(featureKey))
	if want == "" {
		return false, nil
	}
	var pos []mysql.ClassifyModelPO
	if err := db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).Where("is_deleted = ?", false).Find(&pos).Error; err != nil {
		return false, err
	}
	for i := range pos {
		m := mysql.PoToClassifyModel(&pos[i])
		if classifier.SanitizeFeatureKey(m.Name) == want {
			return true, nil
		}
	}
	return false, nil
}
