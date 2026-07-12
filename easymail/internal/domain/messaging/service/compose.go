package service

import "easymail/internal/domain/messaging"

// Recipient represents a structured recipient with name and email
type Recipient struct {
	Name  string
	Email string
}

// ComposeRequest holds data for webmail compose.
type ComposeRequest struct {
	ID             int64                   // Draft ID for update; 0 means create new
	From           *Recipient              // Optional sender override
	To             string                  // Comma-separated emails
	Cc             string                  // Comma-separated emails
	Bcc            string                  // Comma-separated emails
	ToRecipients   []Recipient             // Structured To recipients (for encoding)
	CcRecipients   []Recipient             // Structured Cc recipients (for encoding)
	BccRecipients  []Recipient             // Structured Bcc recipients (for encoding)
	Subject        string
	Text           string
	HTML           string
	Attachments    []messaging.Attachment
	FolderID       int64  // Target folder (used for draft save); 0 means unspecified
	SaveSent       bool   // Save copy to Sent folder after sending (default: true)
	Signature      string // User signature to append (optional)
}
