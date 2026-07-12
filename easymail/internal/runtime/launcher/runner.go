package launcher

import (
	"fmt"
	"log"

	appi18n "easymail/pkg/i18n"
	"easymail/pkg/heartbeat"
)

// Runner is a background service started and stopped by the composite launcher.
type Runner interface {
	Name() string
	Start() error
	Stop() error
}

// HeartbeatRunner is a runner that supports heartbeat registration
type HeartbeatRunner interface {
	Runner
	RegisterHeartbeat(mgr *heartbeat.ServiceManager)
}

// Manager owns an ordered list of runners; StartAll starts in order, StopAll stops in reverse order.
type Manager struct {
	Runners []Runner
	// HeartbeatManager is the global heartbeat service manager
	HeartbeatMgr *heartbeat.ServiceManager
}

// StartAll starts each runner in order. On failure, already-started runners are stopped in reverse order.
func (m *Manager) StartAll() error {
	for i, r := range m.Runners {
		if err := r.Start(); err != nil {
			for j := i - 1; j >= 0; j-- {
				_ = m.Runners[j].Stop()
			}
			return fmt.Errorf("%s: %w", r.Name(), err)
		}
		log.Printf("%s", appi18n.LogMessage(appi18n.KeyLogLauncherStarted, map[string]interface{}{"Name": r.Name()}))
	}
	return nil
}

// StopAll stops every runner in reverse registration order.
func (m *Manager) StopAll() {
	for i := len(m.Runners) - 1; i >= 0; i-- {
		r := m.Runners[i]
		if err := r.Stop(); err != nil {
			log.Printf("%s", appi18n.LogMessage(appi18n.KeyLogLauncherStopErr, map[string]interface{}{"Name": r.Name(), "Err": err.Error()}))
		} else {
			log.Printf("%s", appi18n.LogMessage(appi18n.KeyLogLauncherStopped, map[string]interface{}{"Name": r.Name()}))
		}
	}
}

// RegisterHeartbeatRunners registers all heartbeat-capable runners with the global manager
func (m *Manager) RegisterHeartbeatRunners() {
	if m.HeartbeatMgr == nil {
		return
	}

	for _, r := range m.Runners {
		if hr, ok := r.(HeartbeatRunner); ok {
			hr.RegisterHeartbeat(m.HeartbeatMgr)
		}
	}
}

