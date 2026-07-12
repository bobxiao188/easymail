package admin

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	redisstore "easymail/internal/infrastructure/filter/persistence/redis"
	"easymail/internal/infrastructure/filter/utcdate"
	"easymail/internal/infrastructure/persistence/mysql"
	"easymail/pkg/database"
	"easymail/pkg/heartbeat"
	"easymail/pkg/i18n"
	"easymail/pkg/mailstats"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type dashboardServiceImpl struct {
	rdb *redis.Client
	db  *gorm.DB
}

func NewDashboardService(rdb *redis.Client, db *gorm.DB) DashboardService {
	return &dashboardServiceImpl{rdb: rdb, db: db}
}

func (s *dashboardServiceImpl) GetServiceStatus(ctx context.Context) ([]ServiceStatus, error) {
	return getServiceStatus(ctx, s.rdb)
}

func (s *dashboardServiceImpl) GetMailStatsDaily(ctx context.Context, days int) ([]MailStats, error) {
	return getMailStatsDaily(ctx, s.rdb, days)
}

func (s *dashboardServiceImpl) GetMailStatsMonthly(ctx context.Context, months int) ([]MailStats, error) {
	return getMailStatsMonthly(ctx, s.rdb, months)
}

func (s *dashboardServiceImpl) GetTopSenders(ctx context.Context, senderType string, hours int, limit int) ([]TopSender, error) {
	return getTopSenders(ctx, s.rdb, senderType, hours, limit)
}

func (s *dashboardServiceImpl) GetFilterPolicyStatsDaily(ctx context.Context, days int) ([]MailStats, error) {
	return getFilterPolicyStatsDaily(ctx, s.rdb, s.db, days)
}

func parseInt64Field(m map[string]string, key string) int64 {
	s := strings.TrimSpace(m[key])
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func mailStatsFromRedisHash(date string, h map[string]string) MailStats {
	if len(h) == 0 {
		return MailStats{Date: date}
	}
	return MailStats{
		Date:            date,
		TotalCount:      parseInt64Field(h, "total_count"),
		NormalCount:     parseInt64Field(h, "normal_count"),
		SpamCount:       parseInt64Field(h, "spam_count"),
		RejectCount:     parseInt64Field(h, "reject_count"),
		QuarantineCount: parseInt64Field(h, "quarantine_count"),
		TotalSize:       parseInt64Field(h, "total_size"),
		NormalSize:      parseInt64Field(h, "normal_size"),
		SpamSize:        parseInt64Field(h, "spam_size"),
		RejectSize:      parseInt64Field(h, "reject_size"),
		QuarantineSize:  parseInt64Field(h, "quarantine_size"),
	}
}

func filterPolicyRedisHashToMailStats(date string, h map[string]string) MailStats {
	if len(h) == 0 {
		return MailStats{Date: date}
	}
	return MailStats{
		Date:            date,
		TotalCount:      parseInt64Field(h, "total_count"),
		NormalCount:     parseInt64Field(h, "count_accept"),
		SpamCount:       parseInt64Field(h, "count_spam"),
		RejectCount:     parseInt64Field(h, "count_reject"),
		QuarantineCount: parseInt64Field(h, "count_quarantine"),
	}
}

func getServiceList() []ServiceStatus {
	cfg := database.GetAppConfig()
	if cfg == nil {
		return []ServiceStatus{}
	}
	services := []ServiceStatus{
		{Name: "dovecot", DisplayName: "Dovecot", ConfigEnabled: cfg.Dovecot.Enable, Description: i18n.KeyDashboardSvcDovecot},
		{Name: "milter", DisplayName: "Milter", ConfigEnabled: cfg.Milter.Enable, Description: i18n.KeyDashboardSvcMilter},
		{Name: "filter", DisplayName: "filter", ConfigEnabled: cfg.Milter.Filter.Enable, Description: i18n.KeyDashboardSvcFilter},
		{Name: "lmtp", DisplayName: "LMTP", ConfigEnabled: cfg.LMTP.Enable, Description: i18n.KeyDashboardSvcLMTP},
		{Name: "admin", DisplayName: "Admin", ConfigEnabled: cfg.Admin.Enable, Description: i18n.KeyDashboardSvcAdmin},
		{Name: "webmail", DisplayName: "Webmail", ConfigEnabled: cfg.Webmail.Enable, Description: i18n.KeyDashboardSvcWebmail},
		{Name: "imap", DisplayName: "IMAP", ConfigEnabled: cfg.IMAP.Enable, Description: i18n.KeyDashboardSvcIMAP},
	}
	return services
}

func getServiceStatus(ctx context.Context, rdb *redis.Client) ([]ServiceStatus, error) {
	if rdb == nil {
		return getServiceList(), nil
	}
	result := make([]ServiceStatus, 0)
	for _, svc := range getServiceList() {
		status := svc
		key := heartbeat.ServiceStatusPrefix + svc.Name + heartbeat.ServiceHeartbeatSuffix
		val, err := rdb.Get(ctx, key).Result()
		if err == nil {
			status.LastHeartbeat = val
			ts, err := time.Parse(time.RFC3339, val)
			if err == nil && time.Since(ts) < 30*time.Second {
				status.Status = heartbeat.StatusRunning
			} else if err == nil && time.Since(ts) < 120*time.Second {
				status.Status = heartbeat.StatusWarning
			} else {
				status.Status = heartbeat.StatusStopped
			}
		} else {
			status.Status = heartbeat.StatusStopped
		}
		result = append(result, status)
	}
	return result, nil
}

func getMailStatsDaily(ctx context.Context, rdb *redis.Client, days int) ([]MailStats, error) {
	if rdb == nil {
		result := make([]MailStats, 0, days)
		now := time.Now()
		for i := days - 1; i >= 0; i-- {
			result = append(result, MailStats{Date: now.AddDate(0, 0, -i).Format("2006-01-02")})
		}
		return result, nil
	}
	result := make([]MailStats, 0, days)
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		dateStr := now.AddDate(0, 0, -i).Format("2006-01-02")
		key := mailstats.MailStatsDailyPrefix + dateStr
		h, err := rdb.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result = append(result, mailStatsFromRedisHash(dateStr, h))
	}
	return result, nil
}

