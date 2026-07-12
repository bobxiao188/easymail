package sqlite

import (
	"context"
	"path/filepath"

	"easymail/internal/domain/messaging"
	"easymail/internal/domain/messaging/repository"
	"easymail/internal/domain/messaging/storagepath"
	"easymail/internal/domain/shared"
	"easymail/pkg/constants"

	"gorm.io/gorm"
)

// EmailPO is the persistence object for Email (old int64-based model).
type EmailPO struct {
	ID             int64           `gorm:"primaryKey;autoIncrement"`
	MailUserID     shared.GlobalID `gorm:"type:varchar(36);index;not null"`
	JobID          string          `gorm:"size:255;default:"`
	QueueID        string          `gorm:"size:255;default:"`
	Subject        string          `gorm:"size:1024"`
	Sender         string          `gorm:"size:512;not null"`
	Recipient      string          `gorm:"size:512;not null"`
	CarbonCopy     string          `gorm:"type:text"`
	BlindCopy      string          `gorm:"type:text"`
	MailTime       string          `gorm:"size:64"`
	FolderID       int64           `gorm:"index;not null"`
	ReadStatus     int             `gorm:"type:tinyint;default:0"`
	Flagged        bool            `gorm:"type:tinyint(1);default:0"`
	HasAttachments bool            `gorm:"type:tinyint(1);default:0;index"` // 是否有附件
	Body           string          `gorm:"type:text"`
	Snippet        string          `gorm:"size:512"` // 邮件摘要，最多512字符
	MailSize       int64           `gorm:"default:0"`
	IsDeleted      bool            `gorm:"type:tinyint(1);default:0;index"`
	FilePath       string          `gorm:"size:1024;not null"`
	SMTPStatus     string          `gorm:"size:16;default:;index"`
	SMTPError      string          `gorm:"type:text"`
	SMTPSentAt     string          `gorm:"size:64"`
}

type FolderIDMapPO struct {
	GlobalID  string `gorm:"primaryKey;type:varchar(36);not null"`
	NumericID int64  `gorm:"not null;index"`
}

func (FolderIDMapPO) TableName() string { return "folder_id_map" }

func (EmailPO) TableName() string { return "emails" }

func poToEmail(po *EmailPO) *messaging.Email {
	if po == nil {
		return nil
	}
	return &messaging.Email{
		ID:             po.ID,
		MailUserID:     po.MailUserID,
		JobID:          po.JobID,
		QueueID:        po.QueueID,
		Subject:        po.Subject,
		Sender:         po.Sender,
		Recipient:      po.Recipient,
		CarbonCopy:     po.CarbonCopy,
		BlindCopy:      po.BlindCopy,
		MailTime:       po.MailTime,
		FolderID:       po.FolderID,
		ReadStatus:     constants.ReadStatus(po.ReadStatus),
		Flagged:        po.Flagged,
		HasAttachments: po.HasAttachments,
		Body:           po.Body,
		Snippet:        po.Snippet,
		MailSize:       po.MailSize,
		IsDeleted:      po.IsDeleted,
		FilePath:       po.FilePath,
		SMTPStatus:     po.SMTPStatus,
		SMTPError:      po.SMTPError,
		SMTPSentAt:     po.SMTPSentAt,
	}
}

func emailToPO(e *messaging.Email) *EmailPO {
	if e == nil {
		return nil
	}
	return &EmailPO{
		ID:             e.ID,
		MailUserID:     e.MailUserID,
		JobID:          e.JobID,
		QueueID:        e.QueueID,
		Subject:        e.Subject,
		Sender:         e.Sender,
		Recipient:      e.Recipient,
		CarbonCopy:     e.CarbonCopy,
		BlindCopy:      e.BlindCopy,
		MailTime:       e.MailTime,
		FolderID:       e.FolderID,
		ReadStatus:     int(e.ReadStatus),
		Flagged:        e.Flagged,
		HasAttachments: e.HasAttachments,
		Body:           e.Body,
		Snippet:        e.Snippet,
		MailSize:       e.MailSize,
		IsDeleted:      e.IsDeleted,
		FilePath:       e.FilePath,
		SMTPStatus:     e.SMTPStatus,
		SMTPError:      e.SMTPError,
		SMTPSentAt:     e.SMTPSentAt,
	}
}

