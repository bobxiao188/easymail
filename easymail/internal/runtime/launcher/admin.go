package launcher

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	appAdmin "easymail/internal/app/admin"
	filterSvc "easymail/internal/app/filter"
	managementSvc "easymail/internal/app/management"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence/mysql"
	"easymail/internal/infrastructure/persistence/sqlite"
	"easymail/internal/portal/admin"
	"easymail/internal/portal/admin/handler"
	"easymail/internal/runtime"
	"easymail/pkg/heartbeat"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"
)

type adminRunner struct {
	srv          *http.Server
	log          *easylog.Logger
	heartbeat    *HeartbeatManager
	rollupCancel context.CancelFunc
	tlsEnable    bool
}

func (r *adminRunner) Name() string { return "admin" }

func (r *adminRunner) Start() error {
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

func (r *adminRunner) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if r.rollupCancel != nil {
		r.rollupCancel()
	}
	return r.srv.Shutdown(ctx)
}

// RegisterHeartbeat registers this runner with the global heartbeat manager
func (r *adminRunner) RegisterHeartbeat(mgr *heartbeat.ServiceManager) {
	if mgr == nil || !r.heartbeat.IsEnabled() {
		return
	}
	mgr.Register(r.heartbeat.ServiceName(), r.heartbeat.Interval(), r.log)
}

func newAdminRunner(rt *runtime.Runtime, logger *easylog.Logger) (Runner, error) {
	// Collect storage partition IDs from config
	var storageIDs []int
	for _, p := range rt.Config.MailStorage.Local {
		storageIDs = append(storageIDs, p.StorageID)
	}

	// SQLite pool for per-user mailbox databases
	pool := sqlite.NewPool(sqlite.Config{WAL: true})

	// Build app-layer services
	authenticationSvc := appAdmin.NewAuthenticationService(
		rt.AdminUserRepo,
		rt.Config.Admin.JWT.Secret,
		rt.Config.Admin.JWT.ExpireHours,
	)

	// Provision service must be created before MailDomainService (needed for PurgeDomain).
	provisionSvc := managementSvc.NewUserProvisionService(pool,
		func(ctx context.Context, uid shared.GlobalID) (root, dataPath string, err error) {
			user, err := rt.MailUserRepo.FindByID(ctx, uid)
			if err != nil {
				return "", "", err
			}
			if user == nil {
				return "", "", fmt.Errorf("mail user %s not found", uid)
			}
			return rt.Config.MailStorage.RootForStorage(user.StorageID), user.DataPath, nil
		})
	mailDomainSvc := managementSvc.NewMailDomainService(rt.MailDomainRepo, rt.MailUserRepo, provisionSvc, rt.Config.MailStorage.RootForStorage(0), logger)

	mailAccountSvc := managementSvc.NewMailUserService(rt.MailUserRepo,
		rt.Config.MailStorage.RootForStorage(0), provisionSvc, logger)
	classifyModelSvc := filterSvc.NewClassifyModelService(rt.DB,
		rt.Config.Classifier.FastTextExecutable,
		rt.Config.Classifier.ModelRoot,
		rt.Config.Classifier.ONNXRuntimeLib)
	publicSampleSvc, publicSampleCategorySvc := filterSvc.NewPublicSampleService(rt.DB)
	trainingSvc := filterSvc.NewTrainingService(rt.DB, rt.Config.Classifier.FastTextExecutable, rt.Config.Classifier.ModelRoot)
	filterAdminSvc := appAdmin.NewFilterAdminService(rt.DB)
	dashboardSvc := appAdmin.NewDashboardService(rt.RedisClient, rt.DB)

	// Build Postfix configuration management service
	pdbp := mysql.NewPersistenceDBProvider(mysql.NewStaticDB(rt.DB))
	postfixAgentRepo := mysql.NewPostfixAgentRepository(pdbp)
	postfixConfigRepo := mysql.NewPostfixConfigRepository(pdbp)
	postfixDeliveryLogRepo := mysql.NewPostfixDeliveryLogRepository(pdbp)
	// MailDomainRepo already available as rt.MailDomainRepo, but we need the
	// concrete type for the repository. Since it's an interface, we can use it directly.
	postfixSvc := managementSvc.NewPostfixConfigService(
		postfixAgentRepo,
		postfixConfigRepo,
		postfixDeliveryLogRepo,
		rt.MailDomainRepo, // MailDomainRepository (interface)
		rt.Config,
	)

	// Build portal handler
	h := handler.New(
		storageIDs,
		authenticationSvc,
		mailDomainSvc,
		mailAccountSvc,
		provisionSvc,
		classifyModelSvc,
		publicSampleSvc,
		publicSampleCategorySvc,
		filterAdminSvc,
		dashboardSvc,
		postfixSvc,
		trainingSvc,
		logger,
	)

	// Admin HTTP router (Gin)
	router := admin.SetupRouter(rt.Config, h)

	// --- TLS ---
	cfg := rt.Config.Admin
	certFile := cfg.CertFile
	if certFile == "" {
		certFile = cfg.CertificateFile
	}
	keyFile := cfg.KeyFile
	if keyFile == "" {
		keyFile = cfg.CertificateKey
	}
	tlsEnabled := cfg.TLSEnable && certFile != "" && keyFile != ""

	// Create heartbeat manager (disabled if interval is 0 or redis is nil)
	var heartbeatMgr *HeartbeatManager
	if rt.Config.HeartbeatIntervalSec > 0 {
		heartbeatMgr = NewHeartbeatManager(
			true,
			time.Duration(rt.Config.HeartbeatIntervalSec)*time.Second,
			heartbeat.DefaultTTL,
			"admin",
		)
	}

	srv := &http.Server{
		Addr:    cfg.Listen,
		Handler: router,
	}

	if tlsEnabled {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("admin tls load cert: %w", err)
		}
		srv.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	return &adminRunner{
		srv:          srv,
		log:          logger,
		heartbeat:    heartbeatMgr,
		rollupCancel: nil,
		tlsEnable:    tlsEnabled,
	}, nil
}
