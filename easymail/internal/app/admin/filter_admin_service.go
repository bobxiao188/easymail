package admin

import (
	"context"

	"easymail/internal/domain/filter/rule"
)

type FilterLogFilter struct {
	IP            string
	Sender        string
	Recipient     string
	CreatedAfter  *string
	CreatedBefore *string
}

// FilterAdminService defines operations for scanner/filter management.
type FilterAdminService interface {
	// Features
	ListBuiltinFeatures(ctx context.Context) ([]rule.BuiltinFeature, error)
	ListCustomFeatures(ctx context.Context) ([]rule.CustomFeature, error)
	GetCustomFeature(ctx context.Context, id int64) (*rule.CustomFeature, error)
	CreateCustomFeature(ctx context.Context, key, name, dataType string, fields []string) (*rule.CustomFeature, error)
	UpdateCustomFeature(ctx context.Context, id int64, key, name, dataType string, fields []string) error
	PatchCustomFeature(ctx context.Context, id int64, enabled bool) error
	DeleteCustomFeature(ctx context.Context, id int64) error

	// Rules
	ListRules(ctx context.Context, keyword string, page, pageSize int) ([]rule.Rule, int64, error)
	GetRule(ctx context.Context, id int64) (*rule.Rule, error)
	CreateRule(ctx context.Context, r *rule.Rule) error
	UpdateRule(ctx context.Context, r *rule.Rule) error
	PatchRule(ctx context.Context, id int64, enabled bool) error
	DeleteRule(ctx context.Context, id int64) error

	// Filter Logs
	ListFilterLogs(ctx context.Context, filter *FilterLogFilter, page, pageSize int) ([]rule.FilterLog, int64, error)
	GetFilterLog(ctx context.Context, id int64) (*rule.FilterLog, error)

	// Condition validation
	ValidateConditionJSON(jsonStr string) error
	SetRuleStageFromCondition(ctx context.Context, r *rule.Rule) error
	InvalidateFilterRulesCache()
	InvalidateClassifyModelsCache()
	FeatureKeyReserved(ctx context.Context, featureKey string) (bool, error)
}
