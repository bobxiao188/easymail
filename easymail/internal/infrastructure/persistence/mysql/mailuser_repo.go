package mysql

import (
	"context"
	"errors"
	"time"

	"easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MailUserPO is the persistence object for MailUser.
type MailUserPO struct {
	ID               shared.GlobalID `gorm:"primaryKey;type:varchar(36);not null"`
	DomainID         shared.GlobalID `gorm:"type:varchar(36);not null;index"`
	Username         string          `gorm:"size:255;not null"`
	PasswordHash     string          `gorm:"size:255;not null"`
	Email            string          `gorm:"uniqueIndex;size:128;not null"`
	Active           bool            `gorm:"type:tinyint(1);default(1)"`
	IsDeleted        bool            `gorm:"type:tinyint(1);default(0)"`
	StorageQuota     int64           `gorm:"default:0"`
	DataPath         string          `gorm:"size:512;not null;default:\"\""`
	StorageID        int             `gorm:"default:0"`
	PasswordExpireAt time.Time       `gorm:"type:timestamp"`
	CreateTime       time.Time       `gorm:"autoCreateTime"`
	UpdateTime       time.Time       `gorm:"autoUpdateTime"`
	DeleteTime       time.Time       `gorm:"type:timestamp"`
}

func (MailUserPO) TableName() string { return "mail_users" }

func poToMailUser(po *MailUserPO) *management.MailUser {
	if po == nil {
		return nil
	}
	return &management.MailUser{
		ID:               po.ID,
		DomainID:         po.DomainID,
		Username:         po.Username,
		PasswordHash:     po.PasswordHash,
		Email:            po.Email,
		Active:           po.Active,
		IsDeleted:        po.IsDeleted,
		StorageQuota:     po.StorageQuota,
		DataPath:         po.DataPath,
		StorageID:        po.StorageID,
		PasswordExpireAt: po.PasswordExpireAt,
		CreateTime:       po.CreateTime,
		UpdateTime:       po.UpdateTime,
		DeleteTime:       po.DeleteTime,
	}
}

func mailUserToPO(u *management.MailUser) *MailUserPO {
	if u == nil {
		return nil
	}
	return &MailUserPO{
		ID:               u.ID,
		DomainID:         u.DomainID,
		Username:         u.Username,
		PasswordHash:     u.PasswordHash,
		Email:            u.Email,
		Active:           u.Active,
		IsDeleted:        u.IsDeleted,
		StorageQuota:     u.StorageQuota,
		DataPath:         u.DataPath,
		StorageID:        u.StorageID,
		PasswordExpireAt: u.PasswordExpireAt,
		CreateTime:       u.CreateTime,
		UpdateTime:       u.UpdateTime,
		DeleteTime:       u.DeleteTime,
	}
}

type MailUserRepository struct {
	db persistence.DBProvider
}

func NewMailUserRepository(db persistence.DBProvider) *MailUserRepository {
	return &MailUserRepository{db: db}
}

func (r *MailUserRepository) Save(ctx context.Context, u *management.MailUser) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}

	// Check for duplicate email (exclude current record by ID).
	var exists int64
	if err := g.Model(&MailUserPO{}).
		Where("email = ? AND id <> ?", u.Email, u.ID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return management.ErrMailUserExists
	}

	po := mailUserToPO(u)
	po.UpdateTime = time.Now()
	return g.Save(po).Error
}

func (r *MailUserRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.MailUser, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po MailUserPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrMailUserNotFound
		}
		return nil, err
	}
	return poToMailUser(&po), nil
}

func (r *MailUserRepository) FindByFullEmail(ctx context.Context, email string) (*management.MailUser, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po MailUserPO
	if err := g.Where("email = ?", email).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrMailUserNotFound
		}
		return nil, err
	}
	return poToMailUser(&po), nil
}

func (r *MailUserRepository) FindByUsername(ctx context.Context, domainID shared.GlobalID, username string) (*management.MailUser, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po MailUserPO
	if err := g.Where("domain_id = ? AND username = ?", domainID, username).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrMailUserNotFound
		}
		return nil, err
	}
	return poToMailUser(&po), nil
}

func (r *MailUserRepository) FindByDomainID(ctx context.Context, domainID shared.GlobalID) ([]management.MailUser, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []MailUserPO
	if err := g.Where("domain_id = ?", domainID).Find(&pos).Error; err != nil {
		return nil, err
	}
	users := make([]management.MailUser, len(pos))
	for i := range pos {
		users[i] = *poToMailUser(&pos[i])
	}
	return users, nil
}

func (r *MailUserRepository) Search(ctx context.Context, domainID shared.GlobalID, keyword string, status int, page, pageSize int) ([]management.MailUser, int64, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}

	// Build base query filtered by domain.
	db := g.Model(&MailUserPO{}).Where("domain_id = ?", domainID).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("username LIKE ? OR email LIKE ?", like, like)
	}
	// Status filter: -1 = all, 0 = inactive, 1 = active
	if status >= 0 {
		db = db.Where("active = ?", status == 1)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := g.Model(&MailUserPO{}).Where("domain_id = ?", domainID).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR email LIKE ?", like, like)
	}
	if status >= 0 {
		q = q.Where("active = ?", status == 1)
	}
	offset := (page - 1) * pageSize
	var pos []MailUserPO
	if err := q.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "username"},
		Desc:   false,
	}).Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	users := make([]management.MailUser, len(pos))
	for i := range pos {
		users[i] = *poToMailUser(&pos[i])
	}
	return users, total, nil
}

func (r *MailUserRepository) SoftDelete(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	now := time.Now()
	return g.Model(&MailUserPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted":  true,
			"active":      false,
			"update_time": now,
			"delete_time": now,
		}).Error
}

func (r *MailUserRepository) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	var po MailUserPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		return err
	}
	return g.Model(&MailUserPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"active":      !po.Active,
			"update_time": time.Now(),
		}).Error
}

func (r *MailUserRepository) UpdatePassword(ctx context.Context, id shared.GlobalID, hash string) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Model(&MailUserPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"password_hash": hash,
			"update_time":   time.Now(),
		}).Error
}

func (r *MailUserRepository) HardDelete(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Where("id = ?", id).Delete(&MailUserPO{}).Error
}

func (r *MailUserRepository) ChangePassword(ctx context.Context, id shared.GlobalID, oldPassword, newPassword string) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	
	// Verify old password first
	var po MailUserPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		return err
	}
	
	// Verify old password
	if err := shared.Verify(po.PasswordHash, oldPassword); err != nil {
		return management.ErrMailUserInvalidPass
	}
	
	// Hash new password
	newHash, err := shared.Hash(newPassword)
	if err != nil {
		return err
	}
	
	// Update password
	return g.Model(&MailUserPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"password_hash": newHash,
			"update_time":   time.Now(),
		}).Error
}
