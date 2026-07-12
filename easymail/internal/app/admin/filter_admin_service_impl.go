package admin

import (
	"context"
	"time"

	filtersvc "easymail/internal/app/filter"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

type filterAdminServiceImpl struct {
	db *gorm.DB
}

func NewFilterAdminService(db *gorm.DB) FilterAdminService {
	return &filterAdminServiceImpl{db: db}
}

func (s *filterAdminServiceImpl) ListBuiltinFeatures(ctx context.Context) ([]rule.BuiltinFeature, error) {
	return mysql.ListFeatureDefs(ctx, s.db)
}

func (s *filterAdminServiceImpl) ListCustomFeatures(ctx context.Context) ([]rule.CustomFeature, error) {
	return mysql.ListCustomFeatureDefs(ctx, s.db)
}

func (s *filterAdminServiceImpl) GetCustomFeature(ctx context.Context, id int64) (*rule.CustomFeature, error) {
	return mysql.GetCustomFeatureDef(ctx, s.db, id)
}

func (s *filterAdminServiceImpl) CreateCustomFeature(ctx context.Context, key, name, dataType string, fields []string) (*rule.CustomFeature, error) {
	f := &rule.CustomFeature{
		FeatureKey: key,
		Label:      name,
		Type:       dataType,
	}
	if err := mysql.CreateCustomFeatureDef(ctx, s.db, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *filterAdminServiceImpl) UpdateCustomFeature(ctx context.Context, id int64, key, name, dataType string, fields []string) error {
	f, err := mysql.GetCustomFeatureDef(ctx, s.db, id)
	if err != nil {
		return err
	}
	f.FeatureKey = key
	f.Label = name
	f.Type = dataType
	return mysql.SaveCustomFeatureDef(ctx, s.db, f)
}

func (s *filterAdminServiceImpl) PatchCustomFeature(ctx context.Context, id int64, enabled bool) error {
	f, err := mysql.GetCustomFeatureDef(ctx, s.db, id)
	if err != nil {
		return err
	}
	f.Enabled = enabled
	return mysql.SaveCustomFeatureDef(ctx, s.db, f)
}

func (s *filterAdminServiceImpl) DeleteCustomFeature(ctx context.Context, id int64) error {
	return mysql.DeleteCustomFeatureDef(ctx, s.db, id)
}

func (s *filterAdminServiceImpl) ListRules(ctx context.Context, keyword string, page, pageSize int) ([]rule.Rule, int64, error) {
	total, rows, err := mysql.ListRulesPaged(ctx, s.db, page, pageSize)
	return rows, total, err
}

func (s *filterAdminServiceImpl) GetRule(ctx context.Context, id int64) (*rule.Rule, error) {
	return mysql.GetRule(ctx, s.db, id)
}

func (s *filterAdminServiceImpl) CreateRule(ctx context.Context, r *rule.Rule) error {
	return mysql.CreateRule(ctx, s.db, r)
}

func (s *filterAdminServiceImpl) UpdateRule(ctx context.Context, r *rule.Rule) error {
	return mysql.SaveRule(ctx, s.db, r)
}

func (s *filterAdminServiceImpl) PatchRule(ctx context.Context, id int64, enabled bool) error {
	r, err := mysql.GetRule(ctx, s.db, id)
	if err != nil {
		return err
	}
	r.Enabled = enabled
	return mysql.SaveRule(ctx, s.db, r)
}

func (s *filterAdminServiceImpl) DeleteRule(ctx context.Context, id int64) error {
	return mysql.DeleteRule(ctx, s.db, id)
}

func (s *filterAdminServiceImpl) ListFilterLogs(ctx context.Context, filter *FilterLogFilter, page, pageSize int) ([]rule.FilterLog, int64, error) {
	f := &mysql.FilterLogFilter{}
	if filter != nil {
		f.IP = filter.IP
		f.Sender = filter.Sender
		f.Recipient = filter.Recipient
		if filter.CreatedAfter != nil && *filter.CreatedAfter != "" {
			t, err := parseLogTime(*filter.CreatedAfter, true)
			if err == nil {
				f.CreatedAfter = &t
			}
		}
		if filter.CreatedBefore != nil && *filter.CreatedBefore != "" {
			t, err := parseLogTime(*filter.CreatedBefore, false)
			if err == nil {
				f.CreatedBefore = &t
			}
		}
	}
	total, rows, err := mysql.ListFilterLogs(ctx, s.db, page, pageSize, f)
	return rows, total, err
}

func (s *filterAdminServiceImpl) GetFilterLog(ctx context.Context, id int64) (*rule.FilterLog, error) {
	return mysql.GetFilterLog(ctx, s.db, id)
}

func (s *filterAdminServiceImpl) ValidateConditionJSON(jsonStr string) error {
	return filtersvc.ValidateConditionJSON(jsonStr)
}

func (s *filterAdminServiceImpl) SetRuleStageFromCondition(ctx context.Context, r *rule.Rule) error {
	return filtersvc.SetRuleStageFromCondition(ctx, s.db, r)
}

func (s *filterAdminServiceImpl) InvalidateFilterRulesCache() {
	cache.InvalidateFilterRulesCache()
}

func (s *filterAdminServiceImpl) InvalidateClassifyModelsCache() {
	cache.InvalidateClassifyModelsCache()
}

func parseLogTime(s string, startWhenDateOnly bool) (time.Time, error) {
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
		if err != nil {
			return time.Time{}, err
		}
		if startWhenDateOnly {
			return t, nil
		}
		return t.Add(24*time.Hour - time.Nanosecond), nil
	}
	return time.Parse(time.RFC3339, s)
}

func (s *filterAdminServiceImpl) FeatureKeyReserved(ctx context.Context, featureKey string) (bool, error) {
	return filtersvc.FeatureKeyReservedByClassifyModel(ctx, s.db, featureKey)
}

var _ FilterAdminService = (*filterAdminServiceImpl)(nil)
