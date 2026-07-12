package migrate

import (
	"time"

	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminBootstrap seeds the default admin account (admin / admin888)
// into the database. Safe to run repeatedly — uses OnConflict DoNothing
// so it only inserts when no admin user exists yet.
func AdminBootstrap(db *gorm.DB) error {
	hash, err := shared.Hash("admin888")
	if err != nil {
		return err
	}
	now := time.Now()
	admin := mysql.AdminUserPO{
		ID:           shared.GlobalID("00000000-0000-0000-0000-000000000001"),
		Username:     "admin",
		PasswordHash: hash,
		Nickname:     "admin",
		Email:        "admin@example.com",
		Language:     "zh",
		Skin:         "dark",
		Active:       true,
		CreateTime:   now,
		UpdateTime:   now,
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "username"}},
		DoNothing: true,
	}).Create(&admin).Error
}