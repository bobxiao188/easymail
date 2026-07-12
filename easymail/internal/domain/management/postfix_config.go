// Package management contains the bounded context for admin operations.
// internal/domain/management/postfix_config.go - Postfix configuration entities

package management

import (
	"context"
	"errors"
	"time"

	"easymail/internal/domain/shared"
)

// Sentinel errors for PostfixConfig operations.
var (
	ErrPostfixConfigNotFound      = errors.New("postfix config parameter not found")
	ErrPostfixConfigDuplicate     = errors.New("postfix config parameter already exists")
	ErrPostfixConfigNotEditable   = errors.New("postfix config parameter is system-managed and read-only")
	ErrPostfixDeliveryLogNotFound = errors.New("postfix delivery log not found")
)

// PostfixConfigCategory enumerates configuration categories.
type PostfixConfigCategory string

const (
	ConfigCategoryMain PostfixConfigCategory = "main"
)

// PostfixConfig represents a single main.cf parameter managed from admin.
type PostfixConfig struct {
	ID          shared.GlobalID `json:"id"`
	ParamName   string          `json:"paramName"`
	ParamValue  string          `json:"paramValue"`
	Category    string          `json:"category"`
	IsManaged   bool            `json:"isManaged"`
	Enabled     bool            `json:"enabled"`
	Description string          `json:"description,omitempty"`
	SortOrder   int             `json:"sortOrder"`
	CreateTime  time.Time       `json:"createTime"`
	UpdateTime  time.Time       `json:"updateTime"`
}

// NewPostfixConfig creates a new user-defined config parameter.
func NewPostfixConfig(paramName, paramValue, description string) (*PostfixConfig, error) {
	if paramName == "" {
		return nil, errors.New("param_name is required")
	}
	now := time.Now()
	return &PostfixConfig{
		ID:          shared.NewGlobalID(),
		ParamName:   paramName,
		ParamValue:  paramValue,
		Category:    string(ConfigCategoryMain),
		IsManaged:   false,
		Enabled:     true,
		Description: description,
		SortOrder:   0,
		CreateTime:  now,
		UpdateTime:  now,
	}, nil
}

// PostfixConfigRepository port
type PostfixConfigRepository interface {
	Save(ctx context.Context, cfg *PostfixConfig) error
	FindByID(ctx context.Context, id shared.GlobalID) (*PostfixConfig, error)
	FindByParamName(ctx context.Context, paramName string) (*PostfixConfig, error)
	FindAllManaged(ctx context.Context) ([]PostfixConfig, error)
	FindAllUserDefined(ctx context.Context) ([]PostfixConfig, error)
	FindAll(ctx context.Context) ([]PostfixConfig, error)
	Search(ctx context.Context, keyword string, page, pageSize int) ([]PostfixConfig, int64, error)
	Delete(ctx context.Context, id shared.GlobalID) error
	DeleteByCategory(ctx context.Context, category string) error
}

// DeliveryAction represents the type of delivery operation.
type DeliveryAction string

const (
	DeliveryActionPush     DeliveryAction = "push"
	DeliveryActionApply    DeliveryAction = "apply"
	DeliveryActionRollback DeliveryAction = "rollback"
)

// DeliveryStatus represents the result status.
type DeliveryStatus string

const (
	DeliveryStatusSuccess DeliveryStatus = "success"
	DeliveryStatusFailed  DeliveryStatus = "failed"
)

// PostfixDeliveryLog records a configuration delivery operation to a Postfix agent.
type PostfixDeliveryLog struct {
	ID             shared.GlobalID `json:"id"`
	AgentID        shared.GlobalID `json:"agentId"`
	Action         string          `json:"action"`
	Status         string          `json:"status"`
	ConfigSnapshot string          `json:"configSnapshot,omitempty"`
	ErrorMessage   string          `json:"errorMessage,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	AgentName      string          `json:"agentName,omitempty" gorm:"-"`
}

// PostfixDeliveryLogRepository port
type PostfixDeliveryLogRepository interface {
	Save(ctx context.Context, log *PostfixDeliveryLog) error
	FindByAgent(ctx context.Context, agentID shared.GlobalID, limit int) ([]PostfixDeliveryLog, error)
	Search(ctx context.Context, keyword string, page, pageSize int) ([]PostfixDeliveryLog, int64, error)
}
