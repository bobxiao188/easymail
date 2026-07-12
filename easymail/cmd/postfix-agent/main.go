// Postfix Agent — runs on Postfix servers to receive config from EasyMail and manage Postfix.
//
// Uses postconf -e to inject EasyMail-managed parameters into main.cf one by one,
// preserving all other configurations. Easymail.cf is kept as a backup/source-of-truth
// for the managed params but is not directly loaded by Postfix.
//
// Endpoints:
//
//	GET    /api/v1/agent/status        — Postfix process + config status
//	POST   /api/v1/agent/config/push   — Receive and stage config (easymail.cf)
//	POST   /api/v1/agent/config/apply  — backup → postconf -e → reload
//	POST   /api/v1/agent/config/rollback — Restore last easymail.cf backup via postconf -e
//
// Authentication: X-Agent-Token header (pre-shared, set via --token or env AGENT_TOKEN)
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	listenAddr   string
	agentToken   string
	postfixDir   string
	stagingDir   string
	backupDir    string
	logDir       string
	allowedIPs   string // comma-separated IP/CIDR list, empty = allow all
	agentVersion = "1.0.0"
)

// stagedConfig holds the most recently pushed configuration.
var (
	mu               sync.RWMutex
	stagedEasymailCf string
)

// easymailCfFilename is the name of the EasyMail-managed config file in staging and postfixDir.
const easymailCfFilename = "easymail.cf"

// ipAllowList holds parsed allowed networks
var ipAllowList []*net.IPNet

// sessionLogger writes detailed session logs
var sessionLogger *log.Logger

func main() {
	flag.StringVar(&listenAddr, "listen", ":8081", "Agent HTTP listen address (env: LISTEN_ADDR)")
	flag.StringVar(&agentToken, "token", "", "Pre-shared auth token (env: AGENT_TOKEN)")
	flag.StringVar(&postfixDir, "postfix-dir", "/etc/postfix", "Postfix configuration directory (env: POSTFIX_DIR)")
	flag.StringVar(&stagingDir, "staging-dir", "/tmp/easymail-staging", "Staging directory for config before apply (env: STAGING_DIR)")
	flag.StringVar(&backupDir, "backup-dir", "/etc/postfix/backups", "Backup directory for rollback (env: BACKUP_DIR)")
	flag.StringVar(&logDir, "log-dir", "/var/log/easymail-agent", "Directory for session logs (env: LOG_DIR)")
	flag.StringVar(&allowedIPs, "allowed-ips", "", "Comma-separated list of allowed IPs/CIDRs (env: ALLOWED_IPS, empty = allow all)")
	flag.Parse()

	// Env fallbacks (flag takes precedence)
	if listenAddr == ":8081" {
		if v := os.Getenv("LISTEN_ADDR"); v != "" {
			listenAddr = v
		}
	}
	if agentToken == "" {
		agentToken = os.Getenv("AGENT_TOKEN")
	}
	if agentToken == "" {
		log.Fatal("agent token is required: set --token or AGENT_TOKEN env")
	}
	if postfixDir == "/etc/postfix" {
		if v := os.Getenv("POSTFIX_DIR"); v != "" {
			postfixDir = v
		}
	}
	if stagingDir == "/tmp/easymail-staging" {
		if v := os.Getenv("STAGING_DIR"); v != "" {
			stagingDir = v
		}
	}
	if backupDir == "/etc/postfix/backups" {
		if v := os.Getenv("BACKUP_DIR"); v != "" {
			backupDir = v
		}
	}
	if logDir == "/var/log/easymail-agent" {
		if v := os.Getenv("LOG_DIR"); v != "" {
			logDir = v
		}
	}
	if allowedIPs == "" {
		allowedIPs = os.Getenv("ALLOWED_IPS")
	}

	// Ensure directories exist
	for _, dir := range []string{stagingDir, backupDir, logDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("cannot create directory %s: %v", dir, err)
		}
	}

	// Parse allowed IPs
	if allowedIPs != "" {
		ipAllowList = parseAllowedIPs(allowedIPs)
		if len(ipAllowList) == 0 {
			log.Fatal("invalid --allowed-ips format")
		}
		log.Printf("IP allowlist loaded: %d entries", len(ipAllowList))
	} else {
		log.Println("WARNING: No IP allowlist configured, all IPs allowed")
	}

	// Setup session logger
	logFile, err := os.OpenFile(filepath.Join(logDir, "sessions.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("cannot open session log file: %v", err)
	}
	sessionLogger = log.New(logFile, "[SESSION] ", log.LstdFlags|log.Lmicroseconds)

	mux := http.NewServeMux()
	// Status
	mux.HandleFunc("/api/v1/agent/status", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(statusHandler))))

	// Config management
	mux.HandleFunc("/api/v1/agent/config/push", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(pushConfigHandler))))
	mux.HandleFunc("/api/v1/agent/config/apply", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(applyConfigHandler))))
	mux.HandleFunc("/api/v1/agent/config/rollback", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(rollbackConfigHandler))))

	// Queue management
	mux.HandleFunc("/api/v1/agent/queue/list", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(listQueueHandler))))
	mux.HandleFunc("/api/v1/agent/queue/stats", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(getQueueStatsHandler))))
	mux.HandleFunc("/api/v1/agent/queue/delete", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(deleteQueueMessagesHandler))))
	mux.HandleFunc("/api/v1/agent/queue/resend", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(resendQueueMessagesHandler))))
	mux.HandleFunc("/api/v1/agent/queue/flush", sessionLogMiddleware(ipAllowMiddleware(authMiddleware(flushQueueHandler))))

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down agent...")
		server.Close()
	}()

	log.Printf("Postfix Agent v%s starting on %s", agentVersion, listenAddr)
	log.Printf("Postfix config dir: %s", postfixDir)
	log.Printf("Staging dir: %s", stagingDir)
	log.Printf("Backup dir: %s", backupDir)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

