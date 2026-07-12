package heartbeat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"easymail/pkg/logger/easylog"

	"github.com/redis/go-redis/v9"
)

// ServiceManager is a global heartbeat service manager
type ServiceManager struct {
	store    HeartbeatStore
	interval time.Duration
	ttl      time.Duration
	logger   *easylog.Logger
	mu       sync.RWMutex
	services map[string]*registeredService
	ctx      context.Context
	cancel   context.CancelFunc
}

// registeredService holds per-service heartbeat state
type registeredService struct {
	name          string
	interval      time.Duration
	ttl           time.Duration
	lastHeartbeat time.Time
	heartbeatFn   func() error
	logger        *easylog.Logger
}

// NewServiceManager creates a new global heartbeat service manager.
// When rdb is nil, a MemoryHeartbeatStore is used as fallback.
func NewServiceManager(rdb *redis.Client, interval, ttl time.Duration, logger *easylog.Logger) *ServiceManager {
	var store HeartbeatStore
	if rdb != nil {
		store = NewRedisHeartbeatStore(rdb)
	} else {
		store = NewMemoryHeartbeatStore()
	}
	return newServiceManagerWithStore(store, interval, ttl, logger)
}

// NewServiceManagerWithStore creates a ServiceManager with an explicit HeartbeatStore.
func NewServiceManagerWithStore(store HeartbeatStore, interval, ttl time.Duration, logger *easylog.Logger) *ServiceManager {
	return newServiceManagerWithStore(store, interval, ttl, logger)
}

func newServiceManagerWithStore(store HeartbeatStore, interval, ttl time.Duration, logger *easylog.Logger) *ServiceManager {
	if interval == 0 {
		interval = DefaultInterval
	}
	if ttl == 0 {
		ttl = DefaultTTL
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ServiceManager{
		store:    store,
		interval: interval,
		ttl:      ttl,
		logger:   logger,
		services: make(map[string]*registeredService),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the heartbeat loop for all registered services
func (m *ServiceManager) Start() {
	if m.store == nil {
		if m.logger != nil {
			m.logger.Warn("heartbeat: store is nil, heartbeat disabled")
		}
		return
	}

	if m.logger != nil {
		m.logger.Infof("heartbeat manager started (interval=%v, ttl=%v)", m.interval, m.ttl)
	}

	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.sendAllHeartbeats()
			case <-m.ctx.Done():
				if m.logger != nil {
					m.logger.Info("heartbeat manager stopped")
				}
				return
			}
		}
	}()
}

// Stop stops the heartbeat manager
func (m *ServiceManager) Stop() {
	m.cancel()
}

// Register adds a service to heartbeat monitoring
func (m *ServiceManager) Register(name string, interval time.Duration, logger *easylog.Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.services[name]; exists {
		if m.logger != nil {
			m.logger.Warnf("heartbeat: service %q already registered", name)
		}
		return
	}

	svc := &registeredService{
		name:        name,
		interval:    interval,
		ttl:         m.ttl,
		heartbeatFn: m.makeHeartbeatFunc(name),
		logger:      logger,
	}
	m.services[name] = svc

	if m.logger != nil {
		m.logger.Infof("heartbeat: service %q registered (interval=%v)", name, interval)
	}
}

// Unregister removes a service from heartbeat monitoring
func (m *ServiceManager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.services[name]; !exists {
		return
	}

	delete(m.services, name)
	if m.logger != nil {
		m.logger.Infof("heartbeat: service %q unregistered", name)
	}
}

// IsRegistered returns true if a service is registered
func (m *ServiceManager) IsRegistered(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.services[name]
	return exists
}

// sendAllHeartbeats sends heartbeats for all registered services
func (m *ServiceManager) sendAllHeartbeats() {
	m.mu.RLock()
	services := make([]*registeredService, 0, len(m.services))
	for _, svc := range m.services {
		services = append(services, svc)
	}
	m.mu.RUnlock()

	for _, svc := range services {
		if err := svc.heartbeatFn(); err != nil {
			if svc.logger != nil {
				svc.logger.Errorf("heartbeat: failed for %q: %v", svc.name, err)
			}
		}
	}
}

// makeHeartbeatFunc creates a heartbeat function for a service
func (m *ServiceManager) makeHeartbeatFunc(name string) func() error {
	return func() error {
		if m.store == nil {
			return fmt.Errorf("heartbeat store is nil")
		}

		key := ServiceStatusPrefix + name + ServiceHeartbeatSuffix
		value := time.Now().Format(time.RFC3339)

		if err := m.store.Set(m.ctx, key, value, m.ttl); err != nil {
			return fmt.Errorf("set heartbeat: %w", err)
		}

		return nil
	}
}

// GetStatus returns the current status of a service
func (m *ServiceManager) GetStatus(name string) (string, time.Time, error) {
	if m.store == nil {
		return StatusStopped, time.Time{}, fmt.Errorf("heartbeat store is nil")
	}

	key := ServiceStatusPrefix + name + ServiceHeartbeatSuffix
	val, err := m.store.Get(context.Background(), key)
	if err != nil {
		return StatusStopped, time.Time{}, nil
	}

	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return StatusStopped, time.Time{}, nil
	}

	return StatusRunning, t, nil
}

// ListAllServices returns all registered service names
func (m *ServiceManager) ListAllServices() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.services))
	for name := range m.services {
		names = append(names, name)
	}
	return names
}

// Global manager instance (lazy initialization)
var globalManager *ServiceManager
var managerOnce sync.Once

// GetGlobalManager returns the global heartbeat manager
func GetGlobalManager() *ServiceManager {
	managerOnce.Do(func() {
		globalManager = NewServiceManager(nil, 0, 0, nil)
	})
	return globalManager
}

// SetGlobalManager sets the global heartbeat manager (typically called in main)
func SetGlobalManager(manager *ServiceManager) {
	globalManager = manager
}

// InitGlobal initializes the global manager with config
func InitGlobal(rdb *redis.Client, interval, ttl time.Duration, logger *easylog.Logger) {
	SetGlobalManager(NewServiceManager(rdb, interval, ttl, logger))
}