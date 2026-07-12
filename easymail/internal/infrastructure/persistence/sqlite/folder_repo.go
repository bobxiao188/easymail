package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"easymail/internal/domain/mailbox"
	"easymail/internal/domain/messaging/storagepath"
	"easymail/internal/domain/shared"

	"gorm.io/gorm"
)

// FolderPO is the SQLite persistence object for Folder (stored in user.db).
type FolderPO struct {
	ID         string `gorm:"primaryKey;type:varchar(36);not null"`
	MailUserID string `gorm:"type:varchar(36);not null;index"`
	FolderName string `gorm:"size:255;not null"`
	FolderKind int    `gorm:"type:tinyint;not null;default:0"`
	// UIDValidity is the IMAP UIDVALIDITY, persisted per folder.
	UIDValidity uint32 `gorm:"type:int;not null;default:0"`
	IsDeleted  bool   `gorm:"type:tinyint(1);default:0"`
	CreateTime time.Time `gorm:"autoCreateTime"`
	UpdateTime time.Time `gorm:"autoUpdateTime"`
}

func (FolderPO) TableName() string { return "folders" }

func poToFolder(po *FolderPO) *mailbox.Folder {
	if po == nil {
		return nil
	}
	return &mailbox.Folder{
		ID:         shared.GlobalID(po.ID),
		MailUserID: shared.GlobalID(po.MailUserID),
		FolderName: po.FolderName,
		FolderKind: mailbox.FolderKind(po.FolderKind),
		UIDValidity: po.UIDValidity,
		CreateTime: po.CreateTime,
		UpdateTime: po.UpdateTime,
	}
}

func FolderToPO(f *mailbox.Folder) *FolderPO {
	if f == nil {
		return nil
	}
	return &FolderPO{
		ID:         string(f.ID),
		MailUserID: string(f.MailUserID),
		FolderName: f.FolderName,
		FolderKind: int(f.FolderKind),
		UIDValidity: f.UIDValidity,
		IsDeleted:  false,
		CreateTime: time.Time{},
		UpdateTime: time.Time{},
	}
}

// FolderRepository implements mailbox.FolderRepository using per-user SQLite.
type FolderRepository struct {
	pool        *Pool
	getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)
}

func NewFolderRepository(pool *Pool, getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)) *FolderRepository {
	return &FolderRepository{
		pool:        pool,
				getDataPath: getDataPath,
	}
}

func (r *FolderRepository) dbForUser(ctx context.Context, userID shared.GlobalID) (*gorm.DB, error) {
	root, dp, err := r.getDataPath(ctx, userID)
	if err != nil {
		return nil, err
	}
	if dp == "" {
		return nil, gorm.ErrRecordNotFound
	}
	abs := filepath.Join(root, dp)
	path := storagepath.UserDBPath(abs)
	db, err := r.pool.DB(path)
	if err != nil {
		return nil, err
	}
	// Auto-migrate folders table on first access
	_ = db.AutoMigrate(&FolderPO{})
	return db, nil
}

func (r *FolderRepository) Save(ctx context.Context, f *mailbox.Folder) error {
	db, err := r.dbForUser(ctx, f.MailUserID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Save(FolderToPO(f)).Error
}

func (r *FolderRepository) FindByID(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) (*mailbox.Folder, error) {
	db, err := r.dbForUser(ctx, mailUserID)
	if err != nil {
		return nil, err
	}
	var po FolderPO
	if err := db.WithContext(ctx).Where("id = ? AND mail_user_id = ? AND is_deleted = ?", string(id), string(mailUserID), false).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mailbox.ErrFolderNotFound
		}
		return nil, err
	}
	return poToFolder(&po), nil
}

func (r *FolderRepository) FindByMailUserAndKind(ctx context.Context, mailUserID shared.GlobalID, kind mailbox.FolderKind) (*mailbox.Folder, error) {
	db, err := r.dbForUser(ctx, mailUserID)
	if err != nil {
		return nil, err
	}
	var po FolderPO
	if err := db.WithContext(ctx).Where("mail_user_id = ? AND folder_kind = ? AND is_deleted = ?", string(mailUserID), int(kind), false).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mailbox.ErrFolderNotFound
		}
		return nil, err
	}
	return poToFolder(&po), nil
}

func (r *FolderRepository) ListByMailUser(ctx context.Context, mailUserID shared.GlobalID) ([]*mailbox.Folder, error) {
	db, err := r.dbForUser(ctx, mailUserID)
	if err != nil {
		return nil, err
	}
	var pos []FolderPO
	if err := db.WithContext(ctx).Where("mail_user_id = ? AND is_deleted = ?", string(mailUserID), false).Order("folder_kind ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	// Backfill UIDVALIDITY for folders created before this field existed.
	// Changing it forces IMAP clients to discard stale UID caches and resync.
	for i := range pos {
		if pos[i].UIDValidity == 0 {
			pos[i].UIDValidity = mailbox.GenerateUIDValidity()
			if err := db.WithContext(ctx).Model(&FolderPO{}).Where("id = ?", pos[i].ID).Update("uid_validity", pos[i].UIDValidity).Error; err != nil {
				return nil, err
			}
		}
	}
	result := make([]*mailbox.Folder, len(pos))
	for i := range pos {
		result[i] = poToFolder(&pos[i])
	}
	return result, nil
}

func (r *FolderRepository) Update(ctx context.Context, f *mailbox.Folder) error {
	db, err := r.dbForUser(ctx, f.MailUserID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Save(FolderToPO(f)).Error
}

func (r *FolderRepository) UpdateName(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID, name string) error {
	db, err := r.dbForUser(ctx, mailUserID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&FolderPO{}).Where("id = ? AND mail_user_id = ? AND is_deleted = ?", string(id), string(mailUserID), false).Update("folder_name", name).Error
}

func (r *FolderRepository) SoftDelete(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) error {
	db, err := r.dbForUser(ctx, mailUserID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&FolderPO{}).Where("id = ? AND mail_user_id = ?", string(id), string(mailUserID)).Update("is_deleted", true).Error
}
