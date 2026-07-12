package launcher

import (
	"fmt"
	"time"

	"easymail/internal/runtime"
	"easymail/pkg/heartbeat"
	"easymail/pkg/logger/easylog"
	"easymail/pkg/logger/servicelog"
)

// BuildRunners constructs the list of service runners from config.
func BuildRunners(rt *runtime.Runtime, logger *easylog.Logger) ([]Runner, error) {
	var runners []Runner

	if rt.Config.Classifier.Enable {
		lg, err := servicelog.Open(logger, rt.Config.Classifier.Logs, "classifier")
		if err != nil {
			return nil, err
		}
		r, err := NewClassifierRunner(rt, rt.Config.Classifier, lg)
		if err != nil {
			return nil, err
		}
		if r != nil {
			runners = append(runners, r)
		}
	}

	if rt.Config.Dovecot.Enable {
		lg, err := servicelog.Open(logger, rt.Config.Dovecot.Logs, "dovecot")
		if err != nil {
			return nil, err
		}
		r, err := NewDovecotRunner(rt, rt.Config.Dovecot, lg, "")
		if err != nil {
			return nil, err
		}
		if r != nil {
			runners = append(runners, r)
		}
	}

	if rt.Config.LMTP.Enable {
		lg, err := servicelog.Open(logger, rt.Config.LMTP.Logs, "lmtp")
		if err != nil {
			return nil, err
		}
		r, err := NewLMTPRunner(rt, rt.Config.LMTP, "", lg)
		if err != nil {
			return nil, err
		}
		if r != nil {
			runners = append(runners, r)
		}
	}

	if rt.Config.Milter.Enable {
		lg, err := servicelog.Open(logger, rt.Config.Milter.Logs, "milter")
		if err != nil {
			return nil, err
		}
		r, err := NewMilterRunner(rt, rt.Config.Milter, "", lg)
		if err != nil {
			return nil, err
		}
		if r != nil {
			runners = append(runners, r)
		}
	}

	if rt.Config.Admin.Enable {
		lg, err := servicelog.Open(logger, rt.Config.Admin.Logs, "admin")
		if err != nil {
			return nil, err
		}
		r, err := newAdminRunner(rt, lg)
		if err != nil {
			return nil, err
		}
		runners = append(runners, r)
	}

	if rt.Config.Webmail.Enable {
		lg, err := servicelog.Open(logger, rt.Config.Webmail.Logs, "webmail")
		if err != nil {
			return nil, err
		}
		r, err := newWebmailRunner(rt, lg)
		if err != nil {
			return nil, err
		}
		runners = append(runners, r)
	}

	if rt.Config.IMAP.Enable {
		lg, err := servicelog.Open(logger, rt.Config.IMAP.Logs, "imap")
		if err != nil {
			return nil, err
		}
		r, err := NewIMAPRunner(rt, rt.Config.IMAP, "", lg)
		if err != nil {
			return nil, err
		}
		if r != nil {
			runners = append(runners, r)
		}
	}

	if len(runners) == 0 {
		return nil, fmt.Errorf("launcher: no service enabled (enable dovecot, lmtp, milter, admin, webmail, or imap in config)")
	}
	return runners, nil
}

// BuildManager creates a Manager with runners and optional heartbeat manager
func BuildManager(rt *runtime.Runtime, runners []Runner, logger *easylog.Logger) *Manager {
	var heartbeatMgr *heartbeat.ServiceManager

	if rt.Config.HeartbeatIntervalSec > 0 {
		heartbeatMgr = heartbeat.NewServiceManager(
			rt.RedisClient,
			time.Duration(rt.Config.HeartbeatIntervalSec)*time.Second,
			heartbeat.DefaultTTL,
			logger,
		)
	}

	return &Manager{
		Runners:      runners,
		HeartbeatMgr: heartbeatMgr,
	}
}
