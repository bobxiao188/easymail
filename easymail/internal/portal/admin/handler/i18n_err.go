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

package handler

import (
	"errors"

	appAdminEx "easymail/internal/app/admin/exception"
	filtersvc "easymail/internal/app/filter"
	"easymail/internal/domain/management"
	appi18n "easymail/pkg/i18n"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func messageMailAccountOp(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, management.ErrMailUserInvalidEmail):
		return appi18n.Message(c, appi18n.KeyErrInvalidCreateAccount)
	case errors.Is(err, management.ErrMailUserNotFound):
		return appi18n.Message(c, appi18n.KeyErrInvalidID)
	case errors.Is(err, management.ErrMailUserDomainInvalid):
		return appi18n.Message(c, appi18n.KeyErrInvalidDomainUsername)
	case errors.Is(err, management.ErrMailUserInvalidPass):
		return appi18n.Message(c, appi18n.KeyErrInvalidAccountPassword)
	case errors.Is(err, management.ErrMailUserNotDeleted):
		return appi18n.Message(c, appi18n.KeyErrMailUserNotDeleted)
	case errors.Is(err, management.ErrMailUserExists):
		return appi18n.Message(c, appi18n.KeyErrMailUserAlreadyExists)
	default:
		// Return the actual error text so frontend can display it.
		return err.Error()
	}
}

func messageMailDomainOp(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, management.ErrDomainNotDeleted):
		return appi18n.Message(c, appi18n.KeyErrDomainNotDeleted)
	case errors.Is(err, management.ErrDomainExists):
		return appi18n.Message(c, appi18n.KeyErrDomainAlreadyExists)
	default:
		return err.Error()
	}
}

func messageSpamModelOp(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return appi18n.Message(c, appi18n.KeyErrNotFoundClassifyModel)
	case errors.Is(err, filtersvc.ErrClassifyModelNameInvalidFeatureKey):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelNameInvalidFeatureKey)
	case errors.Is(err, filtersvc.ErrClassifyModelNameConflictsBuiltin):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelNameConflictsBuiltin)
	case errors.Is(err, filtersvc.ErrClassifyModelNameConflictsCustom):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelNameConflictsCustom)
	case errors.Is(err, filtersvc.ErrClassifyModelNameDuplicate):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelNameDuplicate)
	case errors.Is(err, filtersvc.ErrModelSampleNotFound):
		return appi18n.Message(c, appi18n.KeyErrNotFoundModelSample)
	case errors.Is(err, filtersvc.ErrModelSampleInvalid):
		return appi18n.Message(c, appi18n.KeyErrModelSampleInvalid)
	case errors.Is(err, filtersvc.ErrModelSampleBatchTooLarge):
		return appi18n.Message(c, appi18n.KeyErrModelSampleBatchTooLarge)
	case errors.Is(err, filtersvc.ErrClassifyModelRemoveFiles):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelRemoveFiles)
	case errors.Is(err, filtersvc.ErrClassifyModelActivationNotReady):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelActivationNotReady)
	case errors.Is(err, filtersvc.ErrFastTextExecutableNotConfigured):
		return appi18n.Message(c, appi18n.KeyErrFastTextExecutableNotConfigured)
	case errors.Is(err, filtersvc.ErrClassifyModelModelRootNotConfigured):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelModelRootNotConfigured)
	case errors.Is(err, filtersvc.ErrClassifyModelTrainNotFastText):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelTrainNotFastText)
	case errors.Is(err, filtersvc.ErrClassifyModelTrainAlreadyRunning):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelTrainAlreadyRunning)
	case errors.Is(err, filtersvc.ErrClassifyModelTrainNoSamples):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelTrainNoSamples)
	case errors.Is(err, filtersvc.ErrClassifyModelPredictEmptyText):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelPredictTextRequired)
	case errors.Is(err, filtersvc.ErrClassifyModelOnnxRequired):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelOnnxRequired)
	case errors.Is(err, filtersvc.ErrClassifyModelOnnxInvalidExt):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelOnnxInvalidExt)
	case errors.Is(err, filtersvc.ErrClassifyModelOnnxWriteFailed):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelOnnxSaveFailed)
	case errors.Is(err, filtersvc.ErrClassifyModelDirRenameFailed):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelDirRenameFailed)
	case errors.Is(err, filtersvc.ErrClassifyModelNotDistilBERT):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelNotDistilBERT)
	case errors.Is(err, filtersvc.ErrClassifyModelOnnxLabelsParse):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelOnnxLabelsParse)
	case errors.Is(err, filtersvc.ErrClassifyModelExportNoBin):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelExportNoBin)
	case errors.Is(err, filtersvc.ErrClassifyModelImportInvalidZip):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelImportInvalidZip)
	case errors.Is(err, filtersvc.ErrClassifyModelImportMissingConf):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelImportMissingConf)
	case errors.Is(err, filtersvc.ErrClassifyModelImportMissingBin):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelImportMissingBin)
	case errors.Is(err, filtersvc.ErrClassifyModelImportConfParse):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelImportConfParse)
	case errors.Is(err, filtersvc.ErrClassifyModelImportNameConflict):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelImportNameConflict)
	case errors.Is(err, filtersvc.ErrClassifyModelImportWriteFailed):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelImportWriteFailed)
	case errors.Is(err, filtersvc.ErrClassifyModelImportAlgorithmMismatch):
		return appi18n.Message(c, appi18n.KeyErrClassifyModelImportAlgorithmMismatch)
	default:
		return appi18n.Message(c, appi18n.KeyErrOperationFailed)
	}
}

