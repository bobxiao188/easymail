package webmail

import (
	"context"

	"easymail/internal/domain/profile"
	"easymail/internal/domain/shared"
)

type profileServiceImpl struct {
	settingsRepo profile.UserSettingsRepository
}

func NewProfileService(settingsRepo profile.UserSettingsRepository) ProfileService {
	return &profileServiceImpl{settingsRepo: settingsRepo}
}

func (s *profileServiceImpl) GetSettings(ctx context.Context, userID shared.GlobalID) (*UserSettingsDTO, error) {
	settings, err := s.settingsRepo.Get(ctx, userID)
	if err != nil {
		// If not found, create defaults
		if _, ok := err.(*profile.ErrSettings); ok {
			defaults := profile.NewUserSettings(userID)
			if saveErr := s.settingsRepo.Save(ctx, defaults); saveErr != nil {
				return nil, saveErr
			}
			return settingsToDTO(defaults), nil
		}
		return nil, err
	}
	return settingsToDTO(settings), nil
}

func (s *profileServiceImpl) UpdateSettings(ctx context.Context, userID shared.GlobalID, input UpdateSettingsInput) (*UserSettingsDTO, error) {
	settings, err := s.settingsRepo.Get(ctx, userID)
	if err != nil {
		if _, ok := err.(*profile.ErrSettings); ok {
			settings = profile.NewUserSettings(userID)
		} else {
			return nil, err
		}
	}

	if input.DisplayName != nil {
		settings.UpdateDisplayName(*input.DisplayName)
	}
	if input.Signature != nil {
		settings.UpdateSignature(*input.Signature)
	}
	if input.Language != nil || input.Theme != nil {
		lang := settings.Language
		if input.Language != nil {
			lang = *input.Language
		}
		thm := settings.Theme
		if input.Theme != nil {
			thm = *input.Theme
		}
		settings.UpdateAppearance(lang, thm)
	}
	if input.PageSize != nil || input.ReadingPanePosition != nil {
		ps := settings.PageSize
		if input.PageSize != nil {
			ps = *input.PageSize
		}
		rp := settings.ReadingPanePosition
		if input.ReadingPanePosition != nil {
			rp = *input.ReadingPanePosition
		}
		settings.UpdateMailPrefs(ps, rp)
	}
	if input.AutoReplyEnabled != nil || input.AutoReplySubject != nil || input.AutoReplyBody != nil {
		enabled := settings.AutoReplyEnabled
		if input.AutoReplyEnabled != nil {
			enabled = *input.AutoReplyEnabled
		}
		subj := settings.AutoReplySubject
		if input.AutoReplySubject != nil {
			subj = *input.AutoReplySubject
		}
		body := settings.AutoReplyBody
		if input.AutoReplyBody != nil {
			body = *input.AutoReplyBody
		}
		settings.UpdateAutoReply(enabled, subj, body)
	}


	if input.Phone != nil {
		settings.Phone = *input.Phone
	}
	if input.Company != nil {
		settings.Company = *input.Company
	}
	if input.JobTitle != nil {
		settings.JobTitle = *input.JobTitle
	}
	if input.NotificationSound != nil {
		settings.NotificationSound = *input.NotificationSound
	}
	if input.DesktopNotification != nil {
		settings.DesktopNotification = *input.DesktopNotification
	}
	if input.IncludeOriginalOnReply != nil {
		settings.IncludeOriginalOnReply = *input.IncludeOriginalOnReply
	}
	if input.ForwardingEnabled != nil {
		settings.ForwardingEnabled = *input.ForwardingEnabled
	}
	if input.ForwardingAddress != nil {
		settings.ForwardingAddress = *input.ForwardingAddress
	}
	if input.SaveSent != nil {
		settings.SaveSent = *input.SaveSent
	}

	if err := s.settingsRepo.Save(ctx, settings); err != nil {
		return nil, err
	}
	return settingsToDTO(settings), nil
}

func settingsToDTO(s *profile.UserSettings) *UserSettingsDTO {
	if s == nil {
		return nil
	}
	return &UserSettingsDTO{
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
	}
}
