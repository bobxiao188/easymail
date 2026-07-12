package mysql

import (
	"time"

	"easymail/internal/domain/shared"
)

// PostfixAgentPO is the persistence object for PostfixAgent.
type PostfixAgentPO struct {
	ID          shared.GlobalID `gorm:"primaryKey;type:varchar(36);not null"`
	Name        string          `gorm:"uniqueIndex;size:64;not null"`
	Host        string          `gorm:"size:256;not null"`
	Token       string          `gorm:"size:512;not null"`
	Enabled     bool            `gorm:"type:tinyint(1);default(1)"`
	LastStatus  string          `gorm:"size:32;default:unknown"`
	LastSyncAt  time.Time       `gorm:"type:timestamp"`
	Description string          `gorm:"size:512"`
	CreateTime  time.Time       `gorm:"autoCreateTime"`
	UpdateTime  time.Time       `gorm:"autoUpdateTime"`
}

func (PostfixAgentPO) TableName() string { return "postfix_agents" }

// PostfixConfigPO is the persistence object for PostfixConfig.
type PostfixConfigPO struct {
	ID          shared.GlobalID `gorm:"primaryKey;type:varchar(128);not null"`
	ParamName   string          `gorm:"uniqueIndex;size:128;not null"`
	ParamValue  string          `gorm:"type:text"`
	Category    string          `gorm:"size:32;default:main;not null"`
	IsManaged   bool            `gorm:"type:tinyint(1);default(0)"`
	Enabled     bool            `gorm:"type:tinyint(1);default(1)"`
	Description string          `gorm:"size:512"`
	SortOrder   int             `gorm:"default:0"`
	CreateTime  time.Time       `gorm:"autoCreateTime"`
	UpdateTime  time.Time       `gorm:"autoUpdateTime"`
}

func (PostfixConfigPO) TableName() string { return "postfix_configs" }

// PostfixDeliveryLogPO is the persistence object for PostfixDeliveryLog.
type PostfixDeliveryLogPO struct {
	ID             shared.GlobalID `gorm:"primaryKey;type:varchar(36);not null"`
	AgentID        shared.GlobalID `gorm:"type:varchar(36);not null;index"`
	Action         string          `gorm:"size:32;not null"`
	Status         string          `gorm:"size:16;not null"`
	ConfigSnapshot string          `gorm:"type:longtext"`
	ErrorMessage   string          `gorm:"type:text"`
	CreatedAt      time.Time       `gorm:"autoCreateTime"`
}

func (PostfixDeliveryLogPO) TableName() string { return "postfix_delivery_logs" }
