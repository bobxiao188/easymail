// internal/app/management/provision_service.go
package management

import (
	"context"
	"os"
	"path/filepath"

	"easymail/internal/domain/contact"
	"easymail/internal/domain/mailbox"
	"easymail/internal/domain/messaging/storagepath"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence/sqlite"
)

// UserProvisionService handles creating a user's storage directory and default folders.
type UserProvisionService interface {
	// Provision creates the user's data directory and default folders if they don't exist.
	Provision(ctx context.Context, mailUserID shared.GlobalID) error
	// EnsureFolders checks if folders exist and creates defaults if missing.
	EnsureFolders(ctx context.Context, mailUserID shared.GlobalID) error
	// Deprovision closes the user's SQLite database and removes the data directory.
	Deprovision(ctx context.Context, root, dataPath string) error
}

type userProvisionServiceImpl struct {
	pool        *sqlite.Pool
	getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)
}

func NewUserProvisionService(pool *sqlite.Pool, getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)) UserProvisionService {
	return &userProvisionServiceImpl{
		pool:        pool,
		getDataPath: getDataPath,
	}
}

// defaultFolderDefs returns the default system folder definitions for a new user.
func defaultFolderDefs(mailUserID shared.GlobalID) []struct {
	Name string
	Kind mailbox.FolderKind
} {
	return []struct {
		Name string
		Kind mailbox.FolderKind
	}{
		{mailbox.IMAPNameForKind(mailbox.FolderInbox), mailbox.FolderInbox},
		{mailbox.IMAPNameForKind(mailbox.FolderSent), mailbox.FolderSent},
		{mailbox.IMAPNameForKind(mailbox.FolderDraft), mailbox.FolderDraft},
		{mailbox.IMAPNameForKind(mailbox.FolderTrash), mailbox.FolderTrash},
		{mailbox.IMAPNameForKind(mailbox.FolderSpam), mailbox.FolderSpam},
		{mailbox.IMAPNameForKind(mailbox.FolderQuarantine), mailbox.FolderQuarantine},
	}
}

func (s *userProvisionServiceImpl) Provision(ctx context.Context, mailUserID shared.GlobalID) error {
	// Create user data directory
	root, dp, err := s.getDataPath(ctx, mailUserID)
	if err != nil {
		return err
	}
	abs := filepath.Join(root, dp)
	if err := os.MkdirAll(abs, 0755); err != nil {
		return err
	}

	// Open/create user.db (pool auto-migrates tables)
	userDBPath := storagepath.UserDBPath(abs)
	_, err = s.pool.DB(userDBPath)
	if err != nil {
		return err
	}

	// Create default folders
	if err := s.createDefaultFolders(ctx, mailUserID); err != nil {
		return err
	}

	// Create default contact group
	return s.createDefaultContactGroup(ctx, mailUserID)
}

func (s *userProvisionServiceImpl) createDefaultFolders(ctx context.Context, mailUserID shared.GlobalID) error {
	root, dp, err := s.getDataPath(ctx, mailUserID)
	if err != nil {
		return err
	}
	abs := filepath.Join(root, dp)
	userDBPath := storagepath.UserDBPath(abs)
	db, err := s.pool.DB(userDBPath)
	if err != nil {
		return err
	}

	for _, fd := range defaultFolderDefs(mailUserID) {
		var count int64
		if err := db.WithContext(ctx).Model(&sqlite.FolderPO{}).Where("mail_user_id = ? AND folder_kind = ?", string(mailUserID), int(fd.Kind)).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		f, err := mailbox.NewFolder(mailUserID, fd.Name, fd.Kind)
		if err != nil {
			return err
		}
		f.UIDValidity = mailbox.GenerateUIDValidity()
		if err := db.WithContext(ctx).Create(sqlite.FolderToPO(f)).Error; err != nil {
			return err
		}
	}

	// Backfill UIDVALIDITY for folders created before this field existed.
	// This changes UIDVALIDITY for existing mailboxes, forcing IMAP clients
	// to discard stale UID caches and perform a full resync.
	var zeroFolders []sqlite.FolderPO
	if err := db.WithContext(ctx).Where("mail_user_id = ? AND uid_validity = ?", string(mailUserID), 0).Find(&zeroFolders).Error; err != nil {
		return err
	}
	for i := range zeroFolders {
		zeroFolders[i].UIDValidity = mailbox.GenerateUIDValidity()
		if err := db.WithContext(ctx).Model(&sqlite.FolderPO{}).Where("id = ?", zeroFolders[i].ID).Update("uid_validity", zeroFolders[i].UIDValidity).Error; err != nil {
			return err
		}
	}
	return nil
}

// createDefaultContactGroup creates a "default" contact group if one does not exist.
func (s *userProvisionServiceImpl) createDefaultContactGroup(ctx context.Context, mailUserID shared.GlobalID) error {
	root, dp, err := s.getDataPath(ctx, mailUserID)
	if err != nil {
		return err
	}
	abs := filepath.Join(root, dp)
	userDBPath := storagepath.UserDBPath(abs)
	db, err := s.pool.DB(userDBPath)
	if err != nil {
		return err
	}

	var count int64
	if err := db.WithContext(ctx).Model(&sqlite.ContactGroupPO{}).
		Where("mail_user_id = ? AND group_name = ?", string(mailUserID), "default").
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	g, err := contact.NewContactGroupWithDefault(mailUserID, "default", true)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Create(&sqlite.ContactGroupPO{
		ID:         g.ID,
		MailUserID: g.MailUserID,
		GroupName:  g.GroupName,
		IsDefault:  g.IsDefault,
		CreateTime: g.CreateTime,
	}).Error
}

func (s *userProvisionServiceImpl) EnsureFolders(ctx context.Context, mailUserID shared.GlobalID) error {
	root, dp, err := s.getDataPath(ctx, mailUserID)
	if err != nil {
		return err
	}
	if dp == "" {
		return nil
	}
	// Ensure storage directory exists (may have been deleted)
	abs := filepath.Join(root, dp)
	if err := os.MkdirAll(abs, 0755); err != nil {
		return err
	}
	return s.createDefaultFolders(ctx, mailUserID)
}

func (s *userProvisionServiceImpl) Deprovision(ctx context.Context, root, dataPath string) error {
	abs := filepath.Join(root, dataPath)
	dbPath := filepath.Join(abs, "user.db")
	if err := s.pool.CloseDB(dbPath); err != nil {
		return err
	}
	return os.RemoveAll(abs)
}

var _ UserProvisionService = (*userProvisionServiceImpl)(nil)
