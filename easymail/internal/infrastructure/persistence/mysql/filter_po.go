package mysql

import (
	"time"

	"gorm.io/gorm"
)

// ============================================================
// Persistence Objects (PO) 鈥?GORM-tagged structs for MySQL
// ============================================================

type BuiltinFeaturePO struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	FeatureKey  string    `gorm:"column:feature_key;size:128;uniqueIndex:uk_buildin_feat_key;not null" json:"featureKey"`
	Label       string    `gorm:"size:255;not null" json:"label"`
	ValueType   string    `gorm:"column:value_type;size:32;not null" json:"valueType"`
	Stage       int       `gorm:"column:stage" json:"stage,omitempty"`
	Description string    `gorm:"size:512" json:"description"`
	Unit        string    `gorm:"size:64" json:"unit"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (BuiltinFeaturePO) TableName() string { return "buildin_features" }

type RulePO struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"size:255;not null" json:"name"`
	Enabled       bool           `gorm:"not null;default:true;index" json:"enabled"`
	Priority      int            `gorm:"not null;default:0;index" json:"priority"`
	Stage         *int           `gorm:"column:stage" json:"stage,omitempty"`
	Action        string         `gorm:"size:32;not null" json:"action"`
	ConditionJSON string         `gorm:"column:condition_json;type:longtext;not null" json:"conditionJson"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	IsDeleted     bool           `gorm:"column:is_deleted" json:"isDeleted"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	CreatorId     int64          `gorm:"column:creator_id" json:"creatorId"`
}

func (RulePO) TableName() string { return "filter_rules" }

type FilterLogPO struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TraceID             string    `gorm:"column:trace_id;size:64;index" json:"traceId"`
	QueueID             string    `gorm:"column:queue_id;size:128;index" json:"queueId"`
	IP                  string    `gorm:"column:ip;size:64;index" json:"ip"`
	Sender              string    `gorm:"size:512;index" json:"sender"`
	Recipient           string    `gorm:"size:512;index" json:"recipient"`
	Subject             string    `gorm:"size:512;index" json:"subject"`
	Stage               int       `gorm:"column:stage;index" json:"stage,omitempty"`
	RuleID              *int64    `gorm:"column:rule_id" json:"ruleId,omitempty"`
	ActionApplied       string    `gorm:"column:action_applied;size:32" json:"actionApplied"`
	FeatureSnapshotJSON string    `gorm:"column:feature_snapshot_json;type:longtext" json:"featureSnapshotJson"`
	ConditionTraceJSON  string    `gorm:"column:condition_trace_json;type:longtext" json:"conditionTraceJson"`
	DurationMs          int       `gorm:"column:duration_ms" json:"durationMs"`
	CreatedAt           time.Time `gorm:"index" json:"createdAt"`
}

func (FilterLogPO) TableName() string { return "filter_logs" }

