package mysql

import (
	"context"
	"errors"

	"easymail/internal/domain/mailbox"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence"

	"time"

	"gorm.io/gorm"
)

// FolderPO is the persistence object for Folder.
type FolderPO struct {
	ID         shared.GlobalID    `gorm:"primaryKey;type:varchar(36);not null"`
	MailUserID shared.GlobalID    `gorm:"type:varchar(36);not null;index"`
	FolderName string             `gorm:"size:255;not null"`
	FolderKind mailbox.FolderKind `gorm:"type:tinyint;not null;default:0"`
	IsDeleted  bool               `gorm:"type:tinyint(1);default:0"`
	CreateTime time.Time          `gorm:"autoCreateTime"`
	UpdateTime time.Time          `gorm:"autoUpdateTime"`
}

func (FolderPO) TableName() string { return "mailbox_folders" }

func poToFolder(po *FolderPO) *mailbox.Folder {
	if po == nil { return nil }
	return &mailbox.Folder{
		ID:         po.ID,
		MailUserID: po.MailUserID,
		FolderName: po.FolderName,
		FolderKind: po.FolderKind,
		CreateTime: po.CreateTime,
		UpdateTime: po.UpdateTime,
	}
}

func folderToPO(f *mailbox.Folder) *FolderPO {
	if f == nil { return nil }
	return &FolderPO{
		ID:         f.ID,
		MailUserID: f.MailUserID,
		FolderName: f.FolderName,
		FolderKind: f.FolderKind,
		IsDeleted:  false,
		CreateTime: time.Time{},
		UpdateTime: time.Time{},
	}
}

// txDBProvider wraps a *gorm.DB (typically a transaction) as persistence.DBProvider.
type txDBProvider struct { tx *gorm.DB }
func (p *txDBProvider) DB(ctx context.Context) (any, error) { return p.tx.WithContext(ctx), nil }
func (p *txDBProvider) Ping(ctx context.Context) error { sqlDB, err := p.tx.DB(); if err != nil { return err }; return sqlDB.PingContext(ctx) }
func (p *txDBProvider) Close() error { return nil }

// FolderRepository implements mailbox.FolderRepository.
type FolderRepository struct {
	db persistence.DBProvider
}

func NewFolderRepository(db persistence.DBProvider) *FolderRepository {
	return &FolderRepository{db: db}
}

func (r *FolderRepository) Save(ctx context.Context, f *mailbox.Folder) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Save(folderToPO(f)).Error
}

func (r *FolderRepository) FindByID(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) (*mailbox.Folder, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po FolderPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mailbox.ErrFolderNotFound
		}
		return nil, err
	}
	return poToFolder(&po), nil
}

func (r *FolderRepository) FindByMailUserAndKind(ctx context.Context, mailUserID shared.GlobalID, kind mailbox.FolderKind) (*mailbox.Folder, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po FolderPO
	if err := g.Where("mail_user_id = ? AND folder_kind = ? AND is_deleted = ?", mailUserID, kind, false).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mailbox.ErrFolderNotFound
		}
		return nil, err
	}
	return poToFolder(&po), nil
}

func (r *FolderRepository) ListByMailUser(ctx context.Context, mailUserID shared.GlobalID) ([]*mailbox.Folder, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []FolderPO
	if err := g.Where("mail_user_id = ? AND is_deleted = ?", mailUserID, false).Order("folder_kind ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]*mailbox.Folder, len(pos))
	for i := range pos {
		result[i] = poToFolder(&pos[i])
	}
	return result, nil
}

func (r *FolderRepository) Update(ctx context.Context, f *mailbox.Folder) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Save(folderToPO(f)).Error
}

func (r *FolderRepository) UpdateName(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID, name string) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Model(&FolderPO{}).Where("id = ? AND is_deleted = ?", id, false).Update("folder_name", name).Error
}

func (r *FolderRepository) SoftDelete(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Model(&FolderPO{}).Where("id = ?", id).Update("is_deleted", true).Error
}
