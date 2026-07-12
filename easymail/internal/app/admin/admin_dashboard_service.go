package admin

import (
	"context"
)

type ServiceStatus struct {
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Status        string `json:"status"`
	LastHeartbeat string `json:"lastHeartbeat"`
	ConfigEnabled bool   `json:"configEnabled"`
	Description   string `json:"description"`
}

type MailStats struct {
	Date            string `json:"date"`
	TotalCount      int64  `json:"totalCount"`
	NormalCount     int64  `json:"normalCount"`
	SpamCount       int64  `json:"spamCount"`
	RejectCount     int64  `json:"rejectCount"`
	QuarantineCount int64  `json:"quarantineCount"`
	TotalSize       int64  `json:"totalSize"`
	NormalSize      int64  `json:"normalSize"`
	SpamSize        int64  `json:"spamSize"`
	RejectSize      int64  `json:"rejectSize"`
	QuarantineSize  int64  `json:"quarantineSize"`
}

type TopSender struct {
	Rank   int    `json:"rank"`
	Sender string `json:"sender"`
	Count  int64  `json:"count"`
	Type   string `json:"type"`
}

type DashboardService interface {
	GetServiceStatus(ctx context.Context) ([]ServiceStatus, error)
	GetMailStatsDaily(ctx context.Context, days int) ([]MailStats, error)
	GetMailStatsMonthly(ctx context.Context, months int) ([]MailStats, error)
	GetTopSenders(ctx context.Context, senderType string, hours int, limit int) ([]TopSender, error)
	GetFilterPolicyStatsDaily(ctx context.Context, days int) ([]MailStats, error)
}