type CustomFeaturePO struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	FeatureKey  string         `gorm:"column:feature_key;size:128;uniqueIndex:uk_custom_feat_key;not null" json:"featureKey"`
	Label       string         `gorm:"size:255;not null" json:"label"`
	Stage       int            `gorm:"column:stage" json:"stage,omitempty"`
	Type        string         `gorm:"column:type;size:32;not null;index" json:"type"`
	ValueType   string         `gorm:"column:value_type;size:32;not null" json:"valueType"`
	Enabled     bool           `gorm:"not null;default:true;index" json:"enabled"`
	SpecJSON    string         `gorm:"column:spec_json;type:longtext;not null" json:"specJson"`
	Description string         `gorm:"size:512" json:"description"`
	Unit        string         `gorm:"size:64" json:"unit"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	IsDeleted   bool           `gorm:"column:is_deleted" json:"isDeleted"`
	CreatorId   int64          `gorm:"column:creator_id" json:"creatorId"`
}

func (CustomFeaturePO) TableName() string { return "custom_features" }

type ClassifyModelPO struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"column:name;type:varchar(255)" json:"name"`
	Algorithm     string    `gorm:"column:algorithm;type:varchar(50)" json:"algorithm"`
	Tokenizer     string    `gorm:"column:tokenizer;type:varchar(50)" json:"tokenizer"`
	Languages     string    `gorm:"column:languages;type:longtext" json:"languages"`
	SavePath      string    `gorm:"column:save_path;type:varchar(500)" json:"savePath"`
	Params        string    `gorm:"column:params;type:longtext" json:"params"`
	MaxTextLength int       `gorm:"column:max_text_length;default:256" json:"maxTextLength"`
	EmailFields   string    `gorm:"column:email_fields;type:longtext" json:"emailFields"`
	ClassLabels   string    `gorm:"column:class_labels;type:longtext" json:"classLabels"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	Enabled       bool      `gorm:"column:enabled;default:true" json:"enabled"`
	TrainStatus   string    `gorm:"column:train_status;type:varchar(50);default:pending" json:"trainStatus"`
	TrainResult   string    `gorm:"column:train_result;type:text" json:"trainResult"`
	TrainTime     time.Time `gorm:"column:train_time" json:"trainTime,omitempty"`
	IsDeleted     bool      `gorm:"column:is_deleted;default:false" json:"isDeleted"`
	DeleteTime    time.Time `gorm:"column:delete_time" json:"deleteTime"`
	CreatorId     int64     `gorm:"column:creator_id" json:"creatorId"`
}

func (ClassifyModelPO) TableName() string { return "classify_models" }

type ModelSamplePO struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ClassifyModelID uint      `gorm:"column:classify_model_id;index:idx_model_samples_cm_label,priority:1;not null" json:"modelId"`
	Text            string    `gorm:"column:text;type:text;not null" json:"text"`
	Label           string    `gorm:"column:label;type:varchar(255);index:idx_model_samples_cm_label,priority:2;not null" json:"label"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (ModelSamplePO) TableName() string { return "model_samples" }

// PublicSamplePO is a global training sample not tied to a specific model.
type PublicSamplePO struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CategoryID uint      `gorm:"column:category_id;index:idx_public_samples_category_id;not null" json:"categoryId"`
	Tag        string    `gorm:"column:tag;type:varchar(255);not null" json:"tag"`
	Text       string    `gorm:"column:text;type:text;not null" json:"text"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (PublicSamplePO) TableName() string { return "public_samples" }

// PublicSampleCategoryPO is a managed category for public samples.
type PublicSampleCategoryPO struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(255);uniqueIndex:uk_public_sample_category_name;not null" json:"name"`
	Description string    `gorm:"column:description;type:varchar(500)" json:"description"`
	SampleCount int64     `gorm:"column:sample_count;default:0" json:"sampleCount"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (PublicSampleCategoryPO) TableName() string { return "public_sample_categories" }

type FilterMailStatsDailyPO struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	StatDate      time.Time `gorm:"column:stat_date;type:date;not null;uniqueIndex:uk_filter_mail_stats_day_action,priority:1" json:"statDate"`
	ActionApplied string    `gorm:"column:action_applied;size:32;not null;uniqueIndex:uk_filter_mail_stats_day_action,priority:2" json:"actionApplied"`
	MailCount     int64     `gorm:"column:mail_count;not null" json:"mailCount"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (FilterMailStatsDailyPO) TableName() string { return "filter_mail_stats_daily" }

type FilterStatsRollupWatermarkPO struct {
	ID                    int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	JobName               string     `gorm:"column:job_name;size:64;uniqueIndex;not null" json:"jobName"`
	LastCompletedStatDate *time.Time `gorm:"column:last_completed_stat_date;type:date" json:"lastCompletedStatDate,omitempty"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (FilterStatsRollupWatermarkPO) TableName() string { return "filter_stats_rollup_watermark" }