func messagePublicSampleOp(c *gin.Context, err error) string {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return appi18n.Message(c, appi18n.KeyErrNotFoundRecord)
	}
	return appi18n.Message(c, appi18n.KeyErrOperationFailed)
}

func messageTrainingOp(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, filtersvc.ErrTrainingModelNameRequired):
		return appi18n.Message(c, appi18n.KeyErrTrainingModelNameRequired)
	case errors.Is(err, filtersvc.ErrTrainingMinClasses):
		return appi18n.Message(c, appi18n.KeyErrTrainingMinClasses)
	case errors.Is(err, filtersvc.ErrTrainingClassNameInvalid):
		return appi18n.Message(c, appi18n.KeyErrTrainingClassNameInvalid)
	case errors.Is(err, filtersvc.ErrTrainingClassNameDuplicate):
		return appi18n.Message(c, appi18n.KeyErrTrainingClassNameDuplicate)
	case errors.Is(err, filtersvc.ErrTrainingNoTags):
		return appi18n.Message(c, appi18n.KeyErrTrainingNoTags)
	case errors.Is(err, gorm.ErrRecordNotFound):
		return appi18n.Message(c, appi18n.KeyErrNotFoundRecord)
	default:
		return messageSpamModelOp(c, err)
	}
}

func messageAdminChangePassword(c *gin.Context, err error) string {
	switch {
	case errors.Is(err, appAdminEx.ErrInvalidCredentials):
		return appi18n.Message(c, appi18n.KeyAuthInvalidCredentials)
	case errors.Is(err, appAdminEx.ErrUserInactive):
		return appi18n.Message(c, appi18n.KeyAuthAccountInactive)
	case errors.Is(err, appAdminEx.ErrPasswordTooShort):
		return appi18n.Message(c, appi18n.KeyErrPasswordMinLen)
	case errors.Is(err, appAdminEx.ErrNewPasswordMustDiffer):
		return appi18n.Message(c, appi18n.KeyErrNewPasswordMustDiffer)
	default:
		return appi18n.Message(c, appi18n.KeyErrOperationFailed)
	}
}

func messageAdminInactiveOrOpFailed(c *gin.Context, err error) string {
	if errors.Is(err, appAdminEx.ErrUserInactive) {
		return appi18n.Message(c, appi18n.KeyAuthAccountInactive)
	}
	return appi18n.Message(c, appi18n.KeyErrOperationFailed)
}

func messageInternalError(c *gin.Context) string {
	return appi18n.Message(c, appi18n.KeyErrInternalServer)
}
