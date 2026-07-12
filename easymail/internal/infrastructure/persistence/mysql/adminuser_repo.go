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

// AdminUserPO is the persistence object for AdminUser.
type AdminUserPO struct {
	ID           shared.GlobalID `gorm:"primaryKey;type:varchar(36);not null"`
	Username     string          `gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string          `gorm:"size:255;not null"`
	Nickname     string          `gorm:"size:64;not null"`
	Email        string          `gorm:"size:128"`
	Avatar       string          `gorm:"type:longtext"`
	Language     string          `gorm:"size:32;default:zh"`
	Skin         string          `gorm:"size:32;default:dark"`
	Active       bool            `gorm:"type:tinyint(1);default(1)"`
	IsDeleted    bool            `gorm:"type:tinyint(1);default(0)"`
	CreateTime   time.Time       `gorm:"autoCreateTime"`
	UpdateTime   time.Time       `gorm:"autoUpdateTime"`
	DeleteTime   time.Time       `gorm:"type:timestamp"`
}

func (AdminUserPO) TableName() string { return "admin_users" }

func poToAdminUser(po *AdminUserPO) *management.AdminUser {
	if po == nil {
		return nil
	}
	return &management.AdminUser{
		ID:           po.ID,
		Username:     po.Username,
		PasswordHash: po.PasswordHash,
		Nickname:     po.Nickname,
		Email:        po.Email,
		Avatar:       po.Avatar,
		Language:     po.Language,
		Skin:         po.Skin,
		Active:       po.Active,
		IsDeleted:    po.IsDeleted,
		CreateTime:   po.CreateTime,
		UpdateTime:   po.UpdateTime,
		DeleteTime:   po.DeleteTime,
	}
}

func adminUserToPO(u *management.AdminUser) *AdminUserPO {
	if u == nil {
		return nil
	}
	return &AdminUserPO{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Nickname:     u.Nickname,
		Email:        u.Email,
		Avatar:       u.Avatar,
		Language:     u.Language,
		Skin:         u.Skin,
		Active:       u.Active,
		IsDeleted:    u.IsDeleted,
		CreateTime:   u.CreateTime,
		UpdateTime:   u.UpdateTime,
		DeleteTime:   u.DeleteTime,
	}
}

type AdminUserRepository struct {
	db persistence.DBProvider
}

func NewAdminUserRepository(db persistence.DBProvider) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

func (r *AdminUserRepository) Save(ctx context.Context, u *management.AdminUser) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}

	// Check for duplicate username.
	var exists int64
	if err := g.Model(&AdminUserPO{}).
		Where("username = ? AND id <> ?", u.Username, u.ID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return management.ErrAdminUserExists
	}

	po := adminUserToPO(u)
	po.UpdateTime = time.Now()
	return g.Save(po).Error
}

func (r *AdminUserRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.AdminUser, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po AdminUserPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrAdminUserNotFound
		}
		return nil, err
	}
	return poToAdminUser(&po), nil
}

func (r *AdminUserRepository) FindByUsername(ctx context.Context, username string) (*management.AdminUser, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po AdminUserPO
	if err := g.Where("username = ?", username).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrAdminUserNotFound
		}
		return nil, err
	}
	return poToAdminUser(&po), nil
}

func (r *AdminUserRepository) Search(ctx context.Context, keyword string, page, pageSize int) ([]management.AdminUser, int64, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}

	db := g.Model(&AdminUserPO{}).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", like, like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := g.Model(&AdminUserPO{}).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", like, like, like)
	}
	offset := (page - 1) * pageSize
	var pos []AdminUserPO
	if err := q.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "username"},
		Desc:   false,
	}).Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	users := make([]management.AdminUser, len(pos))
	for i := range pos {
		users[i] = *poToAdminUser(&pos[i])
	}
	return users, total, nil
}

func (r *AdminUserRepository) SoftDelete(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	now := time.Now()
	return g.Model(&AdminUserPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted":  true,
			"active":      false,
			"update_time": now,
			"delete_time": now,
		}).Error
}

func (r *AdminUserRepository) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	var po AdminUserPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		return err
	}
	return g.Model(&AdminUserPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"active":      !po.Active,
			"update_time": time.Now(),
		}).Error
}

func (r *AdminUserRepository) UpdatePassword(ctx context.Context, id shared.GlobalID, hash string) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Model(&AdminUserPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"password_hash": hash,
			"update_time":   time.Now(),
		}).Error
}
