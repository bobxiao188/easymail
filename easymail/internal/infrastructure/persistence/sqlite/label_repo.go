package sqlite

import (
	"context"

	"easymail/internal/domain/shared"
	"gorm.io/gorm"
)

type LabelPO struct {
	ID         int64           `gorm:"primaryKey;autoIncrement"`
	MailUserID shared.GlobalID `gorm:"type:varchar(36);index;not null"`
	Name       string          `gorm:"size:100;not null"`
	Color      string          `gorm:"size:7;not null;default:#3788d8"`
	IsBuiltin  bool            `gorm:"type:tinyint(1);default:0"`
}

func (LabelPO) TableName() string { return "labels" }

type EmailLabelPO struct {
	EmailID int64 `gorm:"primaryKey"`
	LabelID int64 `gorm:"primaryKey"`
}

func (EmailLabelPO) TableName() string { return "email_labels" }

var builtinLabels = []struct{ Name, Color string }{
	{Name: "重要", Color: "#ef4444"},
	{Name: "工作", Color: "#3b82f6"},
	{Name: "个人", Color: "#10b981"},
	{Name: "旅行", Color: "#f59e0b"},
	{Name: "账单", Color: "#8b5cf6"},
	{Name: "社交", Color: "#06b6d4"},
}

func (r *MailIndexRepository) ensureLabelTables(ctx context.Context, uid shared.GlobalID) error {
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return err
	}
	if err := db.AutoMigrate(&LabelPO{}, &EmailLabelPO{}); err != nil {
		return err
	}
	var count int64
	if err := db.WithContext(ctx).Model(&LabelPO{}).Where("mail_user_id = ? AND is_builtin = ?", uid, true).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, l := range builtinLabels {
		po := LabelPO{MailUserID: uid, Name: l.Name, Color: l.Color, IsBuiltin: true}
		if err := db.WithContext(ctx).Create(&po).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *MailIndexRepository) ListLabels(ctx context.Context, uid shared.GlobalID) ([]shared.LabelDTO, error) {
	if err := r.ensureLabelTables(ctx, uid); err != nil {
		return nil, err
	}
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	var pos []LabelPO
	if err := db.WithContext(ctx).Where("mail_user_id = ?", uid).Order("is_builtin DESC, id ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]shared.LabelDTO, len(pos))
	for i, p := range pos {
		result[i] = shared.LabelDTO{ID: p.ID, Name: p.Name, Color: p.Color, IsBuiltin: p.IsBuiltin}
	}
	return result, nil
}

func (r *MailIndexRepository) CreateLabel(ctx context.Context, uid shared.GlobalID, name, color string) (*shared.LabelDTO, error) {
	if err := r.ensureLabelTables(ctx, uid); err != nil {
		return nil, err
	}
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	po := LabelPO{MailUserID: uid, Name: name, Color: color, IsBuiltin: false}
	if err := db.WithContext(ctx).Create(&po).Error; err != nil {
		return nil, err
	}
	return &shared.LabelDTO{ID: po.ID, Name: po.Name, Color: po.Color, IsBuiltin: po.IsBuiltin}, nil
}

func (r *MailIndexRepository) UpdateLabel(ctx context.Context, uid shared.GlobalID, labelID int64, name, color string) error {
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&LabelPO{}).Where("id = ? AND mail_user_id = ? AND is_builtin = ?", labelID, uid, false).Updates(map[string]interface{}{"name": name, "color": color}).Error
}

func (r *MailIndexRepository) DeleteLabel(ctx context.Context, uid shared.GlobalID, labelID int64) error {
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return err
	}
	if err := db.WithContext(ctx).Where("label_id = ?", labelID).Delete(&EmailLabelPO{}).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Where("id = ? AND mail_user_id = ? AND is_builtin = ?", labelID, uid, false).Delete(&LabelPO{}).Error
}

func (r *MailIndexRepository) SetEmailLabels(ctx context.Context, uid shared.GlobalID, emailID int64, labelIDs []int64) error {
	if err := r.ensureLabelTables(ctx, uid); err != nil {
		return err
	}
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("email_id = ?", emailID).Delete(&EmailLabelPO{}).Error; err != nil {
			return err
		}
		for _, lid := range labelIDs {
			if err := tx.Create(&EmailLabelPO{EmailID: emailID, LabelID: lid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MailIndexRepository) GetEmailLabels(ctx context.Context, uid shared.GlobalID, emailID int64) ([]shared.LabelDTO, error) {
	if err := r.ensureLabelTables(ctx, uid); err != nil {
		return nil, err
	}
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	var labels []shared.LabelDTO
	err = db.WithContext(ctx).Table("labels").
		Select("labels.id, labels.name, labels.color, labels.is_builtin").
		Joins("JOIN email_labels ON email_labels.label_id = labels.id").
		Where("email_labels.email_id = ? AND labels.mail_user_id = ?", emailID, uid).
		Find(&labels).Error
	return labels, err
}
func (r *MailIndexRepository) GetLabelsForEmails(ctx context.Context, uid shared.GlobalID, emailIDs []int64) (map[int64][]shared.LabelDTO, error) {
	if len(emailIDs) == 0 {
		return make(map[int64][]shared.LabelDTO), nil
	}
	if err := r.ensureLabelTables(ctx, uid); err != nil {
		return nil, err
	}
	db, err := r.dbForUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	type row struct {
		EmailID    int64
		LabelID    int64
		LabelName  string
		LabelColor string
		IsBuiltin  bool
	}
	var rows []row
	err = db.WithContext(ctx).Table("email_labels").
		Select("email_labels.email_id, labels.id AS label_id, labels.name AS label_name, labels.color AS label_color, labels.is_builtin").
		Joins("JOIN labels ON labels.id = email_labels.label_id").
		Where("email_labels.email_id IN ? AND labels.mail_user_id = ?", emailIDs, uid).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64][]shared.LabelDTO)
	for _, r := range rows {
		result[r.EmailID] = append(result[r.EmailID], shared.LabelDTO{
			ID: r.LabelID, Name: r.LabelName, Color: r.LabelColor, IsBuiltin: r.IsBuiltin,
		})
	}
	return result, nil
}

