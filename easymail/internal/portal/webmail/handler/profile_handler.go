package handler

import (
	"easymail/internal/app/webmail"
	"easymail/internal/domain/management"
	"easymail/internal/portal/webmail/middleware"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"
	"errors"

	"github.com/gin-gonic/gin"
)

// GetSettings returns the current user profile settings.
func (h *Handler) GetSettings(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	settings, err := h.profileService.GetSettings(c.Request.Context(), aid)
	if err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}
	response.Success(c, settings)
}

type updateSettingsReq struct {
	DisplayName            *string `json:"displayName,omitempty"`
	Signature              *string `json:"signature,omitempty"`
	Language               *string `json:"language,omitempty"`
	Theme                  *string `json:"theme,omitempty"`
	PageSize               *int    `json:"pageSize,omitempty"`
	ReadingPanePosition    *string `json:"readingPanePosition,omitempty"`
	AutoReplyEnabled       *bool   `json:"autoReplyEnabled,omitempty"`
	AutoReplySubject       *string `json:"autoReplySubject,omitempty"`
	AutoReplyBody          *string `json:"autoReplyBody,omitempty"`
	Phone                  *string `json:"phone,omitempty"`
	Company                *string `json:"company,omitempty"`
	JobTitle               *string `json:"jobTitle,omitempty"`
	NotificationSound      *bool   `json:"notificationSound,omitempty"`
	DesktopNotification    *bool   `json:"desktopNotification,omitempty"`
	IncludeOriginalOnReply *bool   `json:"includeOriginalOnReply,omitempty"`
	ForwardingEnabled      *bool   `json:"forwardingEnabled,omitempty"`
	ForwardingAddress      *string `json:"forwardingAddress,omitempty"`
	SaveSent               *bool   `json:"saveSent,omitempty"`
}

// UpdateSettings updates user profile settings (partial update allowed).
func (h *Handler) UpdateSettings(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	var req updateSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	in := webmail.UpdateSettingsInput{
		DisplayName:            req.DisplayName,
		Signature:              req.Signature,
		Language:               req.Language,
		Theme:                  req.Theme,
		PageSize:               req.PageSize,
		ReadingPanePosition:    req.ReadingPanePosition,
		AutoReplyEnabled:       req.AutoReplyEnabled,
		AutoReplySubject:       req.AutoReplySubject,
		AutoReplyBody:          req.AutoReplyBody,
		Phone:                  req.Phone,
		Company:                req.Company,
		JobTitle:               req.JobTitle,
		NotificationSound:      req.NotificationSound,
		DesktopNotification:    req.DesktopNotification,
		IncludeOriginalOnReply: req.IncludeOriginalOnReply,
		ForwardingEnabled:      req.ForwardingEnabled,
		ForwardingAddress:      req.ForwardingAddress,
		SaveSent:               req.SaveSent,
	}
	settings, err := h.profileService.UpdateSettings(c.Request.Context(), aid, in)
	if err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}
	response.Success(c, settings)
}

// GetProfile returns the current user profile information.
func (h *Handler) GetProfile(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	// Get user settings which includes display name
	settings, err := h.profileService.GetSettings(c.Request.Context(), aid)
	if err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}

	// Get email from context
	email, _ := c.Get("webmail_email")

	response.Success(c, gin.H{
		"id":            aid.String(),
		"email":         email,
		"name":          settings.DisplayName,
		"phone":         settings.Phone,
		"company":       settings.Company,
		"job_title":     settings.JobTitle,
		"storage_used":  0,          // TODO: implement storage tracking
		"storage_limit": 1073741824, // 1GB default
	})
}

// UpdateProfile updates the current user profile information.
func (h *Handler) UpdateProfile(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	var req struct {
		Name     *string `json:"name,omitempty"`
		Phone    *string `json:"phone,omitempty"`
		Company  *string `json:"company,omitempty"`
		JobTitle *string `json:"job_title,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	// Build update input
	in := webmail.UpdateSettingsInput{}
	if req.Name != nil {
		in.DisplayName = req.Name
	}
	if req.Phone != nil {
		in.Phone = req.Phone
	}
	if req.Company != nil {
		in.Company = req.Company
	}
	if req.JobTitle != nil {
		in.JobTitle = req.JobTitle
	}

	_, err := h.profileService.UpdateSettings(c.Request.Context(), aid, in)
	if err != nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		return
	}

	response.Success(c, nil)
}

// ChangePassword changes the current user's password.
func (h *Handler) ChangePassword(c *gin.Context) {
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}

	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}

	// Get email from context
	email, ok := c.Get("webmail_email")
	if !ok || email == "" {
		response.Unauthorized(c, "")
		return
	}

	// Use auth service to change password
	err := h.authService.ChangePassword(c.Request.Context(), email.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, management.ErrMailUserNotFound),
			errors.Is(err, management.ErrMailUserDomainInvalid):
			response.Unauthorized(c, appi18n.Message(c, appi18n.KeyWebmailAuthInvalidCredentials))
		case errors.Is(err, management.ErrMailUserInvalidPass):
			response.Unauthorized(c, appi18n.Message(c, appi18n.KeyWebmailAuthOldPasswordIncorrect))
		default:
			response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
		}
		return
	}

	response.Success(c, nil)
}