// MailIndexRepository implements repository.EmailRepository using per-mailbox SQLite files.
type MailIndexRepository struct {
	pool        *Pool
	getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)
}

func NewMailIndexRepository(getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error), pool *Pool) *MailIndexRepository {
	return &MailIndexRepository{
		pool:        pool,
		getDataPath: getDataPath,
	}
}

func (r *MailIndexRepository) dbForUser(ctx context.Context, uid shared.GlobalID) (*gorm.DB, error) {
	root, dp, err := r.getDataPath(ctx, uid)
	if err != nil {
		return nil, err
	}
	if dp == "" {
		return nil, gorm.ErrRecordNotFound
	}
	abs := filepath.Join(root, dp)
	path := storagepath.UserDBPath(abs)
	return r.pool.DB(path)
}

func (r *MailIndexRepository) Save(ctx context.Context, msg *messaging.Email) error {
	db, err := r.dbForUser(ctx, msg.MailUserID)
	if err != nil {
		return err
	}
	// For new records (ID == 0), save first to get auto-increment ID
	if msg.ID == 0 {
		po := emailToPO(msg)
		if err := db.WithContext(ctx).Create(po).Error; err != nil {
			return err
		}
		// Copy back auto-generated ID
		msg.ID = po.ID
		// Now set the hash-sharded file path based on the actual ID
		msg.FilePath = storagepath.HashShardPath(msg.ID)
		return db.WithContext(ctx).Model(&EmailPO{}).Where("id = ?", msg.ID).Update("file_path", msg.FilePath).Error
	}
	// For existing records, apply hash sharding if FilePath is empty
	if msg.FilePath == "" {
		msg.FilePath = storagepath.HashShardPath(msg.ID)
	}
	return db.WithContext(ctx).Save(emailToPO(msg)).Error
}

func (r *MailIndexRepository) GetMailQuantity(ctx context.Context, accID shared.GlobalID) (int64, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND is_deleted = ?", accID, false).Count(&count).Error
	return count, err
}

func (r *MailIndexRepository) GetMailUsage(ctx context.Context, accID shared.GlobalID) (int64, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return 0, err
	}
	var total int64
	err = db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND is_deleted = ?", accID, false).Select("COALESCE(SUM(mail_size), 0)").Scan(&total).Error
	return total, err
}

func (r *MailIndexRepository) MarkRead(ctx context.Context, accID shared.GlobalID, mailID int64, readSource constants.ReadStatus) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&EmailPO{}).Where("id = ? AND mail_user_id = ?", mailID, accID).Update("read_status", int(readSource)).Error
}

func (r *MailIndexRepository) GetMail(ctx context.Context, accID shared.GlobalID, mailID int64) (*messaging.Email, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return nil, err
	}
	var po EmailPO
	if err := db.WithContext(ctx).Where("id = ? AND mail_user_id = ?", mailID, accID).First(&po).Error; err != nil {
		return nil, err
	}
	return poToEmail(&po), nil
}

func (r *MailIndexRepository) DeleteMail(ctx context.Context, accID shared.GlobalID, mailID int64) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&EmailPO{}).Where("id = ? AND mail_user_id = ?", mailID, accID).Update("is_deleted", true).Error
}

func (r *MailIndexRepository) HardDeleteMail(ctx context.Context, accID shared.GlobalID, mailID int64) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Where("id = ? AND mail_user_id = ?", mailID, accID).Delete(&EmailPO{}).Error
}

