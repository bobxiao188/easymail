// Package sqlite provides SQLite persistence.
// internal/infrastructure/persistence/sqlite/contact_repo.go - Contact repository

package sqlite

import (
	"context"
	"errors"
	"strings"
	"time"

	"easymail/internal/domain/contact"
	"easymail/internal/domain/shared"

	"path/filepath"

	"gorm.io/gorm"
)

// ContactPO is the persistence object for Contact.
type ContactPO struct {
	ID             shared.GlobalID  `gorm:"primaryKey;type:varchar(36);not null"`
	MailUserID     shared.GlobalID  `gorm:"type:varchar(36);not null;uniqueIndex:uk_contact_email_account"`
	ContactName    string           `gorm:"size:255;not null"`
	ContactEmail   string           `gorm:"size:255;not null;uniqueIndex:uk_contact_email_account"`
	ContactPhone   string           `gorm:"size:255;default:''"`
	ContactAddress string           `gorm:"size:255;default:''"`
	ContactCity    string           `gorm:"size:255;default:''"`
	ContactState   string           `gorm:"size:255;default:''"`
	ContactZip     string           `gorm:"size:255;default:''"`
	ContactCountry string           `gorm:"size:255;default:''"`
	CreateTime     time.Time        `gorm:"autoCreateTime"`
	ContactGroupID *shared.GlobalID `gorm:"type:varchar(36);index"`
}

func (ContactPO) TableName() string { return "contacts" }

// ContactGroupPO is the persistence object for ContactGroup.
type ContactGroupPO struct {
	ID         shared.GlobalID `gorm:"primaryKey;type:varchar(36);not null"`
	MailUserID shared.GlobalID `gorm:"type:varchar(36);not null;uniqueIndex:uk_group_name_account"`
	GroupName  string          `gorm:"size:255;not null;uniqueIndex:uk_group_name_account"`
	IsDefault  bool            `gorm:"type:tinyint;default:0"`
	CreateTime time.Time       `gorm:"autoCreateTime"`
}

func (ContactGroupPO) TableName() string { return "contact_groups" }

// --- PO <-> Domain converters ---

func poToContact(po *ContactPO) *contact.Contact {
	if po == nil {
		return nil
	}
	return &contact.Contact{
		ID:             po.ID,
		MailUserID:     po.MailUserID,
		ContactName:    po.ContactName,
		ContactEmail:   po.ContactEmail,
		ContactPhone:   po.ContactPhone,
		ContactAddress: po.ContactAddress,
		ContactCity:    po.ContactCity,
		ContactState:   po.ContactState,
		ContactZip:     po.ContactZip,
		ContactCountry: po.ContactCountry,
		CreateTime:     po.CreateTime,
		ContactGroupID: po.ContactGroupID,
	}
}

func contactToPO(c *contact.Contact) *ContactPO {
	if c == nil {
		return nil
	}
	return &ContactPO{
		ID:             c.ID,
		MailUserID:     c.MailUserID,
		ContactName:    c.ContactName,
		ContactEmail:   c.ContactEmail,
		ContactPhone:   c.ContactPhone,
		ContactAddress: c.ContactAddress,
		ContactCity:    c.ContactCity,
		ContactState:   c.ContactState,
		ContactZip:     c.ContactZip,
		ContactCountry: c.ContactCountry,
		CreateTime:     c.CreateTime,
		ContactGroupID: c.ContactGroupID,
	}
}

func poToContactGroup(po *ContactGroupPO) *contact.ContactGroup {
	if po == nil {
		return nil
	}
	return &contact.ContactGroup{
		ID:         po.ID,
		MailUserID: po.MailUserID,
		GroupName:  po.GroupName,
		IsDefault:  po.IsDefault,
		CreateTime: po.CreateTime,
	}
}

func contactGroupToPO(g *contact.ContactGroup) *ContactGroupPO {
	if g == nil {
		return nil
	}
	return &ContactGroupPO{
		ID:         g.ID,
		MailUserID: g.MailUserID,
		GroupName:  g.GroupName,
		IsDefault:  g.IsDefault,
		CreateTime: g.CreateTime,
	}
}

// UserContactRepository implements contact.ContactRepository using per-user SQLite.
type UserContactRepository struct {
	pool        *Pool
	getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)
}