// authMiddleware validates the X-Agent-Token header.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Agent-Token")
		if token == "" || token != agentToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ipAllowMiddleware restricts access to allowed IPs only.
func ipAllowMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no allowlist is configured, allow all
		if len(ipAllowList) == 0 {
			next(w, r)
			return
		}

		// Extract client IP
		clientIP := parseClientIP(r)
		if clientIP == nil {
			http.Error(w, `{"error":"cannot determine client IP"}`, http.StatusBadGateway)
			return
		}

		// Check if IP is in allowlist
		allowed := false
		for _, network := range ipAllowList {
			if network.Contains(clientIP) {
				allowed = true
				break
			}
		}

		if !allowed {
			sessionLogger.Printf("REJECTED ip=%s path=%s", clientIP, r.URL.Path)
			http.Error(w, `{"error":"forbidden: IP not allowed"}`, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// sessionLogMiddleware logs detailed request/response information.
func sessionLogMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		clientIP := parseClientIP(r)

		// Wrap response writer to capture status code
		wrapped := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next(wrapped, r)

		duration := time.Since(start)
		sessionLogger.Printf("completed ip=%s method=%s path=%s status=%d duration=%s",
			clientIP, r.Method, r.URL.Path, wrapped.statusCode, duration.Round(time.Millisecond))
	}
}

type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// ==================== Status ====================

type statusResponse struct {
	PostfixRunning bool   `json:"postfixRunning"`
	ConfigHash     string `json:"configHash"`
	LastReloadAt   string `json:"lastReloadAt,omitempty"`
	PostfixVersion string `json:"postfixVersion,omitempty"`
	AgentVersion   string `json:"agentVersion,omitempty"`
	Uptime         string `json:"uptime,omitempty"`
}

var startedAt = time.Now()

