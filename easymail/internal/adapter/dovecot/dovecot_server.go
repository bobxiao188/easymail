// Package dovecot implements a Dovecot authentication protocol server adapter.
package dovecot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"easymail/internal/domain/management"
	"easymail/pkg/logger/easylog"
)

const (
	maxAuthFailures = 10
	lookupTimeout   = 10 * time.Second
)

type DovecotServer struct {
	name         string
	_log         *easylog.Logger
	debug        bool
	count        int64
	userRepo     management.MailUserRepository
	domainRepo   management.MailDomainRepository
	authFailures sync.Map
}

func New(userRepo management.MailUserRepository, domainRepo management.MailDomainRepository) *DovecotServer {
	return &DovecotServer{
		name:       "dovecot-auth-service",
		userRepo:   userRepo,
		domainRepo: domainRepo,
		_log:       easylog.NewDiscardLogger(),
	}
}

func (svr *DovecotServer) SetLogger(_log *easylog.Logger) { svr._log = _log }
func (svr *DovecotServer) SetDebug(debug bool)            { svr.debug = debug }

func (svr *DovecotServer) Serve(ctx context.Context, ln net.Listener) error {
	svr._log.Infof("%s starting on %s", svr.name, ln.Addr())
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				svr._log.Errorf("accept error: %v", err)
				continue
			}
		}
		go svr.Handle(conn)
	}
}

func (svr *DovecotServer) Handle(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			if svr._log != nil {
				svr._log.Errorf("panic in dovecot handle: %v", r)
			}
		}
		conn.Close()
	}()
	atomic.AddInt64(&svr.count, 1)
	scan := bufio.NewScanner(conn)
	writer := bufio.NewWriter(conn)
	handshakeDone := false
	cookie := fmt.Sprintf("%x", atomic.LoadInt64(&svr.count))

	for scan.Scan() {
		line := scan.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "get ") {
			key := strings.TrimSpace(strings.TrimPrefix(line, "get "))
			svr.handleTcpLookup(writer, key)
		_ = writer.Flush()
		continue
		}
		fields := strings.Split(line, "\t")
		cmd := fields[0]
		if !handshakeDone {
			if cmd == "VERSION" || cmd == "CPID" {
				if cmd == "CPID" {
					svr.sendServerHello(writer, cookie)
					handshakeDone = true
				}
				continue
			}
		}
		switch cmd {
		case "AUTH":
			svr.handleAuth(conn, writer, fields)
		case "USER":
			svr.handleUserLookup(writer, fields)
		case "QUIT":
			return
		default:
			if len(fields) > 1 {
				fmt.Fprintf(writer, "FAIL\t%s\terror=unknown_command\n", fields[1])
			}
		}
		writer.Flush()
	}
	if err := scan.Err(); err != nil {
		svr._log.Errorf("dovecot scan error: %v", err)
	}
}

func (svr *DovecotServer) sendServerHello(w *bufio.Writer, cookie string) {
	fmt.Fprintf(w, "VERSION\t1\t1\n")
	fmt.Fprintf(w, "MECH\tPLAIN\tplaintext\n")
	fmt.Fprintf(w, "SPID\t%d\n", os.Getpid())
	fmt.Fprintf(w, "CUID\t%d\n", atomic.LoadInt64(&svr.count))
	fmt.Fprintf(w, "COOKIE\t%s\n", cookie)
	fmt.Fprintf(w, "DONE\n")
	w.Flush()
}

func (svr *DovecotServer) remoteIP(conn net.Conn) string {
	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		return addr.IP.String()
	}
	return conn.RemoteAddr().String()
}