func (r *MailIndexRepository) MoveMail(ctx context.Context, accID shared.GlobalID, mailID int64, folderID int64) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&EmailPO{}).Where("id = ? AND mail_user_id = ?", mailID, accID).Update("folder_id", folderID).Error
}

func (r *MailIndexRepository) CountActiveInFolder(ctx context.Context, accID shared.GlobalID, folderID int64) (int64, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND folder_id = ? AND is_deleted = ?", accID, folderID, false).Count(&count).Error
	return count, err
}

func (r *MailIndexRepository) CountUnreadInFolder(ctx context.Context, accID shared.GlobalID, folderID int64) (int64, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND folder_id = ? AND is_deleted = ? AND read_status = ?", accID, folderID, false, 0).Count(&count).Error
	return count, err
}

type folderCountRow struct {
	FolderID int64 `gorm:"column:folder_id"`
	Total    int64 `gorm:"column:total"`
	Unread   int64 `gorm:"column:unread"`
}

func (r *MailIndexRepository) CountByFolder(ctx context.Context, accID shared.GlobalID) (map[int64]repository.FolderCounts, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return nil, err
	}
	var rows []folderCountRow
	err = db.WithContext(ctx).Model(&EmailPO{}).
		Select("folder_id, COUNT(*) AS total, SUM(CASE WHEN read_status = 0 THEN 1 ELSE 0 END) AS unread").
		Where("mail_user_id = ? AND is_deleted = ?", accID, false).
		Group("folder_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]repository.FolderCounts, len(rows))
	for _, r := range rows {
		result[r.FolderID] = repository.FolderCounts{Total: r.Total, Unread: r.Unread}
	}
	return result, nil
}

func (r *MailIndexRepository) ListAllIDs(ctx context.Context, accID shared.GlobalID, folderID int64) ([]int64, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return nil, err
	}
	var ids []int64
	err = db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND folder_id = ? AND is_deleted = ?", accID, folderID, false).Order("id ASC").Pluck("id", &ids).Error
	return ids, err
}

func (r *MailIndexRepository) SetFlagged(ctx context.Context, accID shared.GlobalID, mailID int64, flagged bool) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&EmailPO{}).Where("id = ? AND mail_user_id = ?", mailID, accID).Update("flagged", flagged).Error
}

func (r *MailIndexRepository) SetDeleted(ctx context.Context, accID shared.GlobalID, mailID int64, deleted bool) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&EmailPO{}).Where("id = ? AND mail_user_id = ?", mailID, accID).Update("is_deleted", deleted).Error
}

