package mysql

import "time"

// TrainingTaskPO records an ad-hoc model training job launched from the admin UI.
// It drives an async FastText supervised training run that aggregates public
// samples (by source tag) into per-class training lines and produces a classify model.
type TrainingTaskPO struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ModelName     string    `gorm:"column:model_name;type:varchar(255);not null" json:"modelName"`
	Algorithm     string    `gorm:"column:algorithm;type:varchar(50);not null" json:"algorithm"`
	Params        string    `gorm:"column:params;type:longtext" json:"params"`
	SampleMappings string   `gorm:"column:sample_mappings;type:longtext" json:"sampleMappings"`
	Status        string    `gorm:"column:status;type:varchar(50);default:pending;not null" json:"status"`
	TrainResult   string    `gorm:"column:train_result;type:longtext" json:"trainResult"`
	ModelID       uint      `gorm:"column:model_id;default:0" json:"modelId"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (TrainingTaskPO) TableName() string { return "training_tasks" }
