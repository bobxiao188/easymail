// Package migrate provides database schema auto-migration.
package migrate

import (
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

// AutoMigrate registers persistent models with GORM.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&mysql.AdminUserPO{},
		&mysql.MailDomainPO{},
		&mysql.MailUserPO{},

		&mysql.BuiltinFeaturePO{},
		&mysql.CustomFeaturePO{},
		&mysql.RulePO{},
		&mysql.FilterLogPO{},
		&mysql.FilterMailStatsDailyPO{},
		&mysql.FilterStatsRollupWatermarkPO{},
		&mysql.ClassifyModelPO{},
		&mysql.ModelSamplePO{},
		&mysql.PublicSampleCategoryPO{},
		&mysql.PublicSamplePO{},
		&mysql.TrainingTaskPO{},
		&mysql.UserSettingsPO{},

		// Postfix configuration management
		&mysql.PostfixAgentPO{},
		&mysql.PostfixConfigPO{},
		&mysql.PostfixDeliveryLogPO{},
	)
}
