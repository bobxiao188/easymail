package profile

import (
	"context"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors
var (
	ErrSettingsNotFound   = NewErrSettings("settings not found")
	ErrSettingsInvalidArg = NewErrSettings("invalid argument")
)

// ErrSettings is a domain error for the profile/settings context.
type ErrSettings struct{ msg string }

func NewErrSettings(msg string) error       { return &ErrSettings{msg: msg} }
func (e *ErrSettings) Error() string        { return e.msg }
func (e *ErrSettings) Is(target error) bool { return target != nil && target.Error() == e.msg }

// UserSettings represents per-user preferences and profile information.
type UserSettings struct {
	UserID                 shared.GlobalID `json:"user_id"`
	DisplayName            string          `json:"display_name"`
	Signature              string          `json:"signature"`
	Language               string          `json:"language"`
	Theme                  string          `json:"theme"`
	PageSize               int             `json:"page_size"`
	ReadingPanePosition    string          `json:"reading_pane_position"` // "right" or "bottom"
	AutoReplyEnabled       bool            `json:"auto_reply_enabled"`
	AutoReplySubject       string          `json:"auto_reply_subject"`
	AutoReplyBody          string          `json:"auto_reply_body"`
	Phone                  string          `json:"phone"`
	Company                string          `json:"company"`
	JobTitle               string          `json:"job_title"`
	NotificationSound      bool            `json:"notification_sound"`
	DesktopNotification    bool            `json:"desktop_notification"`
	IncludeOriginalOnReply bool            `json:"include_original_on_reply"`
	ForwardingEnabled      bool            `json:"forwarding_enabled"`
	ForwardingAddress      string          `json:"forwarding_address"`
	SaveSent               bool            `json:"save_sent"` // Save copy to Sent folder after sending (default: true)
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

// NewUserSettings creates UserSettings with defaults for a new user.
func NewUserSettings(userID shared.GlobalID) *UserSettings {
	now := time.Now()
	return &UserSettings{
		UserID:                 userID,
		DisplayName:            "",
		Signature:              "",
		Language:               "en",
		Theme:                  "blue",
		PageSize:               20,
		ReadingPanePosition:    "right",
		AutoReplyEnabled:       false,
		AutoReplySubject:       "",
		AutoReplyBody:          "",
		Phone:                  "",
		Company:                "",
		JobTitle:               "",
		NotificationSound:      true,
		DesktopNotification:    true,
		IncludeOriginalOnReply: true,
		ForwardingEnabled:      false,
		ForwardingAddress:      "",
		SaveSent:               true,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func (s *UserSettings) UpdateDisplayName(name string) {
	s.DisplayName = name
	s.UpdatedAt = time.Now()
}

func (s *UserSettings) UpdateSignature(sig string) {
	s.Signature = sig
	s.UpdatedAt = time.Now()
}

func (s *UserSettings) UpdateAppearance(language, theme string) {
	if language != "" {
		s.Language = language
	}
	if theme != "" {
		s.Theme = theme
	}
	s.UpdatedAt = time.Now()
}

func (s *UserSettings) UpdateMailPrefs(pageSize int, readingPane string) {
	if pageSize >= 0 {
		s.PageSize = pageSize
	}
	if readingPane != "" {
		s.ReadingPanePosition = readingPane
	}
	s.UpdatedAt = time.Now()
}

func (s *UserSettings) UpdateAutoReply(enabled bool, subject, body string) {
	s.AutoReplyEnabled = enabled
	s.AutoReplySubject = subject
	s.AutoReplyBody = body
	s.UpdatedAt = time.Now()
}

// UserSettingsRepository port (per-user database)
type UserSettingsRepository interface {
	Get(ctx context.Context, userID shared.GlobalID) (*UserSettings, error)
	Save(ctx context.Context, settings *UserSettings) error
}