func (svr *DovecotServer) handleAuth(conn net.Conn, w *bufio.Writer, fields []string) {
	if len(fields) < 4 {
		return
	}
	id := fields[1]

	ip := svr.remoteIP(conn)
	if v, ok := svr.authFailures.Load(ip); ok {
		if failures, ok := v.(int); ok && failures >= maxAuthFailures {
			svr._log.Warnf("rate limit exceeded for %s, rejecting auth", ip)
			fmt.Fprintf(w, "FAIL\t%s\tuser=*\n", id)
			return
		}
	}

	params := parseParams(fields[3:])
	respB64, ok := params["resp"]
	if !ok {
		fmt.Fprintf(w, "FAIL\t%s\terror=missing_resp\n", id)
		return
	}
	data, err := base64.StdEncoding.DecodeString(respB64)
	if err != nil {
		fmt.Fprintf(w, "FAIL\t%s\terror=bad_base64\n", id)
		return
	}
	parts := bytes.SplitN(data, []byte{0}, 3)
	if len(parts) < 3 {
		fmt.Fprintf(w, "FAIL\t%s\terror=invalid_plain_format\n", id)
		return
	}
	user := string(parts[1])
	if user == "" {
		user = string(parts[0])
	}
	pass := string(parts[2])

	err = svr.authenticate(svr.timeoutCtx(), user, pass)
	if err != nil {
		svr._log.Warnf("SASL auth failed for %s: %v", user, err)
		fmt.Fprintf(w, "FAIL\t%s\tuser=%s\n", id, user)
		actual, _ := svr.authFailures.LoadOrStore(ip, 0)
		svr.authFailures.Store(ip, actual.(int)+1)
	} else {
		fmt.Fprintf(w, "OK\t%s\tuser=%s\n", id, user)
	}
}

func (svr *DovecotServer) authenticate(ctx context.Context, email, password string) error {
	user, err := svr.userRepo.FindByFullEmail(ctx, email)
	if err != nil {
		return err
	}
	if !user.Validate() {
		return fmt.Errorf("user inactive")
	}
	if domainStr := extractDomain(email); domainStr != "" {
		if _, err := svr.domainRepo.FindValidatedByName(ctx, domainStr); err != nil {
			return fmt.Errorf("domain inactive")
		}
	}
	if !user.VerifyPassword(password) {
		return fmt.Errorf("invalid password")
	}
	return nil
}

func (svr *DovecotServer) handleUserLookup(w *bufio.Writer, fields []string) {
	if len(fields) < 3 {
		return
	}
	id := fields[1]
	userField := fields[2]
	err := svr.validateDomain(svr.timeoutCtx(), extractDomain(userField))
	if err == nil {
		fmt.Fprintf(w, "OK\t%s\tuser=%s\n", id, userField)
	} else {
		fmt.Fprintf(w, "FAIL\t%s\n", id)
	}
}

func (svr *DovecotServer) validateDomain(ctx context.Context, domain string) error {
	_, err := svr.domainRepo.FindValidatedByName(ctx, domain)
	return err
}

func (svr *DovecotServer) handleTcpLookup(w *bufio.Writer, key string) {
	key = strings.ToLower(strings.TrimSpace(key))
	domain := key
	if at := strings.LastIndex(key, "@"); at >= 0 {
		domain = key[at+1:]
	}
	_, err := svr.domainRepo.FindValidatedByName(svr.timeoutCtx(), domain)
	if err == nil {
		fmt.Fprintf(w, "200 OK\n")
	} else {
		fmt.Fprintf(w, "500 Not found\n")
		if svr._log != nil {
			svr._log.Warnf("TCP Lookup FAILED: key=[%s] domain=[%s] err=%v", key, domain, err)
		}
	}
}

func parseParams(items []string) map[string]string {
	res := make(map[string]string)
	for _, item := range items {
		k, v, ok := strings.Cut(item, "=")
		if ok {
			res[k] = v
		}
	}
	return res
}

func extractDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at >= 0 && at < len(email)-1 {
		return strings.ToLower(email[at+1:])
	}
	return ""
}

func (svr *DovecotServer) timeoutCtx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), lookupTimeout)
	return ctx
}