func statusHandler(w http.ResponseWriter, r *http.Request) {
	resp := statusResponse{
		AgentVersion: agentVersion,
		Uptime:       time.Since(startedAt).Round(time.Second).String(),
	}

	// Check if postfix is running
	resp.PostfixRunning = isPostfixRunning()

	// Read easymail.cf config hash (the source-of-truth for EasyMail-managed params)
	if easymailCf, err := os.ReadFile(filepath.Join(postfixDir, easymailCfFilename)); err == nil {
		h := sha256.New()
		h.Write(easymailCf)
		resp.ConfigHash = fmt.Sprintf("%x", h.Sum(nil))
	}

	// Try postfix version
	if ver, err := exec.Command("postconf", "-d", "mail_version").Output(); err == nil {
		resp.PostfixVersion = strings.TrimSpace(string(ver))
	}

	writeJSON(w, http.StatusOK, resp)
}

// ==================== Push Config ====================

type pushConfigRequest struct {
	MainCf string `json:"mainCf"`
}

func pushConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req pushConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.MainCf == "" {
		writeError(w, http.StatusBadRequest, "mainCf is required")
		return
	}

	// 1. Write easymail.cf to staging directory
	if err := os.WriteFile(filepath.Join(stagingDir, easymailCfFilename), []byte(req.MainCf), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write staging "+easymailCfFilename+": "+err.Error())
		return
	}

	// 2. Build staging/main.cf by merging current production main.cf with easymail params.
	// This is used by postfix -c staging check in the apply step.
	merged, err := mergeParamsToMainCf(req.MainCf)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "build staging main.cf: "+err.Error())
		return
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "main.cf"), []byte(merged), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write staging main.cf: "+err.Error())
		return
	}

	// Copy master.cf from production (we don't manage it, but postfix check needs it)
	copyFile(filepath.Join(postfixDir, "master.cf"), filepath.Join(stagingDir, "master.cf"))

	// 3. Cache in memory for apply
	mu.Lock()
	stagedEasymailCf = req.MainCf
	mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"status": "staged"})
}

// ==================== Apply Config ====================

func applyConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	mu.RLock()
	easymailCf := stagedEasymailCf
	mu.RUnlock()

	if easymailCf == "" {
		writeError(w, http.StatusBadRequest, "no staged config; push config first")
		return
	}

	// 1. Write staging files for validation (rewrite for safety)
	if err := os.WriteFile(filepath.Join(stagingDir, easymailCfFilename), []byte(easymailCf), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write staging "+easymailCfFilename+": "+err.Error())
		return
	}
	merged, err := mergeParamsToMainCf(easymailCf)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "build staging main.cf: "+err.Error())
		return
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "main.cf"), []byte(merged), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write staging main.cf: "+err.Error())
		return
	}

	// Copy master.cf from production (we don't manage it, but postfix check needs it)
	copyFile(filepath.Join(postfixDir, "master.cf"), filepath.Join(stagingDir, "master.cf"))

	// 2. Run postfix check against staging directory
	cmd := exec.Command("postfix", "-c", stagingDir, "check")
	if output, err := cmd.CombinedOutput(); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("postfix check failed:\n%s", string(output)))
		return
	}

	// 3. Backup current easymail.cf
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, timestamp)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "create backup dir: "+err.Error())
		return
	}
	copyFile(filepath.Join(postfixDir, easymailCfFilename), filepath.Join(backupPath, easymailCfFilename))

	// 4. Write new easymail.cf to production (source-of-truth for rollback)
	if err := os.WriteFile(filepath.Join(postfixDir, easymailCfFilename), []byte(easymailCf), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write "+easymailCfFilename+": "+err.Error())
		return
	}

	// 5. Apply each parameter via postconf -e (only modifies EasyMail-managed params)
	params := parseParams(easymailCf)
	if err := applyParams(params); err != nil {
		writeError(w, http.StatusInternalServerError, "postconf -e failed: "+err.Error())
		return
	}

	// 6. postfix reload
	reload := exec.Command("postfix", "reload")
	if output, err := reload.CombinedOutput(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("postfix reload failed:\n%s", string(output)))
		return
	}

	// 7. Cleanup old backups (keep last 10)
	cleanupOldBackups()

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "applied",
		"backup": timestamp,
	})
}

