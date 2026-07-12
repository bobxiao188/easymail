package launcher

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	filter "easymail/internal/app/filter"
	"easymail/internal/runtime"
	"easymail/pkg/config"
	"easymail/pkg/heartbeat"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"
)

type classifierRunner struct {
	name      string
	ln        net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
	log       *easylog.Logger
	cfg       config.ClassifierConfig
	heartbeat *HeartbeatManager
}

func (r *classifierRunner) Name() string { return r.name }

func (r *classifierRunner) Start() error {
	go func() {
		if err := filter.Serve(r.ctx, r.cfg, r.ln, r.log); err != nil && r.ctx.Err() == nil {
			r.log.Errorf("classifier serve error: %v", err)
		}
	}()
	return nil
}

func (r *classifierRunner) Stop() error {
	r.cancel()
	return nil
}

// RegisterHeartbeat registers this runner with the global heartbeat manager
func (r *classifierRunner) RegisterHeartbeat(mgr *heartbeat.ServiceManager) {
	if mgr == nil || !r.heartbeat.IsEnabled() {
		return
	}
	mgr.Register(r.heartbeat.ServiceName(), r.heartbeat.Interval(), r.log)
}

// NewClassifierRunner returns (nil, nil) when classifier is disabled.
func NewClassifierRunner(rt *runtime.Runtime, cfg config.ClassifierConfig, modLog *easylog.Logger) (Runner, error) {
	if !cfg.Enable {
		return nil, nil
	}

	fam := strings.ToLower(strings.TrimSpace(cfg.Family))
	if fam == "" {
		fam = "tcp"
	}

	ln, err := listen(fam, cfg.Listen)
	if err != nil {
		return nil, fmt.Errorf("classifier listen %s %s: %w", fam, cfg.Listen, err)
	}

	modLog.Infof("%s", appi18n.LogMessage(appi18n.KeyLogListen, map[string]interface{}{"Fam": fam, "Addr": cfg.Listen}))

	// Create heartbeat manager (disabled if interval is 0 or redis is nil)
	var heartbeatMgr *HeartbeatManager
	if rt.Config.HeartbeatIntervalSec > 0 {
		heartbeatMgr = NewHeartbeatManager(
			true,
			time.Duration(rt.Config.HeartbeatIntervalSec)*time.Second,
			heartbeat.DefaultTTL,
			"classifier",
		)
	}

	return &classifierRunner{
		name:      "classifier",
		ln:        ln,
		ctx:       context.Background(),
		cancel:    nil,
		log:       modLog,
		cfg:       cfg,
		heartbeat: heartbeatMgr,
	}, nil
}