func (r *MailIndexRepository) QueryByFolder(ctx context.Context, accID shared.GlobalID, folderID int64, orderField, orderDir string, page, pageSize int, search string, labelID int64) (total int64, unread int64, emails []messaging.Email, err error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return 0, 0, nil, err
	}

	// Build query with label filter
	var base *gorm.DB
	if labelID > 0 {
		// When label filter is applied, use subquery to find emails with the label
		base = db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND folder_id = ? AND is_deleted = ? AND id IN (SELECT email_id FROM email_labels WHERE label_id = ?)", accID, folderID, false, labelID)
	} else {
		base = db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND folder_id = ? AND is_deleted = ?", accID, folderID, false)
	}

	if search != "" {
		// Use INSTR for Chinese character support (COLLATE NOCASE doesn't work for Chinese)
		// INSTR returns 0 if not found, >0 if found
		base = base.Where("(INSTR(subject, ?) > 0 OR INSTR(sender, ?) > 0 OR INSTR(recipient, ?) > 0 OR INSTR(snippet, ?) > 0)", search, search, search, search)
	}

	// Count total matching records
	if err := base.Count(&total).Error; err != nil {
		return 0, 0, nil, err
	}
	// Count unread records
	if err := base.Where("read_status = ?", int(constants.UnRead)).Count(&unread).Error; err != nil {
		return 0, 0, nil, err
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	orderClause := orderField + " " + orderDir
	// Run paginated query using a fresh Model to avoid Count-mutated SELECT
	var pos []EmailPO
	if labelID > 0 {
		queryDB := db.WithContext(ctx).Model(&EmailPO{}).Where("mail_user_id = ? AND folder_id = ? AND is_deleted = ? AND id IN (SELECT email_id FROM email_labels WHERE label_id = ?)", accID, folderID, false, labelID)
		if search != "" {
			queryDB = queryDB.Where("(INSTR(subject, ?) > 0 OR INSTR(sender, ?) > 0 OR INSTR(recipient, ?) > 0 OR INSTR(snippet, ?) > 0)", search, search, search, search)
		}
		if err := queryDB.Order(orderClause).Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
			return 0, 0, nil, err
		}
	} else {
		queryDB := base.Session(&gorm.Session{NewDB: true}).Model(&EmailPO{}).Where("mail_user_id = ? AND folder_id = ? AND is_deleted = ?", accID, folderID, false)
		if search != "" {
			queryDB = queryDB.Where("(INSTR(subject, ?) > 0 OR INSTR(sender, ?) > 0 OR INSTR(recipient, ?) > 0 OR INSTR(snippet, ?) > 0)", search, search, search, search)
		}
		if err := queryDB.Order(orderClause).Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
			return 0, 0, nil, err
		}
	}
	result := make([]messaging.Email, len(pos))
	for i := range pos {
		result[i] = *poToEmail(&pos[i])
	}
	return total, unread, result, nil
}

func (r *MailIndexRepository) GetFolderNumericID(ctx context.Context, accID shared.GlobalID, globalID string) (int64, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return 0, err
	}
	_ = db.AutoMigrate(&FolderIDMapPO{})
	var m FolderIDMapPO
	err = db.WithContext(ctx).Where("global_id = ?", globalID).First(&m).Error
	if err != nil {
		return 0, err
	}
	return m.NumericID, nil
}

func (r *MailIndexRepository) SetFolderNumericID(ctx context.Context, accID shared.GlobalID, globalID string, numericID int64) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	_ = db.AutoMigrate(&FolderIDMapPO{})
	return db.WithContext(ctx).Save(&FolderIDMapPO{GlobalID: globalID, NumericID: numericID}).Error
}

func (r *MailIndexRepository) GetGlobalIDByNumericID(ctx context.Context, accID shared.GlobalID, numericID int64) (string, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return "", err
	}
	_ = db.AutoMigrate(&FolderIDMapPO{})
	var m FolderIDMapPO
	err = db.WithContext(ctx).Where("numeric_id = ?", numericID).First(&m).Error
	if err != nil {
		return "", err
	}
	return m.GlobalID, nil
}

func (r *MailIndexRepository) GetNextCustomFolderID(ctx context.Context, accID shared.GlobalID) (int64, error) {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return 0, err
	}
	_ = db.AutoMigrate(&FolderIDMapPO{})
	// Find the maximum existing numeric_id for custom folders (>= 100)
	var maxID int64
	err = db.WithContext(ctx).Model(&FolderIDMapPO{}).Select("COALESCE(MAX(numeric_id), 0)").Where("numeric_id >= 100").Scan(&maxID).Error
	if err != nil {
		return 0, err
	}
	if maxID < 100 {
		return 100, nil
	}
	return maxID + 1, nil
}

// UpdateSMTPStatus updates the SMTP delivery status fields for an email.
func (r *MailIndexRepository) UpdateSMTPStatus(ctx context.Context, accID shared.GlobalID, emailID int64, status, errMsg, sentAt string) error {
	db, err := r.dbForUser(ctx, accID)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&EmailPO{}).Where("id = ? AND mail_user_id = ?", emailID, accID).Updates(map[string]interface{}{
		"smtp_status":  status,
		"smtp_error":   errMsg,
		"smtp_sent_at": sentAt,
	}).Error
}
