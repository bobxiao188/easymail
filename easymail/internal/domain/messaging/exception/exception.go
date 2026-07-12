package exception

import "errors"

var (
	ErrNotFound              = errors.New("message not found")
	ErrForbidden             = errors.New("message: forbidden")
	ErrInvalidArgument       = errors.New("message: invalid argument")
	ErrPurgeNotAllowed       = errors.New("purge not allowed: message must be soft-deleted first")
	ErrFolderSystem          = errors.New("folder: system folder, cannot modify or delete")
	ErrFolderNotEmpty        = errors.New("folder: not empty")
	ErrAlreadyExists         = errors.New("folder: already exists")
	ErrComposeNoRecipient    = errors.New("compose: no recipient")
	ErrComposeExternalRecipient = errors.New("compose: external recipient not allowed")
	ErrComposeAddress        = errors.New("compose: invalid address")
	ErrComposeEmptyBody      = errors.New("compose: empty body")
	
	// Send errors
	ErrComposeSMTPConnect    = errors.New("compose: SMTP connection failed")
	ErrComposeSMTPSend       = errors.New("compose: SMTP send failed")
	ErrComposeRecipientNotFound = errors.New("compose: recipient not found")
	ErrComposeSaveFailed     = errors.New("compose: failed to save message")
	ErrComposeWriteFile      = errors.New("compose: failed to write message file")
)
