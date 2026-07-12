package redis

import (
	"context"
	"strings"
	"sync"
	"time"

	"easymail/internal/domain/filter"
	"easymail/pkg/database"
)

// memoryCounterStore is a per-package in-memory fallback for Incr/GetInt.
// Only used when Redis is unavailable. Not shared across processes.
type memoryCounterEntry struct {
	count     int64
	expiresAt time.Time
}

type memoryCounterStore struct {
	mu   sync.RWMutex
	data map[string]memoryCounterEntry
	done chan struct{}
}

var globalMemStore = newMemoryCounterStore()

func newMemoryCounterStore() *memoryCounterStore {
	s := &memoryCounterStore{
		data: make(map[string]memoryCounterEntry),
		done: make(chan struct{}),
	}
	go s.evictLoop()
	return s
}

func (s *memoryCounterStore) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.evictExpired()
		case <-s.done:
			return
		}
	}
}

func (s *memoryCounterStore) evictExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.data {
		if now.After(v.expiresAt) {
			delete(s.data, k)
		}
	}
}

func (s *memoryCounterStore) incr(k string, ttl time.Duration) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[k]
	if !ok || time.Now().After(e.expiresAt) {
		e = memoryCounterEntry{expiresAt: time.Now().Add(ttl)}
	}
	e.count++
	s.data[k] = e
	return e.count
}

func (s *memoryCounterStore) get(k string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[k]
	if !ok || time.Now().After(e.expiresAt) {
		return 0
	}
	return e.count
}

// Window identifies a simple TTL-based counting window.
type Window struct {
	Name string
	TTL  time.Duration
}

var (
	Window1m = Window{Name: "1m", TTL: 1 * time.Minute}
	Window5m = Window{Name: "5m", TTL: 5 * time.Minute}
)

const FilterDayCounterTTL = 24 * time.Hour

func LocalDateYYYYMMDD(t time.Time) string {
	return t.In(time.Local).Format("20060102")
}

func SecondsSinceLocalMidnight(now time.Time) int64 {
	l := now.In(time.Local)
	start := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, l.Location())
	sec := int64(l.Sub(start) / time.Second)
	if sec < 1 {
		return 1
	}
	return sec
}

func key(parts ...string) string {
	return "filter:" + strings.Join(parts, ":")
}