func getMailStatsMonthly(ctx context.Context, rdb *redis.Client, months int) ([]MailStats, error) {
	if rdb == nil {
		result := make([]MailStats, 0, months)
		now := time.Now()
		for i := months - 1; i >= 0; i-- {
			result = append(result, MailStats{Date: now.AddDate(0, -i, 0).Format("2006-01")})
		}
		return result, nil
	}
	result := make([]MailStats, 0, months)
	now := time.Now()
	for i := months - 1; i >= 0; i-- {
		dateStr := now.AddDate(0, -i, 0).Format("2006-01")
		key := mailstats.MailStatsMonthlyPrefix + dateStr
		h, err := rdb.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result = append(result, mailStatsFromRedisHash(dateStr, h))
	}
	return result, nil
}

func getTopSenders(ctx context.Context, rdb *redis.Client, senderType string, hours int, limit int) ([]TopSender, error) {
	if rdb == nil {
		return make([]TopSender, 0), nil
	}
	result := make([]TopSender, 0, limit)
	now := time.Now()
	startTime := now.Add(-time.Duration(hours) * time.Hour)
	pattern := fmt.Sprintf("%s%s:*", mailstats.TopSenderPrefix, senderType)
	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	type senderCount struct {
		sender string
		count  int64
	}
	var counts []senderCount
	for _, key := range keys {
		lastUpdate, err := rdb.HGet(ctx, key, "updated_at").Result()
		if err != nil {
			continue
		}
		updateTime, err := time.Parse(time.RFC3339, lastUpdate)
		if err != nil || updateTime.Before(startTime) {
			continue
		}
		countStr, err := rdb.HGet(ctx, key, "count").Result()
		if err != nil {
			continue
		}
		var count int64
		fmt.Sscanf(countStr, "%d", &count)
		sender := key[len(mailstats.TopSenderPrefix)+len(senderType)+1:]
		counts = append(counts, senderCount{sender: sender, count: count})
	}
	sort.Slice(counts, func(i, j int) bool { return counts[i].count > counts[j].count })
	if len(counts) > limit {
		counts = counts[:limit]
	}
	for i, sc := range counts {
		result = append(result, TopSender{Rank: i + 1, Sender: sc.sender, Count: sc.count, Type: senderType})
	}
	return result, nil
}

func getFilterPolicyStatsDaily(ctx context.Context, rdb *redis.Client, db *gorm.DB, days int) ([]MailStats, error) {
	if days < 1 {
		days = 7
	}
	now := time.Now().UTC()
	today := utcdate.DateUTC(now)
	startOldest := utcdate.DateUTC(now.AddDate(0, 0, -(days - 1)))
	dbByDate := make(map[string]MailStats, days)
	if db != nil {
		var pos []mysql.FilterMailStatsDailyPO
		if err := db.WithContext(ctx).Model(&mysql.FilterMailStatsDailyPO{}).Where("stat_date >= ? AND stat_date < ?", startOldest, today).Find(&pos).Error; err != nil {
			return nil, err
		}
		for _, po := range pos {
			ds := utcdate.DateUTC(po.StatDate).Format("2006-01-02")
			m := dbByDate[ds]
			m.Date = ds
			switch strings.ToLower(strings.TrimSpace(po.ActionApplied)) {
			case "accept":
				m.NormalCount += po.MailCount
			case "spam":
				m.SpamCount += po.MailCount
			case "reject":
				m.RejectCount += po.MailCount
			case "quarantine":
				m.QuarantineCount += po.MailCount
			default:
				m.NormalCount += po.MailCount
			}
			dbByDate[ds] = m
		}
	}
	out := make([]MailStats, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := utcdate.DateUTC(now.AddDate(0, 0, -i))
		ds := d.Format("2006-01-02")
		if d.Before(today) {
			m := dbByDate[ds]
			if m.Date == "" {
				m.Date = ds
			}
			out = append(out, m)
			continue
		}
		var h map[string]string
		if rdb != nil {
			key := redisstore.FilterDayStatsPrefix + ds
			h, _ = rdb.HGetAll(ctx, key).Result()
		}
		out = append(out, filterPolicyRedisHashToMailStats(ds, h))
	}
	return out, nil
}

