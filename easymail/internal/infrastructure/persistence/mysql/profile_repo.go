package mysql

import (
	"context"
	"errors"
	"time"

	"easymail/internal/domain/profile"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence"

	"gorm.io/gorm"
)

type UserSettingsPO struct {
	UserID               shared.GlobalID `gorm:"primaryKey;type:varchar(36);not null"`
	DisplayName          string          `gorm:"size:255;default:''"`
	Signature            string          `gorm:"type:text"`
	Language             string          `gorm:"size:10;default:'en'"`
	Theme                string          `gorm:"size:20;default:'blue'"`
	PageSize             int             `gorm:"default:20"`
	ReadingPanePosition  string          `gorm:"size:10;default:'right'"`
	AutoReplyEnabled     bool            `gorm:"default:false"`
	AutoReplySubject     string          `gorm:"size:255;default:''"`
	AutoReplyBody        string          `gorm:"type:text"`
	Phone                string          `gorm:"size:20;default:''"`
	Company              string          `gorm:"size:255;default:''"`
	JobTitle             string          `gorm:"size:255;default:''"`
	NotificationSound    bool            `gorm:"default:true"`
	DesktopNotification  bool            `gorm:"default:true"`
	IncludeOriginalOnReply bool           `gorm:"default:true"`
	ForwardingEnabled    bool            `gorm:"default:false"`
	ForwardingAddress    string          `gorm:"size:255;default:''"`
	SaveSent             bool            `gorm:"default:true"`
	CreatedAt            time.Time       `gorm:"autoCreateTime"`
	UpdatedAt            time.Time       `gorm:"autoUpdateTime"`
}

func (UserSettingsPO) TableName() string { return "user_settings" }

func poToUserSettings(po *UserSettingsPO) *profile.UserSettings {
	if po == nil {
		return nil
	}
	return &profile.UserSettings{
		UserID:               po.UserID,
		DisplayName:          po.DisplayName,
		Signature:            po.Signature,
		Language:             po.Language,
		Theme:                po.Theme,
		PageSize:             po.PageSize,
		ReadingPanePosition:  po.ReadingPanePosition,
		AutoReplyEnabled:     po.AutoReplyEnabled,
		AutoReplySubject:     po.AutoReplySubject,
		AutoReplyBody:        po.AutoReplyBody,
		Phone:                po.Phone,
		Company:              po.Company,
		JobTitle:             po.JobTitle,
		NotificationSound:    po.NotificationSound,
		DesktopNotification:  po.DesktopNotification,
		IncludeOriginalOnReply: po.IncludeOriginalOnReply,
		ForwardingEnabled:    po.ForwardingEnabled,
		ForwardingAddress:    po.ForwardingAddress,
		SaveSent:             po.SaveSent,
		CreatedAt:            po.CreatedAt,
		UpdatedAt:            po.UpdatedAt,
	}
}

func userSettingsToPO(s *profile.UserSettings) *UserSettingsPO {
	if s == nil {
		return nil
	}
	return &UserSettingsPO{
		UserID:               s.UserID,
		DisplayName:          s.DisplayName,
		Signature:            s.Signature,
		Language:             s.Language,
		Theme:                s.Theme,
		PageSize:             s.PageSize,
		ReadingPanePosition:  s.ReadingPanePosition,
		AutoReplyEnabled:     s.AutoReplyEnabled,
		AutoReplySubject:     s.AutoReplySubject,
		AutoReplyBody:        s.AutoReplyBody,
		Phone:                s.Phone,
		Company:              s.Company,
		JobTitle:             s.JobTitle,
		NotificationSound:    s.NotificationSound,
		DesktopNotification:  s.DesktopNotification,
		IncludeOriginalOnReply: s.IncludeOriginalOnReply,
		ForwardingEnabled:    s.ForwardingEnabled,
		ForwardingAddress:    s.ForwardingAddress,
		SaveSent:             s.SaveSent,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

type UserSettingsRepository struct {
	db persistence.DBProvider
}

func NewUserSettingsRepository(db persistence.DBProvider) *UserSettingsRepository {
	return &UserSettingsRepository{db: db}
}

func (r *UserSettingsRepository) Get(ctx context.Context, userID shared.GlobalID) (*profile.UserSettings, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po UserSettingsPO
	if err := g.Where("user_id = ?", userID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, profile.NewErrSettings("settings not found")
		}
		return nil, err
	}
	return poToUserSettings(&po), nil
}

func (r *UserSettingsRepository) Save(ctx context.Context, settings *profile.UserSettings) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Save(userSettingsToPO(settings)).Error
}