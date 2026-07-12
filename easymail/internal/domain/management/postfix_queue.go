// Package management contains the bounded context for admin operations.
// internal/domain/management/postfix_queue.go - Postfix queue management entities

package management

import (
	"easymail/internal/domain/shared"
)

// QueueMessage represents a single message in the Postfix queue.
type QueueMessage struct {
	QueueID    string   `json:"queueId"`    // Queue ID (message ID)
	Size       int      `json:"size"`       // Message size in bytes
	Age        string   `json:"age"`        // Time in queue (e.g., "0:12", "1:23")
	Sender     string   `json:"sender"`     // Sender email address
	Recipients []string `json:"recipients"` // Recipient email addresses
	Status     string   `json:"status"`     // Queue status: active, deferred, held
	StatusText string   `json:"statusText"` // Human-readable status description
}

// QueueStats provides summary statistics for the mail queue.
type QueueStats struct {
	Total    int `json:"total"`    // Total messages in queue
	Active   int `json:"active"`   // Messages in active queue
	Deferred int `json:"deferred"` // Messages in deferred queue
	Held     int `json:"held"`     // Messages held (censored)
}

// QueueFilter specifies filters for querying the queue.
type QueueFilter struct {
	Status     string `json:"status"`      // Filter by status: active, deferred, held, all
	Sender     string `json:"sender"`      // Filter by sender email
	Recipient  string `json:"recipient"`   // Filter by recipient email
	QueueID    string `json:"queueId"`     // Filter by specific queue ID
	Page       int    `json:"page"`        // Page number (1-based)
	PageSize   int    `json:"pageSize"`    // Number of items per page
}

// QueueListResponse contains the paginated list of queue messages.
type QueueListResponse struct {
	Messages []QueueMessage `json:"messages"` // List of messages
	Total    int            `json:"total"`    // Total number of messages matching filter
	Page     int            `json:"page"`     // Current page number
	PageSize int            `json:"pageSize"` // Number of items per page
}

// QueueActionRequest contains the request for queue operations.
type QueueActionRequest struct {
	MessageIDs []string `json:"messageIds"` // List of message IDs to operate on
	AgentID    shared.GlobalID
}