// ==================== Rollback ====================

func rollbackConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Find the latest backup
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read backup dir: "+err.Error())
		return
	}

	var latest string
	var latestTime time.Time
	for _, e := range entries {
		if e.IsDir() {
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(latestTime) {
				latest = e.Name()
				latestTime = info.ModTime()
			}
		}
	}
	if latest == "" {
		writeError(w, http.StatusNotFound, "no backup found")
		return
	}

	backupPath := filepath.Join(backupDir, latest)

	// Check backup easymail.cf exists
	backupCfPath := filepath.Join(backupPath, easymailCfFilename)
	if _, err := os.Stat(backupCfPath); os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, "backup "+easymailCfFilename+" not found")
		return
	}

	// Read backup config
	backupCf, err := os.ReadFile(backupCfPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read backup "+easymailCfFilename+": "+err.Error())
		return
	}

	// Validate: merge backup params with current main.cf → staging check
	merged, err := mergeParamsToMainCf(string(backupCf))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "build staging main.cf for rollback check: "+err.Error())
		return
	}
	rollbackCheckDir := filepath.Join(backupPath, ".check")
	os.MkdirAll(rollbackCheckDir, 0755)
	os.WriteFile(filepath.Join(rollbackCheckDir, easymailCfFilename), backupCf, 0644)
	os.WriteFile(filepath.Join(rollbackCheckDir, "main.cf"), []byte(merged), 0644)

	cmd := exec.Command("postfix", "-c", rollbackCheckDir, "check")
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(rollbackCheckDir)
		writeError(w, http.StatusBadRequest, fmt.Sprintf("backup postfix check failed:\n%s", string(output)))
		return
	}
	os.RemoveAll(rollbackCheckDir)

	// Write backup to production easymail.cf
	if err := os.WriteFile(filepath.Join(postfixDir, easymailCfFilename), backupCf, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "restore "+easymailCfFilename+": "+err.Error())
		return
	}

	// Apply backup params via postconf -e
	params := parseParams(string(backupCf))
	if err := applyParams(params); err != nil {
		writeError(w, http.StatusInternalServerError, "postconf -e rollback failed: "+err.Error())
		return
	}

	reload := exec.Command("postfix", "reload")
	if output, err := reload.CombinedOutput(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("postfix reload after rollback failed:\n%s", string(output)))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "rolled_back",
		"backup": latest,
	})
}

// ==================== Helpers ====================

// paramEntry holds a single parsed Postfix parameter from easymail.cf.
type paramEntry struct {
	Name  string
	Value string
}

// parseParams extracts parameter name/value pairs from easymail.cf content.
// Lines starting with # are comments; only lines matching "name = value" or "name=value" are parsed.
func parseParams(content string) []paramEntry {
	var params []paramEntry
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split on first = sign
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])
		if name == "" {
			continue
		}
		params = append(params, paramEntry{Name: name, Value: value})
	}
	return params
}

// applyParams injects params into main.cf via postconf -e.
func applyParams(params []paramEntry) error {
	for _, p := range params {
		// postconf -e only modifies the specified parameter; others are left intact.
		cmd := exec.Command("postconf", "-e", fmt.Sprintf("%s = %s", p.Name, p.Value))
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("postconf -e %s: %s\n%s", p.Name, err, string(output))
		}
	}
	return nil
}

// readCurrentMainCf reads the current /etc/postfix/main.cf content.
func readCurrentMainCf() string {
	data, err := os.ReadFile(filepath.Join(postfixDir, "main.cf"))
	if err != nil {
		return ""
	}
	return string(data)
}