func Incr(ctx context.Context, k string, ttl time.Duration) (int64, error) {
	rdb := database.GetRedisClient()
	if rdb == nil {
		return globalMemStore.incr(k, ttl), nil
	}
	pipe := rdb.Pipeline()
	incr := pipe.Incr(ctx, k)
	if ttl > 0 {
		pipe.Expire(ctx, k, ttl)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func GetInt(ctx context.Context, k string) (int64, error) {
	rdb := database.GetRedisClient()
	if rdb == nil {
		return globalMemStore.get(k), nil
	}
	n, err := rdb.Get(ctx, k).Int64()
	if err != nil {
		return 0, nil
	}
	return n, nil
}

// --- Key builders ---

func IPConnectKey(w Window, ip string) string {
	return key("ip", w.Name, strings.ToLower(strings.TrimSpace(ip)), "connect")
}

func IPConnectDayKey(yyyymmdd, ip string) string {
	return key("ip", "1d", yyyymmdd, strings.ToLower(strings.TrimSpace(ip)), "connect")
}

func SenderMailFromKey(w Window, sender string) string {
	return key("sender", w.Name, strings.ToLower(strings.TrimSpace(sender)), "mailfrom")
}

func SenderMailFromDayKey(yyyymmdd, sender string) string {
	return key("sender", "1d", yyyymmdd, strings.ToLower(strings.TrimSpace(sender)), "mailfrom")
}

func SenderRcptToKey(w Window, sender string) string {
	return key("sender", w.Name, strings.ToLower(strings.TrimSpace(sender)), "rcptto")
}

func SenderRcptDomainKey(w Window, sender, rcptDomain string) string {
	sender = strings.ToLower(strings.TrimSpace(sender))
	rcptDomain = strings.ToLower(strings.TrimSpace(rcptDomain))
	return key("sender", w.Name, sender, "rcpt_domain", rcptDomain)
}

func SenderRcptDomainDayKey(yyyymmdd, sender, rcptDomain string) string {
	sender = strings.ToLower(strings.TrimSpace(sender))
	rcptDomain = strings.ToLower(strings.TrimSpace(rcptDomain))
	return key("sender", "1d", yyyymmdd, sender, "rcpt_domain", rcptDomain)
}

func IPOutcomeKey(w Window, ip string, outcome filter.Outcome) string {
	return key("ip", w.Name, strings.ToLower(strings.TrimSpace(ip)), "outcome", string(outcome))
}

func SenderOutcomeKey(w Window, sender string, outcome filter.Outcome) string {
	return key("sender", w.Name, strings.ToLower(strings.TrimSpace(sender)), "outcome", string(outcome))
}

func SenderRcptDomainOutcomeKey(w Window, sender, rcptDomain string, outcome filter.Outcome) string {
	return key("sender", w.Name, strings.ToLower(strings.TrimSpace(sender)), "rcpt_domain", strings.ToLower(strings.TrimSpace(rcptDomain)), "outcome", string(outcome))
}

func IPOutcomeDayKey(yyyymmdd, ip string, outcome filter.Outcome) string {
	return key("ip", "1d", yyyymmdd, strings.ToLower(strings.TrimSpace(ip)), "outcome", string(outcome))
}

func SenderOutcomeDayKey(yyyymmdd, sender string, outcome filter.Outcome) string {
	return key("sender", "1d", yyyymmdd, strings.ToLower(strings.TrimSpace(sender)), "outcome", string(outcome))
}

func SenderRcptDomainOutcomeDayKey(yyyymmdd, sender, rcptDomain string, outcome filter.Outcome) string {
	return key("sender", "1d", yyyymmdd, strings.ToLower(strings.TrimSpace(sender)), "rcpt_domain", strings.ToLower(strings.TrimSpace(rcptDomain)), "outcome", string(outcome))
}

func RecordIPOutcome(ctx context.Context, ip string, outcome filter.Outcome) {
	if strings.TrimSpace(ip) == "" {
		return
	}
	day := LocalDateYYYYMMDD(time.Now())
	_, _ = Incr(ctx, IPOutcomeDayKey(day, ip, outcome), FilterDayCounterTTL)
	_, _ = Incr(ctx, IPOutcomeKey(Window5m, ip, outcome), Window5m.TTL)
}

func RecordSenderOutcome(ctx context.Context, sender string, outcome filter.Outcome) {
	sender = strings.ToLower(strings.TrimSpace(sender))
	if sender == "" {
		return
	}
	day := LocalDateYYYYMMDD(time.Now())
	_, _ = Incr(ctx, SenderOutcomeDayKey(day, sender, outcome), FilterDayCounterTTL)
	_, _ = Incr(ctx, SenderOutcomeKey(Window5m, sender, outcome), Window5m.TTL)
}

func RecordSenderRcptDomainOutcome(ctx context.Context, sender, rcptDomain string, outcome filter.Outcome) {
	sender = strings.ToLower(strings.TrimSpace(sender))
	rcptDomain = strings.ToLower(strings.TrimSpace(rcptDomain))
	if sender == "" || rcptDomain == "" {
		return
	}
	day := LocalDateYYYYMMDD(time.Now())
	_, _ = Incr(ctx, SenderRcptDomainOutcomeDayKey(day, sender, rcptDomain, outcome), FilterDayCounterTTL)
	_, _ = Incr(ctx, SenderRcptDomainOutcomeKey(Window5m, sender, rcptDomain, outcome), Window5m.TTL)
}
