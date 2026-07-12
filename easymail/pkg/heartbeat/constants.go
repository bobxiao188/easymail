package heartbeat

import "time"

const (
	ServiceStatusPrefix    = "easymail:service:"
	ServiceHeartbeatSuffix = ":heartbeat"
	DefaultInterval        = 10 * time.Second
	DefaultTTL             = 30 * time.Second
)

// Status constants
const (
	StatusRunning  = "running"
	StatusWarning  = "warning"
	StatusStopped  = "stopped"
)

