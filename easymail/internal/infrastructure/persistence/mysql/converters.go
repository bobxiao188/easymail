package mysql

import (
	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/classifier"
	"easymail/internal/domain/filter/rule"
	"easymail/internal/domain/filter/stats"
	"encoding/json"
)

// ============================================================
// PO ↔ Domain Converters
// ============================================================

func poToBuiltinFeature(po *BuiltinFeaturePO) *rule.BuiltinFeature {
	if po == nil {
		return nil
	}
	return &rule.BuiltinFeature{
		ID:          po.ID,
		FeatureKey:  po.FeatureKey,
		Label:       po.Label,
		ValueType:   po.ValueType,
		Stage:       filter.Stage(po.Stage),
		Description: po.Description,
		Unit:        po.Unit,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func builtinFeatureToPO(f *rule.BuiltinFeature) *BuiltinFeaturePO {
	if f == nil {
		return nil
	}
	stg := int(f.Stage)
	return &BuiltinFeaturePO{
		ID:          f.ID,
		FeatureKey:  f.FeatureKey,
		Label:       f.Label,
		ValueType:   f.ValueType,
		Stage:       stg,
		Description: f.Description,
		Unit:        f.Unit,
	}
}

func poToRule(po *RulePO) *rule.Rule {
	if po == nil {
		return nil
	}
	return &rule.Rule{
		ID:            po.ID,
		Name:          po.Name,
		Enabled:       po.Enabled,
		Priority:      po.Priority,
		Stage:         (*filter.Stage)(po.Stage),
		Action:        filter.Outcome(po.Action),
		ConditionJSON: po.ConditionJSON,
		CreatedAt:     po.CreatedAt,
		UpdatedAt:     po.UpdatedAt,
		IsDeleted:     po.IsDeleted,
		CreatorId:     po.CreatorId,
	}
}

func ruleToPO(r *rule.Rule) *RulePO {
	if r == nil {
		return nil
	}
	var stg *int
	if r.Stage != nil {
		s := int(*r.Stage)
		stg = &s
	}
	return &RulePO{
		ID:            r.ID,
		Name:          r.Name,
		Enabled:       r.Enabled,
		Priority:      r.Priority,
		Stage:         stg,
		Action:        string(r.Action),
		ConditionJSON: r.ConditionJSON,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		IsDeleted:     r.IsDeleted,
		CreatorId:     r.CreatorId,
	}
}

func poToFilterLog(po *FilterLogPO) *rule.FilterLog {
	if po == nil {
		return nil
	}
	return &rule.FilterLog{
		ID:                  po.ID,
		TraceID:             po.TraceID,
		QueueID:             po.QueueID,
		IP:                  po.IP,
		Sender:              po.Sender,
		Recipient:           po.Recipient,
		Subject:             po.Subject,
		Stage:               filter.Stage(po.Stage),
		RuleID:              po.RuleID,
		ActionApplied:       filter.Outcome(po.ActionApplied),
		FeatureSnapshotJSON: po.FeatureSnapshotJSON,
		ConditionTraceJSON:  po.ConditionTraceJSON,
		DurationMs:          po.DurationMs,
		CreatedAt:           po.CreatedAt,
	}
}

func filterLogToPO(s *rule.FilterLog) *FilterLogPO {
	if s == nil {
		return nil
	}
	return &FilterLogPO{
		ID:                  s.ID,
		TraceID:             s.TraceID,
		QueueID:             s.QueueID,
		IP:                  s.IP,
		Sender:              s.Sender,
		Recipient:           s.Recipient,
		Subject:             s.Subject,
		Stage:               int(s.Stage),
		RuleID:              s.RuleID,
		ActionApplied:       string(s.ActionApplied),
		FeatureSnapshotJSON: s.FeatureSnapshotJSON,
		ConditionTraceJSON:  s.ConditionTraceJSON,
		DurationMs:          s.DurationMs,
	}
}

func poToCustomFeature(po *CustomFeaturePO) *rule.CustomFeature {
	if po == nil {
		return nil
	}
	return &rule.CustomFeature{
		ID:          po.ID,
		FeatureKey:  po.FeatureKey,
		Label:       po.Label,
		Stage:       filter.Stage(po.Stage),
		Type:        po.Type,
		ValueType:   po.ValueType,
		Enabled:     po.Enabled,
		SpecJSON:    po.SpecJSON,
		Description: po.Description,
		Unit:        po.Unit,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
		IsDeleted:   po.IsDeleted,
		CreatorId:   po.CreatorId,
	}
}

func customFeatureToPO(f *rule.CustomFeature) *CustomFeaturePO {
	if f == nil {
		return nil
	}
	return &CustomFeaturePO{
		ID:          f.ID,
		FeatureKey:  f.FeatureKey,
		Label:       f.Label,
		Stage:       int(f.Stage),
		Type:        f.Type,
		ValueType:   f.ValueType,
		Enabled:     f.Enabled,
		SpecJSON:    f.SpecJSON,
		Description: f.Description,
		Unit:        f.Unit,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
		IsDeleted:   f.IsDeleted,
		CreatorId:   f.CreatorId,
	}
}

func PoToClassifyModel(po *ClassifyModelPO) *classifier.Model {
	if po == nil {
		return nil
	}
	m := &classifier.Model{
		ID:            po.ID,
		Name:          po.Name,
		Algorithm:     classifier.Algorithm(po.Algorithm),
		Tokenizer:     classifier.Tokenizer(po.Tokenizer),
		SavePath:      po.SavePath,
		MaxTextLength: po.MaxTextLength,
		CreatedAt:     po.CreatedAt,
		UpdatedAt:     po.UpdatedAt,
		Enabled:       po.Enabled,
		TrainStatus:   classifier.TrainStatus(po.TrainStatus),
		TrainResult:   po.TrainResult,
		TrainTime:     po.TrainTime,
		IsDeleted:     po.IsDeleted,
		DeleteTime:    po.DeleteTime,
		CreatorId:     po.CreatorId,
	}
	_ = json.Unmarshal([]byte(po.Languages), &m.Languages)
	_ = json.Unmarshal([]byte(po.Params), &m.Params)
	_ = json.Unmarshal([]byte(po.EmailFields), &m.EmailFields)
	_ = json.Unmarshal([]byte(po.ClassLabels), &m.ClassLabels)
	return m
}

func ClassifyModelToPO(m *classifier.Model) *ClassifyModelPO {
	if m == nil {
		return nil
	}
	langJSON, _ := json.Marshal(m.Languages)
	paramsJSON, _ := json.Marshal(m.Params)
	fieldsJSON, _ := json.Marshal(m.EmailFields)
	labelsJSON, _ := json.Marshal(m.ClassLabels)
	return &ClassifyModelPO{
		ID:            m.ID,
		Name:          m.Name,
		Algorithm:     string(m.Algorithm),
		Tokenizer:     string(m.Tokenizer),
		Languages:     string(langJSON),
		SavePath:      m.SavePath,
		Params:        string(paramsJSON),
		MaxTextLength: m.MaxTextLength,
		EmailFields:   string(fieldsJSON),
		ClassLabels:   string(labelsJSON),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		Enabled:       m.Enabled,
		TrainStatus:   string(m.TrainStatus),
		TrainResult:   m.TrainResult,
		TrainTime:     m.TrainTime,
		IsDeleted:     m.IsDeleted,
		DeleteTime:    m.DeleteTime,
		CreatorId:     m.CreatorId,
	}
}

func PoToModelSample(po *ModelSamplePO) *classifier.Sample {
	if po == nil {
		return nil
	}
	return &classifier.Sample{
		ID:        po.ID,
		ModelID:   po.ClassifyModelID,
		Text:      po.Text,
		Label:     po.Label,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

func ModelSampleToPO(s *classifier.Sample) *ModelSamplePO {
	if s == nil {
		return nil
	}
	return &ModelSamplePO{
		ID:              s.ID,
		ClassifyModelID: s.ModelID,
		Text:            s.Text,
		Label:           s.Label,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

func poToFilterMailStatsDaily(po *FilterMailStatsDailyPO) *stats.FilterMailStatsDaily {
	if po == nil {
		return nil
	}
	return &stats.FilterMailStatsDaily{
		ID:            po.ID,
		StatDate:      po.StatDate,
		ActionApplied: po.ActionApplied,
		MailCount:     po.MailCount,
		CreatedAt:     po.CreatedAt,
		UpdatedAt:     po.UpdatedAt,
	}
}

func filterMailStatsDailyToPO(s *stats.FilterMailStatsDaily) *FilterMailStatsDailyPO {
	if s == nil {
		return nil
	}
	return &FilterMailStatsDailyPO{
		ID:            s.ID,
		StatDate:      s.StatDate,
		ActionApplied: s.ActionApplied,
		MailCount:     s.MailCount,
	}
}

func poToFilterStatsRollupWatermark(po *FilterStatsRollupWatermarkPO) *stats.FilterStatsRollupWatermark {
	if po == nil {
		return nil
	}
	return &stats.FilterStatsRollupWatermark{
		ID:                    po.ID,
		JobName:               po.JobName,
		LastCompletedStatDate: po.LastCompletedStatDate,
		UpdatedAt:             po.UpdatedAt,
	}
}

func filterStatsRollupWatermarkToPO(s *stats.FilterStatsRollupWatermark) *FilterStatsRollupWatermarkPO {
	if s == nil {
		return nil
	}
	return &FilterStatsRollupWatermarkPO{
		ID:                    s.ID,
		JobName:               s.JobName,
		LastCompletedStatDate: s.LastCompletedStatDate,
	}
}
