package launcher

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	managementSvc "easymail/internal/app/management"
	webmailApp "easymail/internal/app/webmail"
	mailservice "easymail/internal/domain/messaging/service"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/filter/signers"
	"easymail/internal/infrastructure/persistence"
	"easymail/internal/infrastructure/persistence/mysql"
	"easymail/internal/infrastructure/persistence/sqlite"
	"easymail/internal/portal/webmail"
	"easymail/internal/portal/webmail/handler"
	"easymail/internal/runtime"
	"easymail/pkg/heartbeat"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"
)

type webmailRunner struct {
	srv       *http.Server
	log       *easylog.Logger
	heartbeat *HeartbeatManager
	tlsEnable bool
}

func (r *webmailRunner) Name() string { return "webmail" }

func (r *webmailRunner) Start() error {
	go func() {
		addr := r.srv.Addr
		if r.tlsEnable {
			r.log.Infof("%s", appi18n.LogMessage(appi18n.KeyLogHTTPSListen, map[string]interface{}{"Addr": addr}))
			if err := r.srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				r.log.Errorf("%s", appi18n.LogMessage(appi18n.KeyLogServerError, map[string]interface{}{"Err": err.Error()}))
			}
		} else {
			r.log.Infof("%s", appi18n.LogMessage(appi18n.KeyLogHTTPListen, map[string]interface{}{"Addr": addr}))
			if err := r.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				r.log.Errorf("%s", appi18n.LogMessage(appi18n.KeyLogServerError, map[string]interface{}{"Err": err.Error()}))
			}
		}
	}()
	return nil
}

func (r *webmailRunner) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.srv.Shutdown(ctx)
}

func (r *webmailRunner) RegisterHeartbeat(mgr *heartbeat.ServiceManager) {
	if mgr == nil || !r.heartbeat.IsEnabled() {
		return
	}
	mgr.Register(r.heartbeat.ServiceName(), r.heartbeat.Interval(), r.log)
}

func newWebmailRunner(rt *runtime.Runtime, logger *easylog.Logger) (Runner, error) {
	domainRepo := rt.MailDomainRepo
	accountRepo := rt.MailUserRepo
	appCfg := rt.Config

	// --- mail storage ---
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
		return appCfg.MailStorage.RootForStorage(u.StorageID), u.DataPath, nil
	}

	// --- domain repos (per-user SQLite) ---
	folderRepo := sqlite.NewFolderRepository(pool, getDataPath)
	emailRepo := sqlite.NewMailIndexRepository(getDataPath, pool)
	contactRepo := sqlite.NewUserContactRepository(pool, getDataPath)
	contactGroupRepo := sqlite.NewUserContactGroupRepository(pool, getDataPath)

	// --- auth backend ---
	authBackend := managementSvc.NewMailUserAuthService(accountRepo, domainRepo)

	// --- mail backend ---
	mailUserFinder := mailservice.MailUserFinderFunc(func(ctx context.Context, email string) (shared.GlobalID, error) {
		u, err := accountRepo.FindByFullEmail(ctx, email)
		if err != nil {
			return "", err
		}
		return u.ID, nil
	})
	dkimSigner := signers.NewDKIMSigner(domainRepo)
	backend := mailservice.NewService(getDataPath, folderRepo, emailRepo, mailUserFinder, dkimSigner, appCfg.MailStorage.RootForStorage(0), &appCfg.SMTP)

	// --- webmail app services ---
	authSvc := webmailApp.NewAuthService(authBackend, appCfg.Webmail.JWT.Secret, appCfg.Webmail.JWT.ExpireHours)
	mailSvc := webmailApp.NewMailService(backend)
	contactSvc := webmailApp.NewContactService(contactRepo, contactGroupRepo)

	gormProvider := persistence.NewGormProvider(rt.DB)
	settingsRepo := mysql.NewUserSettingsRepository(gormProvider)
	profileSvc := webmailApp.NewProfileService(settingsRepo)

	// --- handler & router ---
	h := handler.New(authSvc, mailSvc, contactSvc, profileSvc, logger)
	router := webmail.SetupRouter(appCfg, h)

	// --- heartbeat ---
	var heartbeatMgr *HeartbeatManager
	if rt.Config.HeartbeatIntervalSec > 0 {
		heartbeatMgr = NewHeartbeatManager(
			true,
			time.Duration(rt.Config.HeartbeatIntervalSec)*time.Second,
			heartbeat.DefaultTTL,
			"webmail",
		)
	}

	// --- TLS ---
	webCfg := appCfg.Webmail
	certFile := webCfg.CertFile
	if certFile == "" {
		certFile = webCfg.CertificateFile
	}
	keyFile := webCfg.KeyFile
	if keyFile == "" {
		keyFile = webCfg.CertificateKey
	}
	tlsEnabled := webCfg.TLSEnable && certFile != "" && keyFile != ""

	srv := &http.Server{
		Addr:    webCfg.Listen,
		Handler: router,
	}

	if tlsEnabled {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("webmail tls load cert: %w", err)
		}
		srv.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	return &webmailRunner{
		srv:       srv,
		log:       logger,
		heartbeat: heartbeatMgr,
		tlsEnable: tlsEnabled,
	}, nil
}
