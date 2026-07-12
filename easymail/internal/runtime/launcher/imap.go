package launcher

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	easymailimap "easymail/internal/adapter/imap"
	managementSvc "easymail/internal/app/management"
	mailservice "easymail/internal/domain/messaging/service"
	"easymail/internal/domain/messaging/storagepath"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence/sqlite"
	"easymail/internal/runtime"
	"easymail/pkg/config"
	"easymail/pkg/heartbeat"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"
)

type imapLogger struct{ log *easylog.Logger }

func (l imapLogger) Printf(format string, args ...interface{}) {
	l.log.Infof(format, args...)
}

type imapRunner struct {
	srv       *easymailimap.Server
	ln        net.Listener
	log       *easylog.Logger
	heartbeat *HeartbeatManager
}

func (r *imapRunner) Name() string { return "imap" }

func (r *imapRunner) Start() error {
	go func() {
		if err := r.srv.Serve(r.ln); err != nil && !strings.Contains(err.Error(), "closed") {
			r.log.Errorf("%s", appi18n.LogMessage(appi18n.KeyLogServerError, map[string]interface{}{"Err": err.Error()}))
		}
	}()
	return nil
}

func (r *imapRunner) Stop() error {
	_ = r.ln.Close()
	return nil
}

func (r *imapRunner) RegisterHeartbeat(mgr *heartbeat.ServiceManager) {
	if mgr == nil || !r.heartbeat.IsEnabled() {
		return
	}
	mgr.Register(r.heartbeat.ServiceName(), r.heartbeat.Interval(), r.log)
}

func NewIMAPRunner(rt *runtime.Runtime, cfg config.IMAPConfig, listenOverride string, modLog *easylog.Logger) (Runner, error) {
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
			return nil, fmt.Errorf("imap: unix listen path is empty")
		}
		addr = "127.0.0.1:1143"
	}

	ln, err := listen(fam, addr)
	if err != nil {
		return nil, fmt.Errorf("imap listen %s %s: %w", fam, addr, err)
	}

	// --- domain repos ---
	domainRepo := rt.MailDomainRepo
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
	folderRepo := sqlite.NewFolderRepository(pool, getDataPath)
	emailRepo := sqlite.NewMailIndexRepository(getDataPath, pool)

	// --- application services ---
	authSvc := managementSvc.NewMailUserAuthService(accountRepo, domainRepo)

	// --- domain backend ---
	backend := mailservice.NewService(getDataPath, folderRepo, emailRepo, nil, nil, appCfg.MailStorage.RootForStorage(0), &appCfg.SMTP)

	// --- TLS ---
	var tlsCfg *tls.Config
	if (cfg.TLSEnable || cfg.StartTLS) && cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("imap tls load cert: %w", err)
		}
		tlsCfg = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	}
	var directTLS, startTLSCfg *tls.Config
	if tlsCfg != nil {
		if cfg.TLSEnable {
			directTLS = tlsCfg
		}
		if cfg.StartTLS {
			startTLSCfg = tlsCfg
		}
	}

	// --- IMAP server ---
	srv := easymailimap.NewServer(&easymailimap.ServerOptions{
		TLSConfig:      directTLS,
		StartTLSConfig: startTLSCfg,
		NewMailSession: func() *easymailimap.MailSession {
			return easymailimap.NewMailSession(easymailimap.SessionDeps{
				Auth: authSvc,
				Mail: backend,
				Ctx:  context.Background(),
				TLS:  tlsCfg,
			})
		},
		Logger:        imapLogger{log: modLog},
		MaxConcurrent: 256,
		Debug:         cfg.Debug,
	})

	modLog.Infof("%s", appi18n.LogMessage(appi18n.KeyLogListen, map[string]interface{}{"Fam": fam, "Addr": addr}))

	// --- heartbeat ---
	var heartbeatMgr *HeartbeatManager
	if rt.Config.HeartbeatIntervalSec > 0 {
		heartbeatMgr = NewHeartbeatManager(
			true,
			time.Duration(rt.Config.HeartbeatIntervalSec)*time.Second,
			heartbeat.DefaultTTL,
			"imap",
		)
	}

	return &imapRunner{
		srv:       srv,
		ln:        ln,
		log:       modLog,
		heartbeat: heartbeatMgr,
	}, nil
}