// mergeParamsToMainCf reads the current main.cf, removes any lines that define
// the same params as easymailCf, and appends the easymail params at the end.
// The result is a complete main.cf that represents what would exist after
// postconf -e is applied, suitable for postfix -c staging check.
func mergeParamsToMainCf(easymailCf string) (string, error) {
	params := parseParams(easymailCf)

	// Build a set of param names to remove from main.cf
	removeSet := make(map[string]bool, len(params))
	for _, p := range params {
		removeSet[p.Name] = true
	}

	// Read current main.cf
	currentCf := readCurrentMainCf()

	// Filter out managed param lines, preserving all other content (comments, blank lines, etc.)
	var keptLines []string
	for _, line := range strings.Split(currentCf, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			keptLines = append(keptLines, line)
			continue
		}
		// Check if this line defines a managed param
		eqIdx := strings.IndexByte(trimmed, '=')
		if eqIdx < 0 {
			keptLines = append(keptLines, line)
			continue
		}
		name := strings.TrimSpace(trimmed[:eqIdx])
		if removeSet[name] {
			// Skip this line; it will be replaced by the easymail param below
			continue
		}
		keptLines = append(keptLines, line)
	}

	// Append easymail params at the end
	// Add a header comment to demarcate EasyMail-managed section
	keptLines = append(keptLines, "")
	keptLines = append(keptLines, "# === EasyMail managed config ===")
	for _, p := range params {
		keptLines = append(keptLines, fmt.Sprintf("%s = %s", p.Name, p.Value))
	}
	keptLines = append(keptLines, "# === End EasyMail managed config ===")

	return strings.Join(keptLines, "\n"), nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

func cleanupOldBackups() {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	type backupEntry struct {
		name string
		time time.Time
	}

	var backups []backupEntry
	for _, e := range entries {
		if e.IsDir() {
			info, err := e.Info()
			if err != nil {
				continue
			}
			backups = append(backups, backupEntry{name: e.Name(), time: info.ModTime()})
		}
	}

	// Sort by time descending (newest first)
	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[j].time.After(backups[i].time) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Keep last 10
	keep := 10
	if len(backups) > keep {
		for _, b := range backups[keep:] {
			os.RemoveAll(filepath.Join(backupDir, b.name))
		}
	}
}

// isPostfixRunning checks whether Postfix is running using multiple methods in priority order.
func isPostfixRunning() bool {
	// 1. systemctl is-active (most reliable on systemd systems)
	if err := exec.Command("systemctl", "is-active", "--quiet", "postfix").Run(); err == nil {
		return true
	}

	// 2. postfix status output
	if output, _ := exec.Command("postfix", "status").CombinedOutput(); strings.Contains(string(output), "is running") {
		return true
	}

	// 3. ps -C master (check for postfix master process)
	if output, err := exec.Command("ps", "-C", "master", "-o", "comm=").CombinedOutput(); err == nil && strings.TrimSpace(string(output)) == "master" {
		return true
	}

	return false
}

// parseAllowedIPs parses a comma-separated list of IPs and CIDRs.
func parseAllowedIPs(allowed string) []*net.IPNet {
	var networks []*net.IPNet
	parts := strings.Split(allowed, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// If it's a CIDR notation
		if strings.Contains(part, "/") {
			_, ipNet, err := net.ParseCIDR(part)
			if err != nil {
				log.Printf("WARNING: invalid CIDR %s: %v", part, err)
				continue
			}
			networks = append(networks, ipNet)
		} else {
			// Single IP, convert to /32 or /128
			ip := net.ParseIP(part)
			if ip == nil {
				log.Printf("WARNING: invalid IP %s", part)
				continue
			}

			if ip.To4() != nil {
				// IPv4
				ipNet := &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(32, 32),
				}
				networks = append(networks, ipNet)
			} else {
				// IPv6
				ipNet := &net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(128, 128),
				}
				networks = append(networks, ipNet)
			}
		}
	}

	return networks
}

// ==================== Queue Management Data Structures ====================

// queueMessage represents a single message in the Postfix queue.
type queueMessage struct {
	QueueID    string   `json:"queueId"`
	Size       int      `json:"size"`
	Age        string   `json:"age"`
	Sender     string   `json:"sender"`
	Recipients []string `json:"recipients"`
	Status     string   `json:"status"`
	StatusText string   `json:"statusText"`
}

