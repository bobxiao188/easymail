package launcher

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"easymail/internal/adapter/lmtp"
	filtersvc "easymail/internal/app/filter"
	managementSvc "easymail/internal/app/management"
	"easymail/internal/domain/messaging/storagepath"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence/sqlite"
	"easymail/internal/runtime"
	"easymail/pkg/config"
	"easymail/pkg/heartbeat"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"
)

type lmtpRunner struct {
	name      string
	srv       *lmtp.Server
	ln        net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
	log       *easylog.Logger
	heartbeat *HeartbeatManager
}

func (r *lmtpRunner) Name() string { return r.name }

func (r *lmtpRunner) Start() error {
	go func() {
		if err := r.srv.Serve(r.ctx, r.ln); err != nil && r.ctx.Err() == nil {
			r.log.Errorf("%s", appi18n.LogMessage(appi18n.KeyLogServerError, map[string]interface{}{"Err": err.Error()}))
		}
	}()
	return nil
}

func (r *lmtpRunner) Stop() error {
	r.cancel()
	return r.ln.Close()
}

func (r *lmtpRunner) RegisterHeartbeat(mgr *heartbeat.ServiceManager) {
	if mgr == nil || !r.heartbeat.IsEnabled() {
		return
	}
	mgr.Register(r.heartbeat.ServiceName(), r.heartbeat.Interval(), r.log)
}

func NewLMTPRunner(rt *runtime.Runtime, cfg config.LMTPConfig, listenOverride string, modLog *easylog.Logger) (Runner, error) {
	if !cfg.Enable {
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
			return nil, fmt.Errorf("lmtp: unix listen path is empty")
		}
		addr = "127.0.0.1:20025"
	}

	ln, err := listen(fam, addr)
	if err != nil {
		return nil, fmt.Errorf("lmtp listen %s %s: %w", fam, addr, err)
	}

	// --- domain repos ---
	accountRepo := rt.MailUserRepo

	// --- mail storage ---
	appCfg := rt.Config
	sqCfg := sqlite.Config{
		BusyTimeoutMs: appCfg.MailStorage.SQLite.BusyTimeoutMs,
		MaxOpenConns:  appCfg.MailStorage.SQLite.MaxOpenConns,
		WAL:           appCfg.MailStorage.SQLite.WAL,
	}
	pool := sqlite.NewPool(sqCfg)

	getDataPath := func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error) {
		u, err := accountRepo.FindByID(ctx, uid)
		if err != nil {
			return "", "", err
		}
		dp := u.DataPath
		if dp == "" {
			dp = storagepath.MailUserDataPath(strings.SplitN(u.Email, "@", 2)[1], u.Email)
		}
		return appCfg.MailStorage.RootForStorage(u.StorageID), dp, nil
	}

	// --- infrastructure repos ---
	emailRepo := sqlite.NewMailIndexRepository(getDataPath, pool)

	// --- application services ---
	provisionSvc := managementSvc.NewUserProvisionService(pool, getDataPath)
	storageRoot := appCfg.MailStorage.RootForStorage(0)

	// --- LMTP server ---
	ctx, cancel := context.WithCancel(context.Background())
	srv := &lmtp.Server{
		Accounts:  accountRepo,
		Hostname:  "",
		Log:       modLog,
		InboundLMTP: &filtersvc.LMTPRouteOptions{
			Config: appCfg.Milter.Filter.ToFilterConfig(),
		},
		Provision: provisionSvc,
		EmailRepo: emailRepo,
		Root:      storageRoot,
	}

	modLog.Infof("%s", appi18n.LogMessage(appi18n.KeyLogListen, map[string]interface{}{"Fam": fam, "Addr": addr}))

	// --- heartbeat ---
	var heartbeatMgr *HeartbeatManager
	if rt.Config.HeartbeatIntervalSec > 0 {
		heartbeatMgr = NewHeartbeatManager(
			true,
			time.Duration(rt.Config.HeartbeatIntervalSec)*time.Second,
			heartbeat.DefaultTTL,
			"lmtp",
		)
	}

	return &lmtpRunner{
		name:      "lmtp",
		srv:       srv,
		ln:        ln,
		ctx:       ctx,
		cancel:    cancel,
		log:       modLog,
		heartbeat: heartbeatMgr,
	}, nil
}
