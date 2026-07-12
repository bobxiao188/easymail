package launcher

import (
	"time"
)

// HeartbeatManager manages heartbeat registration for all runners
type HeartbeatManager struct {
	enabled     bool
	interval    time.Duration
	ttl         time.Duration
	serviceName string
}

// NewHeartbeatManager creates a heartbeat manager for a runner
func NewHeartbeatManager(enabled bool, interval, ttl time.Duration, serviceName string) *HeartbeatManager {
	return &HeartbeatManager{
		enabled:     enabled,
		interval:    interval,
		ttl:         ttl,
		serviceName: serviceName,
	}
}

// IsEnabled returns true if heartbeat is enabled for this runner
func (h *HeartbeatManager) IsEnabled() bool {
	return h.enabled
}

// ServiceName returns the service name for heartbeat
func (h *HeartbeatManager) ServiceName() string {
	return h.serviceName
}

// Interval returns the heartbeat interval
func (h *HeartbeatManager) Interval() time.Duration {
	return h.interval
}

// TTL returns the heartbeat TTL
func (h *HeartbeatManager) TTL() time.Duration {
	return h.ttl
}