func NewUserContactRepository(pool *Pool, getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)) *UserContactRepository {
	return &UserContactRepository{pool: pool, getDataPath: getDataPath}
}

func (r *UserContactRepository) dbForUser(ctx context.Context, uid shared.GlobalID) (*gorm.DB, error) {
	root, dp, err := r.getDataPath(ctx, uid)
	if err != nil {
		return nil, err
	}
	if dp == "" {
		return nil, gorm.ErrRecordNotFound
	}
	abspath := filepath.Join(root, dp)
	return r.pool.DB(filepath.Join(abspath, "user.db"))
}

func (r *UserContactRepository) Save(ctx context.Context, c *contact.Contact) error {
	db, err := r.dbForUser(ctx, c.MailUserID)
	if err != nil {
		return err
	}
	g := db.WithContext(ctx)
	var exists int64
	if err := g.Model(&ContactPO{}).
		Where("contact_email = ? AND mail_user_id = ? AND id <> ?", c.ContactEmail, c.MailUserID, c.ID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return contact.ErrContactDuplicate
	}
	return g.Save(contactToPO(c)).Error
}

func (r *UserContactRepository) FindByID(ctx context.Context, id shared.GlobalID) (*contact.Contact, error) {
	db, err := r.dbForUser(ctx, id)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	var po ContactPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contact.ErrContactNotFound
		}
		return nil, err
	}
	return poToContact(&po), nil
}

func (r *UserContactRepository) FindByAccountAndID(ctx context.Context, MailUserID, contactID shared.GlobalID) (*contact.Contact, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	var po ContactPO
	if err := g.Where("id = ? AND mail_user_id = ?", contactID, MailUserID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contact.ErrContactNotFound
		}
		return nil, err
	}
	return poToContact(&po), nil
}

func (r *UserContactRepository) FindByEmail(ctx context.Context, MailUserID shared.GlobalID, email string) (*contact.Contact, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	var po ContactPO
	if err := g.Where("contact_email = ? AND mail_user_id = ?", email, MailUserID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contact.ErrContactNotFound
		}
		return nil, err
	}
	return poToContact(&po), nil
}

func (r *UserContactRepository) Delete(ctx context.Context, MailUserID, contactID shared.GlobalID) error {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return err
	}
	g := db.WithContext(ctx)
	result := g.Where("id = ? AND mail_user_id = ?", contactID, MailUserID).Delete(&ContactPO{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return contact.ErrContactNotFound
	}
	return nil
}

func (r *UserContactRepository) Search(ctx context.Context, MailUserID shared.GlobalID, keyword string, groupID *shared.GlobalID, ungrouped bool) ([]contact.Contact, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	dbq := g.Where("mail_user_id = ?", MailUserID)
	if ungrouped {
		dbq = dbq.Where("contact_group_id IS NULL")
	} else if groupID != nil {
		dbq = dbq.Where("contact_group_id = ?", *groupID)
	}
	if keyword != "" {
		kw := "%" + strings.TrimSpace(keyword) + "%"
		dbq = dbq.Where("contact_name LIKE ? OR contact_email LIKE ?", kw, kw)
	}
	var pos []ContactPO
	if err := dbq.Order("contact_name ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]contact.Contact, len(pos))
	for i := range pos {
		result[i] = *poToContact(&pos[i])
	}
	return result, nil
}

// Count 返回符合条件的联系人总数
func (r *UserContactRepository) Count(ctx context.Context, MailUserID shared.GlobalID, groupID *shared.GlobalID, ungrouped bool) (int64, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return 0, err
	}
	g := db.WithContext(ctx)
	dbq := g.Model(&ContactPO{}).Where("mail_user_id = ?", MailUserID)
	if ungrouped {
		dbq = dbq.Where("contact_group_id IS NULL")
	} else if groupID != nil {
		dbq = dbq.Where("contact_group_id = ?", *groupID)
	}
	var count int64
	if err := dbq.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SearchPaged 返回分页的联系人列表
func (r *UserContactRepository) SearchPaged(ctx context.Context, MailUserID shared.GlobalID, keyword string, groupID *shared.GlobalID, ungrouped bool, page, pageSize int) ([]contact.Contact, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	dbq := g.Where("mail_user_id = ?", MailUserID)
	if ungrouped {
		dbq = dbq.Where("contact_group_id IS NULL")
	} else if groupID != nil {
		dbq = dbq.Where("contact_group_id = ?", *groupID)
	}
	if keyword != "" {
		kw := "%" + strings.TrimSpace(keyword) + "%"
		dbq = dbq.Where("contact_name LIKE ? OR contact_email LIKE ?", kw, kw)
	}
	// 计算偏移量
	offset := (page - 1) * pageSize
	var pos []ContactPO
	if err := dbq.Order("contact_name ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]contact.Contact, len(pos))
	for i := range pos {
		result[i] = *poToContact(&pos[i])
	}
	return result, nil
}

// UserContactGroupRepository implements contact.ContactGroupRepository using per-user SQLite.
type UserContactGroupRepository struct {
	pool        *Pool
	getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)
}

func NewUserContactGroupRepository(pool *Pool, getDataPath func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error)) *UserContactGroupRepository {
	return &UserContactGroupRepository{pool: pool, getDataPath: getDataPath}
}

