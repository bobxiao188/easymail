package launcher

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"easymail/internal/adapter/dovecot"
	"easymail/internal/runtime"
	"easymail/pkg/config"
	"easymail/pkg/heartbeat"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"
)

type dovecotRunner struct {
	label     string
	srv       *dovecot.DovecotServer
	ln        net.Listener
	ctx       context.CancelFunc
	log       *easylog.Logger
	heartbeat *HeartbeatManager
}

func (r *dovecotRunner) Name() string { return r.label }

func (r *dovecotRunner) Start() error {
	go func() {
		if err := r.srv.Serve(context.Background(), r.ln); err != nil && r.ctx == nil {
			r.log.Errorf("%s", appi18n.LogMessage(appi18n.KeyLogServerError, map[string]interface{}{"Err": err.Error()}))
		}
	}()
	return nil
}

func (r *dovecotRunner) Stop() error {
	return r.ln.Close()
}

// RegisterHeartbeat registers this runner with the global heartbeat manager
func (r *dovecotRunner) RegisterHeartbeat(mgr *heartbeat.ServiceManager) {
	if mgr == nil || r.heartbeat == nil || !r.heartbeat.IsEnabled() {
		return
	}
	mgr.Register(r.heartbeat.ServiceName(), r.heartbeat.Interval(), r.log)
}

// NewDovecotRunner returns (nil, nil) when dovecot is disabled and listenOverride is empty.
func NewDovecotRunner(rt *runtime.Runtime, cfg config.DovecotConfig, modLog *easylog.Logger, listenOverride string) (Runner, error) {
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
			return nil, fmt.Errorf("dovecot: unix listen path is empty")
		}
		addr = "127.0.0.1:10025"
	}

	ln, err := listen(fam, addr)
	if err != nil {
		return nil, fmt.Errorf("dovecot listen %s %s: %w", fam, addr, err)
	}

	domainRepo := rt.MailDomainRepo
	accountRepo := rt.MailUserRepo
	srv := dovecot.New(accountRepo, domainRepo)
	srv.SetLogger(modLog)
	if cfg.Parameter != nil {
		if v, ok := cfg.Parameter["debug"]; ok {
			if b, err := strconv.ParseBool(v); err == nil {
				srv.SetDebug(b)
			}
		}
	}

	modLog.Infof("%s", appi18n.LogMessage(appi18n.KeyLogListen, map[string]interface{}{"Fam": fam, "Addr": addr}))

	label := "dovecot"

	// Create heartbeat manager (disabled if interval is 0 or redis is nil)
	heartbeatInterval := cfg.HeartbeatIntervalSec
	if heartbeatInterval <= 0 {
		heartbeatInterval = rt.Config.HeartbeatIntervalSec
	}
	var heartbeatMgr *HeartbeatManager
	if heartbeatInterval > 0 {
		heartbeatMgr = NewHeartbeatManager(
			true,
			time.Duration(heartbeatInterval)*time.Second,
			heartbeat.DefaultTTL,
			"dovecot",
		)
	}

	return &dovecotRunner{
		label:     label,
		srv:       srv,
		ln:        ln,
		ctx:       nil,
		log:       modLog,
		heartbeat: heartbeatMgr,
	}, nil
}