// queueStats provides summary statistics for the mail queue.
type queueStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Deferred int `json:"deferred"`
	Held     int `json:"held"`
}

// queueListResponse contains the paginated list of queue messages.
type queueListResponse struct {
	Messages []queueMessage `json:"messages"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

// ==================== Queue Management Handlers ====================

// listQueueHandler returns the list of messages in the queue.
func listQueueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status") // active, deferred, held, all
	sender := r.URL.Query().Get("sender")
	recipient := r.URL.Query().Get("recipient")
	queueID := r.URL.Query().Get("queueId")

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	page := 1
	pageSize := 100

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 500 {
			pageSize = ps
		}
	}

	// Get queue messages
	messages, err := getQueueMessages()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get queue: "+err.Error())
		return
	}

	// Apply filters
	filtered := filterMessages(messages, status, sender, recipient, queueID)

	// Calculate pagination
	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paginated := filtered[start:end]

	writeJSON(w, http.StatusOK, queueListResponse{
		Messages: paginated,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// getQueueStatsHandler returns queue statistics.
func getQueueStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	stats, err := getQueueStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get queue stats: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// deleteQueueMessagesHandler deletes specified messages from the queue.
func deleteQueueMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req struct {
		MessageIDs []string `json:"messageIds"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.MessageIDs) == 0 {
		writeError(w, http.StatusBadRequest, "messageIds is required")
		return
	}

	if err := deleteQueueMessages(req.MessageIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete messages: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// resendQueueMessagesHandler resends specified messages in the queue.
func resendQueueMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req struct {
		MessageIDs []string `json:"messageIds"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(req.MessageIDs) == 0 {
		writeError(w, http.StatusBadRequest, "messageIds is required")
		return
	}

	if err := resendQueueMessages(req.MessageIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resend messages: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "resent"})
}

// flushQueueHandler flushes the entire queue.
func flushQueueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if err := flushQueue(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to flush queue: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "flushed"})
}

// ==================== Queue Management Helpers ====================

// postqueueJSON represents the JSON output from postqueue -j
type postqueueJSON struct {
	QueueName    string                   `json:"queue_name"`
	QueueID      string                   `json:"queue_id"`
	ArrivalTime  int64                    `json:"arrival_time"`
	MessageSize  int                      `json:"message_size"`
	ForcedExpire bool                     `json:"forced_expire"`
	Sender       string                   `json:"sender"`
	Recipients   []postqueueJSONRecipient `json:"recipients"`
}

// postqueueJSONRecipient represents a recipient in the JSON output
type postqueueJSONRecipient struct {
	Address      string `json:"address"`
	OrigAddress  string `json:"orig_address"`
	DelayReason  string `json:"delay_reason"`
	BounceReason string `json:"bounce_reason"`
}

// getQueueMessages executes postqueue -j and parses JSON output.
func getQueueMessages() ([]queueMessage, error) {
	cmd := exec.Command("postqueue", "-j")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("postqueue -j failed: %s", string(output))
	}

	return parseQueueJSON(string(output)), nil
}

// parseQueueJSON parses the JSON LINES output from postqueue -j
func parseQueueJSON(output string) []queueMessage {
	var messages []queueMessage

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var jsonMsg postqueueJSON
		if err := json.Unmarshal([]byte(line), &jsonMsg); err != nil {
			log.Printf("Warning: failed to parse JSON line: %v", err)
			continue
		}

		// Convert queue_name to status
		status := jsonMsg.QueueName
		if status == "hold" {
			status = "held"
		}

		// Extract recipients
		recipients := make([]string, 0, len(jsonMsg.Recipients))
		var statusText string
		for _, r := range jsonMsg.Recipients {
			recipients = append(recipients, r.Address)
			if r.DelayReason != "" {
				if statusText == "" {
					statusText = r.DelayReason
				} else {
					statusText += "; " + r.DelayReason
				}
			}
			if r.BounceReason != "" {
				if statusText == "" {
					statusText = r.BounceReason
				} else {
					statusText += "; " + r.BounceReason
				}
			}
		}

		// Calculate age from arrival_time
		age := formatAge(jsonMsg.ArrivalTime)

		msg := queueMessage{
			QueueID:    jsonMsg.QueueID,
			Size:       jsonMsg.MessageSize,
			Age:        age,
			Sender:     jsonMsg.Sender,
			Recipients: recipients,
			Status:     status,
			StatusText: statusText,
		}
		messages = append(messages, msg)
	}

	return messages
}

// formatAge formats the arrival time as a human-readable age string
func formatAge(arrivalTime int64) string {
	now := time.Now().Unix()
	diff := now - arrivalTime
	if diff < 0 {
		diff = 0
	}

	if diff < 60 {
		return fmt.Sprintf("%ds", diff)
	} else if diff < 3600 {
		return fmt.Sprintf("%dm", diff/60)
	} else if diff < 86400 {
		return fmt.Sprintf("%dh", diff/3600)
	} else {
		return fmt.Sprintf("%dd", diff/86400)
	}
}

// filterMessages filters messages based on criteria.
func filterMessages(messages []queueMessage, status, sender, recipient, queueID string) []queueMessage {
	var filtered []queueMessage

	for _, msg := range messages {
		// Filter by status
		if status != "" && status != "all" && msg.Status != status {
			continue
		}

		// Filter by sender
		if sender != "" && !strings.Contains(strings.ToLower(msg.Sender), strings.ToLower(sender)) {
			continue
		}

		// Filter by recipient
		if recipient != "" {
			found := false
			for _, rec := range msg.Recipients {
				if strings.Contains(strings.ToLower(rec), strings.ToLower(recipient)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by queue ID
		if queueID != "" && msg.QueueID != queueID {
			continue
		}

		filtered = append(filtered, msg)
	}

	return filtered
}

// getQueueStats executes postqueue -j and counts messages by status.
func getQueueStats() (*queueStats, error) {
	cmd := exec.Command("postqueue", "-j")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("postqueue -j failed: %s", string(output))
	}

	return parseQueueStatsJSON(string(output)), nil
}

// parseQueueStatsJSON parses the JSON LINES output and counts by queue_name.
func parseQueueStatsJSON(output string) *queueStats {
	stats := &queueStats{}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var jsonMsg postqueueJSON
		if err := json.Unmarshal([]byte(line), &jsonMsg); err != nil {
			continue
		}

		stats.Total++
		switch jsonMsg.QueueName {
		case "active":
			stats.Active++
		case "deferred":
			stats.Deferred++
		case "hold":
			stats.Held++
		}
	}

	return stats
}

// deleteQueueMessages deletes specified messages using postsuper -d.
func deleteQueueMessages(messageIDs []string) error {
	// postsuper -d message1 message2 ...
	args := append([]string{"-d"}, messageIDs...)
	cmd := exec.Command("postsuper", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postsuper -d failed: %s", string(output))
	}
	return nil
}

// resendQueueMessages resends specified messages using postsuper -r.
func resendQueueMessages(messageIDs []string) error {
	// postsuper -r message1 message2 ...
	args := append([]string{"-r"}, messageIDs...)
	cmd := exec.Command("postsuper", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postsuper -r failed: %s", string(output))
	}
	return nil
}

// flushQueue flushes the queue using postqueue -f.
func flushQueue() error {
	cmd := exec.Command("postqueue", "-f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("postqueue -f failed: %s", string(output))
	}
	return nil
}

// parseClientIP extracts the client IP from the request, considering proxy headers.
func parseClientIP(r *http.Request) net.IP {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		parts := strings.Split(xff, ",")
		ip := net.ParseIP(strings.TrimSpace(parts[0]))
		if ip != nil {
			return ip
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := net.ParseIP(strings.TrimSpace(xri))
		if ip != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return net.ParseIP(r.RemoteAddr)
	}
	return net.ParseIP(host)
}