func (r *UserContactGroupRepository) dbForUser(ctx context.Context, uid shared.GlobalID) (*gorm.DB, error) {
	root, dp, err := r.getDataPath(ctx, uid)
	if err != nil {
		return nil, err
	}
	if dp == "" {
		return nil, gorm.ErrRecordNotFound
	}
	abspath := filepath.Join(root, dp)
	return r.pool.DB(filepath.Join(abspath, "user.db"))
}

func (r *UserContactGroupRepository) Save(ctx context.Context, g *contact.ContactGroup) error {
	db, err := r.dbForUser(ctx, g.MailUserID)
	if err != nil {
		return err
	}
	gdb := db.WithContext(ctx)
	var exists int64
	if err := gdb.Model(&ContactGroupPO{}).
		Where("group_name = ? AND mail_user_id = ? AND id <> ?", g.GroupName, g.MailUserID, g.ID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return contact.ErrGroupDuplicate
	}
	return gdb.Save(contactGroupToPO(g)).Error
}

func (r *UserContactGroupRepository) FindByID(ctx context.Context, id shared.GlobalID) (*contact.ContactGroup, error) {
	// Per-user store: FindByID without MailUserID is not supported.
	// Use FindByAccountAndID(ctx, MailUserID, groupID) instead.
	return nil, contact.ErrGroupNotFound
}

func (r *UserContactGroupRepository) FindByAccountAndID(ctx context.Context, MailUserID, groupID shared.GlobalID) (*contact.ContactGroup, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	var po ContactGroupPO
	if err := g.Where("id = ? AND mail_user_id = ?", groupID, MailUserID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contact.ErrGroupNotFound
		}
		return nil, err
	}
	return poToContactGroup(&po), nil
}

func (r *UserContactGroupRepository) FindByName(ctx context.Context, MailUserID shared.GlobalID, name string) (*contact.ContactGroup, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	var po ContactGroupPO
	if err := g.Where("group_name = ? AND mail_user_id = ?", name, MailUserID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, contact.ErrGroupNotFound
		}
		return nil, err
	}
	return poToContactGroup(&po), nil
}

func (r *UserContactGroupRepository) Delete(ctx context.Context, MailUserID, groupID shared.GlobalID) error {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return err
	}
	g := db.WithContext(ctx)
	return g.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ContactPO{}).
			Where("mail_user_id = ? AND contact_group_id = ?", MailUserID, groupID).
			Update("contact_group_id", nil).Error; err != nil {
			return err
		}
		result := tx.Where("id = ? AND mail_user_id = ?", groupID, MailUserID).Delete(&ContactGroupPO{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return contact.ErrGroupNotFound
		}
		return nil
	})
}

func (r *UserContactGroupRepository) ListByAccount(ctx context.Context, MailUserID shared.GlobalID) ([]contact.ContactGroup, error) {
	db, err := r.dbForUser(ctx, MailUserID)
	if err != nil {
		return nil, err
	}
	g := db.WithContext(ctx)
	var pos []ContactGroupPO
	if err := g.Where("mail_user_id = ?", MailUserID).Order("group_name ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]contact.ContactGroup, len(pos))
	for i := range pos {
		result[i] = *poToContactGroup(&pos[i])
	}
	return result, nil
}
