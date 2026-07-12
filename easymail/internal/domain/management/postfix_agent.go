// Package management contains the bounded context for admin operations.
// internal/domain/management/postfix_agent.go - PostfixAgent aggregate root

package management

import (
	"context"
	"errors"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors for PostfixAgent operations.
var (
	ErrPostfixAgentNotFound   = errors.New("postfix agent not found")
	ErrPostfixAgentExists     = errors.New("postfix agent already exists")
	ErrPostfixAgentOffline    = errors.New("postfix agent is offline")
	ErrPostfixAgentTokenEmpty = errors.New("agent token is required")
)

// AgentStatus represents the operational status of a Postfix agent.
type AgentStatus string

const (
	AgentStatusUnknown AgentStatus = "unknown"
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
	AgentStatusError   AgentStatus = "error"
)

// PostfixAgent represents a registered Postfix server managed via its agent.
type PostfixAgent struct {
	ID          shared.GlobalID `json:"id"`
	Name        string          `json:"name"`
	Host        string          `json:"host"`
	Token       string          `json:"token,omitempty"`
	Enabled     bool            `json:"enabled"`
	LastStatus  string          `json:"lastStatus,omitempty"`
	LastSyncAt  time.Time       `json:"lastSyncAt,omitempty"`
	Description string          `json:"description,omitempty"`
	CreateTime  time.Time       `json:"createTime"`
	UpdateTime  time.Time       `json:"updateTime"`
}

// AgentStatusInfo holds the live status returned from an agent.
type AgentStatusInfo struct {
	PostfixRunning  bool   `json:"postfixRunning"`
	ConfigHash      string `json:"configHash"`
	LastReloadAt    string `json:"lastReloadAt,omitempty"`
	PostfixVersion  string `json:"postfixVersion,omitempty"`
	AgentVersion    string `json:"agentVersion,omitempty"`
	Uptime          string `json:"uptime,omitempty"`
}

// NewPostfixAgent creates a new PostfixAgent.
func NewPostfixAgent(name, host, token, description string) (*PostfixAgent, error) {
	if name == "" || host == "" {
		return nil, errors.New("name and host are required")
	}
	if token == "" {
		return nil, ErrPostfixAgentTokenEmpty
	}
	now := time.Now()
	return &PostfixAgent{
		ID:          shared.NewGlobalID(),
		Name:        name,
		Host:        host,
		Token:       token,
		Enabled:     true,
		LastStatus:  string(AgentStatusUnknown),
		Description: description,
		CreateTime:  now,
		UpdateTime:  now,
	}, nil
}

// MarkOnline updates the agent's status to online.
func (a *PostfixAgent) MarkOnline() {
	a.LastStatus = string(AgentStatusOnline)
	a.UpdateTime = time.Now()
}

// MarkOffline updates the agent's status to offline.
func (a *PostfixAgent) MarkOffline() {
	a.LastStatus = string(AgentStatusOffline)
	a.UpdateTime = time.Now()
}

// PostfixAgentRepository port
type PostfixAgentRepository interface {
	Save(ctx context.Context, agent *PostfixAgent) error
	FindByID(ctx context.Context, id shared.GlobalID) (*PostfixAgent, error)
	FindAll(ctx context.Context) ([]PostfixAgent, error)
	FindEnabled(ctx context.Context) ([]PostfixAgent, error)
	Search(ctx context.Context, keyword string, page, pageSize int) ([]PostfixAgent, int64, error)
	Delete(ctx context.Context, id shared.GlobalID) error
	UpdateStatus(ctx context.Context, id shared.GlobalID, status string) error
}