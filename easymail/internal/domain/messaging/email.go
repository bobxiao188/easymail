package messaging

import (
	"easymail/internal/domain/shared"

	"easymail/pkg/constants"
)

// SMTP delivery status constants.
const (
	SMTPStatusPending = "pending"
	SMTPStatusSent    = "sent"
	SMTPStatusFailed  = "failed"
)

// Email represents a stored email message.
type Email struct {
	ID             int64
	MailUserID     shared.GlobalID
	JobID          string
	QueueID        string
	Subject        string
	Sender         string
	Recipient      string
	CarbonCopy     string
	BlindCopy      string
	MailTime       string
	FolderID       int64
	ReadStatus     constants.ReadStatus
	Flagged        bool
	HasAttachments bool   // 是否有附件
	Body           string
	Snippet        string // 邮件摘要，用于列表显示
	MailSize       int64
	IsDeleted      bool
	FilePath       string
	// SMTP delivery status
	SMTPStatus string // "pending", "sent", "failed"
	SMTPError  string // Error message if delivery failed
	SMTPSentAt string // When the email was sent via SMTP (RFC3339)
}


// Attachment value object
type Attachment struct {
	Name        string
	ContentType string
	Data        []byte
}

