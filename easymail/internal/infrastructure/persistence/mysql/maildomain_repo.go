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

// MailDomainPO is the persistence object for MailDomain.
// GlobalID (UUID v4) is the database primary key — no auto-increment ID needed.
type MailDomainPO struct {
	ID             shared.GlobalID `gorm:"primaryKey;type:varchar(36);not null"`
	Name           string          `gorm:"uniqueIndex;size:128;not null"`
	Description    string          `gorm:"type:varchar(255)"`
	Active         bool            `gorm:"type:tinyint(1);default(0)"`
	IsDeleted      bool            `gorm:"type:tinyint(1);default(0)"`
	DKIMEnabled    bool            `gorm:"type:tinyint(1);default(0)"`
	DKIMSelector   string          `gorm:"type:varchar(128)"`
	DKIMPrivateKey string          `gorm:"type:text"`
	CreateTime     time.Time       `gorm:"autoCreateTime"`
	UpdateTime     time.Time       `gorm:"autoUpdateTime"`
	DeleteTime     time.Time       `gorm:"type:timestamp"`
}

func (MailDomainPO) TableName() string { return "mail_domains" }

func poToMailDomain(po *MailDomainPO) *management.MailDomain {
	if po == nil {
		return nil
	}
	return &management.MailDomain{
		ID:             po.ID,
		Name:           po.Name,
		Description:    po.Description,
		Active:         po.Active,
		IsDeleted:      po.IsDeleted,
		DKIMEnabled:    po.DKIMEnabled,
		DKIMSelector:   po.DKIMSelector,
		DKIMPrivateKey: po.DKIMPrivateKey,
		CreateTime:     po.CreateTime,
		UpdateTime:     po.UpdateTime,
		DeleteTime:     po.DeleteTime,
	}
}

func mailDomainToPO(d *management.MailDomain) *MailDomainPO {
	if d == nil {
		return nil
	}
	return &MailDomainPO{
		ID:             d.ID,
		Name:           d.Name,
		Description:    d.Description,
		Active:         d.Active,
		IsDeleted:      d.IsDeleted,
		DKIMEnabled:    d.DKIMEnabled,
		DKIMSelector:   d.DKIMSelector,
		DKIMPrivateKey: d.DKIMPrivateKey,
		CreateTime:     d.CreateTime,
		UpdateTime:     d.UpdateTime,
		DeleteTime:     d.DeleteTime,
	}
}

type MailDomainRepository struct {
	db persistence.DBProvider
}

func NewMailDomainRepository(db persistence.DBProvider) *MailDomainRepository {
	return &MailDomainRepository{db: db}
}

func (r *MailDomainRepository) Save(ctx context.Context, d *management.MailDomain) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}

	// Check for duplicate domain name (exclude current record by ID).
	var exists int64
	if err := g.Model(&MailDomainPO{}).
		Where("name = ? AND id <> ?", d.Name, d.ID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return management.ErrDomainExists
	}

	po := mailDomainToPO(d)
	po.UpdateTime = time.Now()
	// GORM's Save(): if primary key exists → UPDATE; if not → INSERT.
	// Since GlobalID is the PK, this works correctly with no extra lookups.
	return g.Save(po).Error
}

func (r *MailDomainRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.MailDomain, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po MailDomainPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrDomainNotFound
		}
		return nil, err
	}
	return poToMailDomain(&po), nil
}

func (r *MailDomainRepository) FindByName(ctx context.Context, name string) (*management.MailDomain, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po MailDomainPO
	if err := g.Where("name = ?", name).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrDomainNotFound
		}
		return nil, err
	}
	return poToMailDomain(&po), nil
}

func (r *MailDomainRepository) FindValidatedByName(ctx context.Context, name string) (*management.MailDomain, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po MailDomainPO
	err = g.Where("name = ? AND active = ? AND is_deleted = ?", name, true, false).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrDomainNotFound
		}
		return nil, err
	}
	return poToMailDomain(&po), nil
}

func (r *MailDomainRepository) FindAllValidated(ctx context.Context) ([]management.MailDomain, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []MailDomainPO
	if err := g.Where("active = ? AND is_deleted = ?", true, false).Order("name").Find(&pos).Error; err != nil {
		return nil, err
	}
	domains := make([]management.MailDomain, len(pos))
	for i := range pos {
		domains[i] = *poToMailDomain(&pos[i])
	}
	return domains, nil
}

func (r *MailDomainRepository) Search(ctx context.Context, keyword string, page, pageSize int, includeDeleted bool) ([]management.MailDomain, int64, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}

	// Build base query
	db := g.Model(&MailDomainPO{}).Session(&gorm.Session{})
	if keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}
	if !includeDeleted {
		db = db.Where("is_deleted = ?", false)
	}

	// Count total matching records
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Query paginated results (ordered by name for stable pagination)
	q := g.Model(&MailDomainPO{}).Session(&gorm.Session{})
	if keyword != "" {
		q = q.Where("name LIKE ?", "%"+keyword+"%")
	}
	if !includeDeleted {
		q = q.Where("is_deleted = ?", false)
	}
	offset := (page - 1) * pageSize
	var pos []MailDomainPO
	if err := q.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "name"},
		Desc:   false,
	}).Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	domains := make([]management.MailDomain, len(pos))
	for i := range pos {
		domains[i] = *poToMailDomain(&pos[i])
	}
	return domains, total, nil
}

func (r *MailDomainRepository) SoftDelete(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	now := time.Now()
	return g.Model(&MailDomainPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_deleted":  true,
			"active":      false,
			"update_time": now,
			"delete_time": now,
		}).Error
}

func (r *MailDomainRepository) HardDelete(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Where("id = ?", id).Delete(&MailDomainPO{}).Error
}

func (r *MailDomainRepository) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	var po MailDomainPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		return err
	}
	return g.Model(&MailDomainPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"active":      !po.Active,
			"update_time": time.Now(),
		}).Error
}
