package launcher

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	scanmilter "easymail/internal/adapter/milter"
	"easymail/internal/infrastructure/filter/classifier/modelcache"
	"easymail/internal/protocol/milter"
	"easymail/internal/runtime"
	"easymail/pkg/config"
	"easymail/pkg/heartbeat"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type milterRunner struct {
	name      string
	ln        net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
	log       *easylog.Logger
	filterCfg config.FilterConfig
	db        *gorm.DB
	redisCli  *redis.Client
	heartbeat *HeartbeatManager
}

func (r *milterRunner) Name() string { return r.name }

func (r *milterRunner) Start() error {
	// set up cache before any service goroutine starts
	if r.db != nil {
		mc := modelcache.New()
		modelcache.SetMilterCache(mc)
	}
	act, proto := milter.DefaultFilterNegotiation()
	milterHandler := scanmilter.NewMilterHandlerFactory(r.db, r.filterCfg, r.log, r.redisCli)
	go func() {
		if err := milter.ProtocolServe(r.ctx, r.ln, milterHandler, act, proto, r.log); err != nil && r.ctx.Err() == nil {
			r.log.Errorf("%s", appi18n.LogMessage(appi18n.KeyLogServerError, map[string]interface{}{"Err": err.Error()}))
		}
	}()
	return nil
}

func (r *milterRunner) Stop() error {
	r.cancel()
	if mc := modelcache.MilterCache(); mc != nil {
		mc.Invalidate()
	}
	return r.ln.Close()
}

// RegisterHeartbeat registers this runner with the global heartbeat manager.
// Also registers a "filter" heartbeat since the filter rule engine runs in-process.
func (r *milterRunner) RegisterHeartbeat(mgr *heartbeat.ServiceManager) {
	if mgr == nil || !r.heartbeat.IsEnabled() {
		return
	}
	mgr.Register(r.heartbeat.ServiceName(), r.heartbeat.Interval(), r.log)
	mgr.Register("filter", r.heartbeat.Interval(), r.log)
}

// NewMilterRunner returns (nil, nil) when milter is disabled and listenOverride is empty.
func NewMilterRunner(rt *runtime.Runtime, cfg config.MilterConfig, listenOverride string, modLog *easylog.Logger) (Runner, error) {
	if !cfg.Enable && strings.TrimSpace(listenOverride) == "" {
		return nil, nil
	}

	fam := strings.ToLower(strings.TrimSpace(cfg.Family))
	if fam == "" {
		fam = "tcp"
	}

	addr := strings.TrimSpace(cfg.Listen)
	if listenOverride != "" {
		addr = strings.TrimSpace(listenOverride)
	}
	if addr == "" {
		if fam == "unix" {
			return nil, fmt.Errorf("milter: unix listen path is empty")
		}
		addr = "127.0.0.1:8891"
	}

	ln, err := listen(fam, addr)
	if err != nil {
		return nil, fmt.Errorf("milter listen %s %s: %w", fam, addr, err)
	}

	modLog.Infof("%s", appi18n.LogMessage(appi18n.KeyLogListen, map[string]interface{}{"Fam": fam, "Addr": addr}))

	label := "milter"

	// Create heartbeat manager (disabled if interval is 0 or redis is nil)
	var heartbeatMgr *HeartbeatManager
	if rt.Config.HeartbeatIntervalSec > 0 {
		heartbeatMgr = NewHeartbeatManager(
			true,
			time.Duration(rt.Config.HeartbeatIntervalSec)*time.Second,
			heartbeat.DefaultTTL,
			"milter",
		)
	}

	return &milterRunner{
		name:      label,
		ln:        ln,
		ctx:       context.Background(),
		cancel:    nil,
		log:       modLog,
		filterCfg: cfg.Filter.ToFilterConfig(),
		db:        rt.DB,
		redisCli:  rt.RedisClient,
		heartbeat: heartbeatMgr,
	}, nil
}
