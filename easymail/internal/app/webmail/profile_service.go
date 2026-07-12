package webmail

import (
	"context"
	"easymail/internal/domain/shared"
)

// UserSettingsDTO is the API response for user profile/settings.
type UserSettingsDTO struct {
	DisplayName          string `json:"displayName"`
	Signature            string `json:"signature"`
	Language             string `json:"language"`
	Theme                string `json:"theme"`
	PageSize             int    `json:"pageSize"`
	ReadingPanePosition  string `json:"readingPanePosition"`
	AutoReplyEnabled     bool   `json:"autoReplyEnabled"`
	AutoReplySubject     string `json:"autoReplySubject"`
	AutoReplyBody        string `json:"autoReplyBody"`
	Phone                string `json:"phone"`
	Company              string `json:"company"`
	JobTitle             string `json:"jobTitle"`
	NotificationSound    bool   `json:"notificationSound"`
	DesktopNotification  bool   `json:"desktopNotification"`
	IncludeOriginalOnReply bool `json:"includeOriginalOnReply"`
	ForwardingEnabled    bool   `json:"forwardingEnabled"`
	ForwardingAddress    string `json:"forwardingAddress"`
	SaveSent             bool   `json:"saveSent"`
}

// UpdateSettingsInput is the request payload for updating settings.
type UpdateSettingsInput struct {
	DisplayName          *string `json:"displayName,omitempty"`
	Signature            *string `json:"signature,omitempty"`
	Language             *string `json:"language,omitempty"`
	Theme                *string `json:"theme,omitempty"`
	PageSize             *int    `json:"pageSize,omitempty"`
	ReadingPanePosition  *string `json:"readingPanePosition,omitempty"`
	AutoReplyEnabled     *bool   `json:"autoReplyEnabled,omitempty"`
	AutoReplySubject     *string `json:"autoReplySubject,omitempty"`
	AutoReplyBody        *string `json:"autoReplyBody,omitempty"`
	Phone                *string `json:"phone,omitempty"`
	Company              *string `json:"company,omitempty"`
	JobTitle             *string `json:"jobTitle,omitempty"`
	NotificationSound    *bool   `json:"notificationSound,omitempty"`
	DesktopNotification  *bool   `json:"desktopNotification,omitempty"`
	IncludeOriginalOnReply *bool `json:"includeOriginalOnReply,omitempty"`
	ForwardingEnabled    *bool   `json:"forwardingEnabled,omitempty"`
	ForwardingAddress    *string `json:"forwardingAddress,omitempty"`
	SaveSent             *bool   `json:"saveSent,omitempty"`
}

// ProfileService handles user profile and settings operations.
type ProfileService interface {
	GetSettings(ctx context.Context, userID shared.GlobalID) (*UserSettingsDTO, error)
	UpdateSettings(ctx context.Context, userID shared.GlobalID, input UpdateSettingsInput) (*UserSettingsDTO, error)
}
