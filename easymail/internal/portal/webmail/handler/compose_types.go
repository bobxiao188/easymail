package handler

import (
	"strings"

	mailservice "easymail/internal/domain/messaging/service"
)

// ComposeRequest is the unified input for SaveDraft, UpdateDraft, and SendEmail.
// All fields use camelCase JSON to match the frontend ComposeMessage interface.
type ComposeRequest struct {
	Subject     string            `json:"subject"`
	Text        string            `json:"text"`
	HTML        string            `json:"html"`
	To          []Recipient       `json:"to"`
	Cc          []Recipient       `json:"cc"`
	Bcc         []Recipient       `json:"bcc"`
	From        *Recipient        `json:"from"`
	FolderID    *int64            `json:"folderId"`
	SaveSent    *bool             `json:"saveSent"`
	Attachments []AttachmentInput `json:"attachments"`
	Signature   string            `json:"signature"`
	DraftID     *int64            `json:"draftId,omitempty"`
}

// DraftDetail is the unified response for EditDraft (GET /draft/:id).
// It mirrors ComposeRequest fields but is enriched with metadata
// from the stored email entity, using camelCase JSON.
type DraftDetail struct {
	ID          int64             `json:"id"`
	Subject     string            `json:"subject"`
	Text        string            `json:"text"`
	HTML        string            `json:"html,omitempty"`
	To          []Recipient       `json:"to"`
	Cc          []Recipient       `json:"cc"`
	Bcc         []Recipient       `json:"bcc"`
	From        *Recipient        `json:"from"`
	MailTime    string            `json:"mailTime"`
	FolderID    int64             `json:"folderId"`
	Attachments []AttachmentInput `json:"attachments,omitempty"`
}

// recipientsToStr joins a slice of Recipient into a comma-separated email string.
func recipientsToStr(list []Recipient) string {
	if len(list) == 0 {
		return ""
	}
	s := list[0].Email
	for i := 1; i < len(list); i++ {
		s += "," + list[i].Email
	}
	return s
}

// parseAddresses converts comma-separated email addresses to Recipient slice.
// Supports both "Name <email>" and bare "email" formats.
func parseAddresses(addrStr string) []Recipient {
	if addrStr == "" {
		return nil
	}

	addresses := make([]Recipient, 0)

	parts := strings.Split(addrStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var name, address string

		if strings.Contains(part, "<") && strings.Contains(part, ">") {
			name = strings.Trim(part[:strings.Index(part, "<")], " ")
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			address = strings.Trim(part[start+1:end], " ")
		} else {
			address = part
		}

		addresses = append(addresses, Recipient{
			Name:  name,
			Email: address,
		})
	}

	return addresses
}

// senderToRecipient extracts a *Recipient from the sender email string.
// Returns nil when sender is empty.
func senderToRecipient(sender string) *Recipient {
	if sender == "" {
		return nil
	}
	return &Recipient{Email: sender}
}

// toServiceRecipients converts handler Recipient slice to mailservice Recipient slice.
func toServiceRecipients(list []Recipient) []mailservice.Recipient {
	if len(list) == 0 {
		return nil
	}
	out := make([]mailservice.Recipient, 0, len(list))
	for _, r := range list {
		out = append(out, mailservice.Recipient{Name: r.Name, Email: r.Email})
	}
	return out
}
