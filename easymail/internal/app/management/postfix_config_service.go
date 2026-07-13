// Package management provides application-level services for Postfix configuration management.
package management

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	domain "easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/pkg/config"
)

// variableRegex matches $section.field patterns in parameter values.
var variableRegex = regexp.MustCompile(`\$\{([^}]+)\}|\$([a-zA-Z_][a-zA-Z0-9_.]*)`)

// resolveVariables replaces $section.field variables in paramValue with actual values
// from the EasyMail application configuration.
// When a service listens on 0.0.0.0, the IP is replaced with postfixHost.
func resolveVariables(cfg *config.AppConfig, postfixHost string, paramValue string) string {
	// Build available variables map from AppConfig
	variables := map[string]string{
		// Service listen addresses
		"dovecot.listen": cfg.Dovecot.Listen,
		"lmtp.listen":    cfg.LMTP.Listen,
		"milter.listen":  cfg.Milter.Listen,
		"imap.listen":    cfg.IMAP.Listen,
		"admin.listen":   cfg.Admin.Listen,
		"webmail.listen": cfg.Webmail.Listen,

		// Service family (tcp/unix)
		"dovecot.family": cfg.Dovecot.Family,
		"lmtp.family":    cfg.LMTP.Family,
		"milter.family":  cfg.Milter.Family,
		"imap.family":    cfg.IMAP.Family,

		// Postfix host (for resolving 0.0.0.0)
		"postfix.host": postfixHost,

		// Storage path
		"storage.root": cfg.MailStorage.RootForStorage(0),
	}

	resolver := func(varName string) string {
		val, ok := variables[varName]
		if !ok {
			return ""
		}
		// If the listen address starts with 0.0.0.0, replace with postfixHost
		if strings.HasPrefix(val, "0.0.0.0:") && postfixHost != "" {
			val = postfixHost + strings.TrimPrefix(val, "0.0.0.0")
		}
		return val
	}

	// Replace ${section.field} pattern first (braced form)
	result := variableRegex.ReplaceAllStringFunc(paramValue, func(match string) string {
		var varName string
		if strings.HasPrefix(match, "${") {
			varName = match[2 : len(match)-1]
		} else {
			varName = match[1:]
		}
		resolved := resolver(varName)
		if resolved == "" {
			return match // Keep unresolved variable as-is
		}
		return resolved
	})

	return result
}

// getAvailableVariables returns a map of available variable names and their current resolved values.
func getAvailableVariables(cfg *config.AppConfig, postfixHost string) map[string]string {
	variables := map[string]string{
		"dovecot.listen": cfg.Dovecot.Listen,
		"lmtp.listen":    cfg.LMTP.Listen,
		"milter.listen":  cfg.Milter.Listen,
		"imap.listen":    cfg.IMAP.Listen,
		"admin.listen":   cfg.Admin.Listen,
		"webmail.listen": cfg.Webmail.Listen,
		"dovecot.family": cfg.Dovecot.Family,
		"lmtp.family":    cfg.LMTP.Family,
		"milter.family":  cfg.Milter.Family,
		"imap.family":    cfg.IMAP.Family,
		"postfix.host":   postfixHost,
		"storage.root":   cfg.MailStorage.RootForStorage(0),
	}

	// Resolve 0.0.0.0 → postfixHost for display
	resolved := make(map[string]string, len(variables))
	for k, v := range variables {
		if strings.HasPrefix(v, "0.0.0.0:") && postfixHost != "" {
			v = postfixHost + strings.TrimPrefix(v, "0.0.0.0")
		}
		resolved[k] = v
	}
	return resolved
}

// PostfixSettings holds global Postfix configuration settings.
type PostfixSettings struct {
	// EasyMailHost is the IP address Postfix uses to reach EasyMail services.
	// When services listen on 0.0.0.0, this IP replaces 0.0.0.0 in resolved addresses.
	// Default: 127.0.0.1
	EasyMailHost string `json:"easymailHost"`
}

// PostfixConfigService orchestrates Postfix configuration generation and delivery.
type PostfixConfigService interface {
	// Agent management
	ListAgents(ctx context.Context, keyword string, page, pageSize int) ([]domain.PostfixAgent, int64, error)
	GetAgent(ctx context.Context, id shared.GlobalID) (*domain.PostfixAgent, error)
	CreateAgent(ctx context.Context, name, host, token, description string) (*domain.PostfixAgent, error)
	UpdateAgent(ctx context.Context, id shared.GlobalID, name, host, token, description string, enabled bool) error
	DeleteAgent(ctx context.Context, id shared.GlobalID) error

	// Agent status
	CheckAgentStatus(ctx context.Context, agentID shared.GlobalID) (*domain.AgentStatusInfo, error)

	// Config parameter management
	ListConfigParams(ctx context.Context, keyword string, page, pageSize int) ([]domain.PostfixConfig, int64, error)
	GetConfigParam(ctx context.Context, id shared.GlobalID) (*domain.PostfixConfig, error)
	CreateConfigParam(ctx context.Context, paramName, paramValue, description string) (*domain.PostfixConfig, error)
	UpdateConfigParam(ctx context.Context, id shared.GlobalID, paramValue string) error
	DeleteConfigParam(ctx context.Context, id shared.GlobalID) error
	GetManagedParams(ctx context.Context) ([]domain.PostfixConfig, error)

	// Global settings
	GetSettings(ctx context.Context) (*PostfixSettings, error)
	UpdateSettings(ctx context.Context, settings *PostfixSettings) error

	// Available variables for parameter values
	GetVariables(ctx context.Context) map[string]string

	// Local IP addresses
	GetLocalIPs(ctx context.Context) []string

	// Configuration generation and delivery
	GeneratePreview(ctx context.Context) (*ConfigPreview, error)
	GenerateInstallScript(ctx context.Context) (string, error)
	PushConfig(ctx context.Context, agentID shared.GlobalID) error
	ApplyConfig(ctx context.Context, agentID shared.GlobalID) error
	RollbackConfig(ctx context.Context, agentID shared.GlobalID) error
	PushAndApply(ctx context.Context, agentID shared.GlobalID) error

	// Delivery logs
	ListDeliveryLogs(ctx context.Context, agentID shared.GlobalID, limit int) ([]domain.PostfixDeliveryLog, error)

	// Status summary
	GetConfigStatusSummary(ctx context.Context) (*ConfigStatusSummary, error)

	// Queue management
	ListQueue(ctx context.Context, agentID shared.GlobalID, filter *domain.QueueFilter) (*domain.QueueListResponse, error)
	GetQueueStats(ctx context.Context, agentID shared.GlobalID) (*domain.QueueStats, error)
	DeleteQueueMessages(ctx context.Context, agentID shared.GlobalID, messageIDs []string) error
	ResendQueueMessages(ctx context.Context, agentID shared.GlobalID, messageIDs []string) error
	FlushQueue(ctx context.Context, agentID shared.GlobalID) error
}

// ConfigPreview holds the rendered configuration text for preview.
type ConfigPreview struct {
	MainCf      string `json:"mainCf"`
	ConfigHash  string `json:"configHash"`
	DomainCount int    `json:"domainCount"`
}

// ConfigStatusSummary provides an overview of all agents' configuration status.
type ConfigStatusSummary struct {
	Agents []AgentConfigStatus `json:"agents"`
}

// AgentConfigStatus holds per-agent config status.
type AgentConfigStatus struct {
	AgentID    string `json:"agentId"`
	AgentName  string `json:"agentName"`
	Host       string `json:"host"`
	Online     bool   `json:"online"`
	LastSyncAt string `json:"lastSyncAt,omitempty"`
	ConfigHash string `json:"configHash,omitempty"`
	UpToDate   bool   `json:"upToDate"`
	LastError  string `json:"lastError,omitempty"`
}

// agentHTTPClient communicates with a remote Postfix Agent.
type agentHTTPClient struct {
	httpClient *http.Client
}

func newAgentHTTPClient() *agentHTTPClient {
	return &agentHTTPClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type postfixConfigServiceImpl struct {
	agentRepo    domain.PostfixAgentRepository
	configRepo   domain.PostfixConfigRepository
	deliveryRepo domain.PostfixDeliveryLogRepository
	domainRepo   domain.MailDomainRepository
	easymailCfg  *config.AppConfig
	httpClient   *agentHTTPClient
}

// NewPostfixConfigService creates a new PostfixConfigService.
func NewPostfixConfigService(
	agentRepo domain.PostfixAgentRepository,
	configRepo domain.PostfixConfigRepository,
	deliveryRepo domain.PostfixDeliveryLogRepository,
	domainRepo domain.MailDomainRepository,
	easymailCfg *config.AppConfig,
) PostfixConfigService {
	return &postfixConfigServiceImpl{
		agentRepo:    agentRepo,
		configRepo:   configRepo,
		deliveryRepo: deliveryRepo,
		domainRepo:   domainRepo,
		easymailCfg:  easymailCfg,
		httpClient:   newAgentHTTPClient(),
	}
}

// ==================== Agent Management ====================

func (s *postfixConfigServiceImpl) ListAgents(ctx context.Context, keyword string, page, pageSize int) ([]domain.PostfixAgent, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.agentRepo.Search(ctx, keyword, page, pageSize)
}

func (s *postfixConfigServiceImpl) GetAgent(ctx context.Context, id shared.GlobalID) (*domain.PostfixAgent, error) {
	return s.agentRepo.FindByID(ctx, id)
}

func (s *postfixConfigServiceImpl) CreateAgent(ctx context.Context, name, host, token, description string) (*domain.PostfixAgent, error) {
	agent, err := domain.NewPostfixAgent(name, host, token, description)
	if err != nil {
		return nil, err
	}
	if err := s.agentRepo.Save(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (s *postfixConfigServiceImpl) UpdateAgent(ctx context.Context, id shared.GlobalID, name, host, token, description string, enabled bool) error {
	agent, err := s.agentRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	agent.Name = name
	agent.Host = host
	if token != "" {
		agent.Token = token
	}
	agent.Description = description
	agent.Enabled = enabled
	return s.agentRepo.Save(ctx, agent)
}

func (s *postfixConfigServiceImpl) DeleteAgent(ctx context.Context, id shared.GlobalID) error {
	return s.agentRepo.Delete(ctx, id)
}

// ==================== Agent Status ====================

func (s *postfixConfigServiceImpl) CheckAgentStatus(ctx context.Context, agentID shared.GlobalID) (*domain.AgentStatusInfo, error) {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	status, err := s.queryAgentStatus(agent)
	if err != nil {
		// Mark agent offline on error
		_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusOffline))
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusOnline))
	return status, nil
}

// ==================== Config Parameter Management ====================

func (s *postfixConfigServiceImpl) ListConfigParams(ctx context.Context, keyword string, page, pageSize int) ([]domain.PostfixConfig, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.configRepo.Search(ctx, keyword, page, pageSize)
}

func (s *postfixConfigServiceImpl) GetConfigParam(ctx context.Context, id shared.GlobalID) (*domain.PostfixConfig, error) {
	return s.configRepo.FindByID(ctx, id)
}

func (s *postfixConfigServiceImpl) CreateConfigParam(ctx context.Context, paramName, paramValue, description string) (*domain.PostfixConfig, error) {
	cfg, err := domain.NewPostfixConfig(paramName, paramValue, description)
	if err != nil {
		return nil, err
	}
	if err := s.configRepo.Save(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *postfixConfigServiceImpl) UpdateConfigParam(ctx context.Context, id shared.GlobalID, paramValue string) error {
	cfg, err := s.configRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if cfg.IsManaged {
		return domain.ErrPostfixConfigNotEditable
	}
	cfg.ParamValue = paramValue
	return s.configRepo.Save(ctx, cfg)
}

func (s *postfixConfigServiceImpl) DeleteConfigParam(ctx context.Context, id shared.GlobalID) error {
	return s.configRepo.Delete(ctx, id)
}

func (s *postfixConfigServiceImpl) GetManagedParams(ctx context.Context) ([]domain.PostfixConfig, error) {
	return s.configRepo.FindAllManaged(ctx)
}

// ==================== Global Settings ====================

// settingsParamName is the reserved parameter name for storing Postfix global settings.
const settingsParamName = "__postfix_settings__"

func (s *postfixConfigServiceImpl) GetSettings(ctx context.Context) (*PostfixSettings, error) {
	// Try to read from database first
	existing, err := s.configRepo.FindByParamName(ctx, settingsParamName)
	if err == nil && existing != nil {
		// ParamValue stores JSON-serialized PostfixSettings
		settings := &PostfixSettings{}
		if err := json.Unmarshal([]byte(existing.ParamValue), settings); err == nil {
			return settings, nil
		}
	}
	// Fallback to YAML config
	return &PostfixSettings{
		EasyMailHost: s.easymailCfg.Postfix.EasyMailHost,
	}, nil
}

func (s *postfixConfigServiceImpl) UpdateSettings(ctx context.Context, settings *PostfixSettings) error {
	// Serialize settings to JSON
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("serialize settings: %w", err)
	}

	// Upsert: try to find existing, create if not found
	existing, err := s.configRepo.FindByParamName(ctx, settingsParamName)
	if err != nil {
		// Not found, create new
		cfg, err := domain.NewPostfixConfig(settingsParamName, string(data), "Postfix global settings (EasyMailHost)")
		if err != nil {
			return err
		}
		cfg.IsManaged = true
		cfg.Enabled = false // Hidden from normal param list
		return s.configRepo.Save(ctx, cfg)
	}
	// Update existing
	existing.ParamValue = string(data)
	return s.configRepo.Save(ctx, existing)
}

func (s *postfixConfigServiceImpl) GetVariables(ctx context.Context) map[string]string {
	// Get current settings
	settings, err := s.GetSettings(ctx)
	if err != nil {
		settings = &PostfixSettings{EasyMailHost: s.easymailCfg.Postfix.EasyMailHost}
	}
	host := settings.EasyMailHost
	if host == "" {
		host = s.easymailCfg.Postfix.EasyMailHost
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return getAvailableVariables(s.easymailCfg, host)
}

func (s *postfixConfigServiceImpl) GetLocalIPs(ctx context.Context) []string {
	ips := []string{"127.0.0.1"} // Always include localhost

	// Get network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	seen := make(map[string]bool)
	seen["127.0.0.1"] = true

	for _, iface := range ifaces {
		// Skip down interfaces and loopback (except lo for 127.0.0.1 which is already included)
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			// Parse and validate IP address
			ip := extractValidIPv4(addr.String())
			if ip != "" && !seen[ip] {
				seen[ip] = true
				ips = append(ips, ip)
			}
		}
	}

	return ips
}

// extractValidIPv4 extracts and validates an IPv4 address from an address string
// e.g., "192.168.1.1/24" -> "192.168.1.1", "192.168.1.1" -> "192.168.1.1"
// Returns empty string if not a valid IPv4 address
func extractValidIPv4(addr string) string {
	// Skip empty strings
	if strings.TrimSpace(addr) == "" {
		return ""
	}

	// Handle CIDR notation (e.g., "192.168.1.1/24")
	ipStr := addr
	if idx := strings.IndexByte(addr, '/'); idx > 0 {
		ipStr = addr[:idx]
	}

	// Check again after extracting
	if strings.TrimSpace(ipStr) == "" {
		return ""
	}

	// Parse the IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	// Ensure it's IPv4 (not IPv6)
	if ip.To4() == nil {
		return ""
	}

	// Convert to IPv4 string format
	ipv4 := ip.To4().String()

	// Skip invalid addresses:
	// - 0.0.0.0 (unspecified)
	// - 255.255.255.255 (broadcast)
	// - 169.254.x.x (link-local/APIPA, usually indicates DHCP failure)
	if ipv4 == "0.0.0.0" || ipv4 == "255.255.255.255" {
		return ""
	}
	if strings.HasPrefix(ipv4, "169.254.") {
		return ""
	}

	return ipv4
}

// ==================== Configuration Generation ====================

// assembleConfig collects all parameters from database, resolves variables, and renders templates.
func (s *postfixConfigServiceImpl) assembleConfig(ctx context.Context) (*ConfigPreview, error) {
	// 1. Get all config params from database (both managed and user-defined)
	allParams, err := s.configRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("read config params: %w", err)
	}

	// 2. Get settings from database to obtain easymailHost
	settings, err := s.GetSettings(ctx)
	if err != nil {
		settings = &PostfixSettings{EasyMailHost: s.easymailCfg.Postfix.EasyMailHost}
	}
	postfixHost := settings.EasyMailHost
	if postfixHost == "" {
		postfixHost = s.easymailCfg.Postfix.EasyMailHost
	}
	if postfixHost == "" {
		postfixHost = "127.0.0.1"
	}

	// 3. Get active domains for virtual_mailbox_domains
	domains, err := s.domainRepo.FindAllValidated(ctx)
	if err != nil {
		return nil, fmt.Errorf("read domains: %w", err)
	}
	domainNames := make([]string, 0, len(domains))
	for _, d := range domains {
		domainNames = append(domainNames, d.Name)
	}

	// 4. Resolve variables in param values using EasyMail config and easymailHost
	// This will replace 0.0.0.0 with postfixHost in listen addresses
	resolvedParams := make([]domain.PostfixConfig, len(allParams))
	for i, p := range allParams {
		cp := p
		cp.ParamValue = resolveVariables(s.easymailCfg, postfixHost, p.ParamValue)
		resolvedParams[i] = cp
	}

	// 4. Build template data
	data := map[string]any{
		"GeneratedAt": time.Now().Format(time.RFC3339),
		"Params":      resolvedParams,
		"DomainList":  domainNames,
		"HasDomains":  len(domainNames) > 0,
	}

	// 5. Render main.cf
	mainCfTmpl := template.Must(template.New("main.cf").Funcs(template.FuncMap{
		"join": func(sep string, slice []string) string {
			return strings.Join(slice, sep)
		},
	}).Parse(mainCfTemplate))
	var mainCfBuf bytes.Buffer
	if err := mainCfTmpl.Execute(&mainCfBuf, data); err != nil {
		return nil, fmt.Errorf("render main.cf: %w", err)
	}

	mainCfContent := mainCfBuf.String()

	// Compute hash
	h := sha256.New()
	h.Write([]byte(mainCfContent))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	return &ConfigPreview{
		MainCf:      mainCfContent,
		ConfigHash:  hash,
		DomainCount: len(domainNames),
	}, nil
}

// ==================== Config Delivery ====================

func (s *postfixConfigServiceImpl) GeneratePreview(ctx context.Context) (*ConfigPreview, error) {
	return s.assembleConfig(ctx)
}

// GenerateInstallScript generates a shell script that can be piped via "curl -s URL | sh"
// to apply the Postfix configuration without requiring the postfix-agent binary.
func (s *postfixConfigServiceImpl) GenerateInstallScript(ctx context.Context) (string, error) {
	preview, err := s.assembleConfig(ctx)
	if err != nil {
		return "", err
	}

	// Parse the rendered main.cf into individual param lines
	paramLines := make([]string, 0)
	for _, line := range strings.Split(preview.MainCf, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "virtual_mailbox_domains") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			name := strings.TrimSpace(line[:idx])
			if name != "" {
				paramLines = append(paramLines, line)
			}
		}
	}

	// Ensure mainCf ends with newline for clean heredoc syntax
	mainCf := preview.MainCf
	if !strings.HasSuffix(mainCf, "\n") {
		mainCf += "\n"
	}

	data := installScriptData{
		GeneratedAt: time.Now().Format(time.RFC3339),
		MainCf:      mainCf,
		Params:      paramLines,
		ConfigHash:  preview.ConfigHash,
	}

	tmpl := template.Must(template.New("install.sh").Parse(installScriptTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render install script: %w", err)
	}

	return buf.String(), nil
}

func (s *postfixConfigServiceImpl) PushConfig(ctx context.Context, agentID shared.GlobalID) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}
	preview, err := s.assembleConfig(ctx)
	if err != nil {
		return err
	}
	// Log delivery start
	dl := &domain.PostfixDeliveryLog{
		ID:             shared.NewGlobalID(),
		AgentID:        agentID,
		Action:         string(domain.DeliveryActionPush),
		Status:         string(domain.DeliveryStatusSuccess),
		ConfigSnapshot: preview.ConfigHash,
		CreatedAt:      time.Now(),
	}
	if err := s.pushToAgent(agent, preview.MainCf); err != nil {
		dl.Status = string(domain.DeliveryStatusFailed)
		dl.ErrorMessage = err.Error()
		_ = s.deliveryRepo.Save(ctx, dl)
		_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusError))
		return fmt.Errorf("push failed: %w", err)
	}
	_ = s.deliveryRepo.Save(ctx, dl)
	_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusOnline))
	return nil
}

func (s *postfixConfigServiceImpl) ApplyConfig(ctx context.Context, agentID shared.GlobalID) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}
	dl := &domain.PostfixDeliveryLog{
		ID:        shared.NewGlobalID(),
		AgentID:   agentID,
		Action:    string(domain.DeliveryActionApply),
		Status:    string(domain.DeliveryStatusSuccess),
		CreatedAt: time.Now(),
	}
	if err := s.applyOnAgent(agent); err != nil {
		dl.Status = string(domain.DeliveryStatusFailed)
		dl.ErrorMessage = err.Error()
		_ = s.deliveryRepo.Save(ctx, dl)
		_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusError))
		return fmt.Errorf("apply failed: %w", err)
	}
	_ = s.deliveryRepo.Save(ctx, dl)
	_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusOnline))
	return nil
}

func (s *postfixConfigServiceImpl) RollbackConfig(ctx context.Context, agentID shared.GlobalID) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}
	dl := &domain.PostfixDeliveryLog{
		ID:        shared.NewGlobalID(),
		AgentID:   agentID,
		Action:    string(domain.DeliveryActionRollback),
		Status:    string(domain.DeliveryStatusSuccess),
		CreatedAt: time.Now(),
	}
	if err := s.rollbackOnAgent(agent); err != nil {
		dl.Status = string(domain.DeliveryStatusFailed)
		dl.ErrorMessage = err.Error()
		_ = s.deliveryRepo.Save(ctx, dl)
		return fmt.Errorf("rollback failed: %w", err)
	}
	_ = s.deliveryRepo.Save(ctx, dl)
	_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusOnline))
	return nil
}

func (s *postfixConfigServiceImpl) PushAndApply(ctx context.Context, agentID shared.GlobalID) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}
	preview, err := s.assembleConfig(ctx)
	if err != nil {
		return err
	}
	dl := &domain.PostfixDeliveryLog{
		ID:             shared.NewGlobalID(),
		AgentID:        agentID,
		Action:         string(domain.DeliveryActionApply),
		Status:         string(domain.DeliveryStatusSuccess),
		ConfigSnapshot: preview.ConfigHash,
		CreatedAt:      time.Now(),
	}
	if err := s.pushToAgent(agent, preview.MainCf); err != nil {
		dl.Status = string(domain.DeliveryStatusFailed)
		dl.ErrorMessage = err.Error()
		_ = s.deliveryRepo.Save(ctx, dl)
		_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusError))
		return fmt.Errorf("push failed: %w", err)
	}
	if err := s.applyOnAgent(agent); err != nil {
		dl.Status = string(domain.DeliveryStatusFailed)
		dl.ErrorMessage = err.Error()
		_ = s.deliveryRepo.Save(ctx, dl)
		_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusError))
		return fmt.Errorf("apply failed: %w", err)
	}
	_ = s.deliveryRepo.Save(ctx, dl)
	_ = s.agentRepo.UpdateStatus(ctx, agentID, string(domain.AgentStatusOnline))
	return nil
}

// ==================== Delivery Logs ====================

func (s *postfixConfigServiceImpl) ListDeliveryLogs(ctx context.Context, agentID shared.GlobalID, limit int) ([]domain.PostfixDeliveryLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.deliveryRepo.FindByAgent(ctx, agentID, limit)
}

// ==================== Status Summary ====================

func (s *postfixConfigServiceImpl) GetConfigStatusSummary(ctx context.Context) (*ConfigStatusSummary, error) {
	agents, err := s.agentRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	summary := &ConfigStatusSummary{}
	for _, agent := range agents {
		status := AgentConfigStatus{
			AgentID:   string(agent.ID),
			AgentName: agent.Name,
			Host:      agent.Host,
			Online:    agent.LastStatus == string(domain.AgentStatusOnline),
		}
		if !agent.LastSyncAt.IsZero() {
			status.LastSyncAt = agent.LastSyncAt.Format(time.RFC3339)
		}
		// Try to query live status
		liveStatus, err := s.queryAgentStatus(&agent)
		if err == nil {
			status.ConfigHash = liveStatus.ConfigHash
		}
		summary.Agents = append(summary.Agents, status)
	}
	return summary, nil
}

// ==================== HTTP Communication with Agent ====================

type agentPushRequest struct {
	MainCf string `json:"mainCf"`
}

type agentApplyRequest struct{}

type agentStatusResponse struct {
	PostfixRunning bool   `json:"postfixRunning"`
	ConfigHash     string `json:"configHash"`
	LastReloadAt   string `json:"lastReloadAt,omitempty"`
	PostfixVersion string `json:"postfixVersion,omitempty"`
	AgentVersion   string `json:"agentVersion,omitempty"`
	Uptime         string `json:"uptime,omitempty"`
}

func (s *postfixConfigServiceImpl) buildAgentURL(agent *domain.PostfixAgent, path string) string {
	host := strings.TrimRight(agent.Host, "/")
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	return host + path
}

func (s *postfixConfigServiceImpl) agentRequest(agent *domain.PostfixAgent, method, url string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", agent.Token)
	return s.httpClient.httpClient.Do(req)
}

func (s *postfixConfigServiceImpl) pushToAgent(agent *domain.PostfixAgent, mainCf string) error {
	url := s.buildAgentURL(agent, "/api/v1/agent/config/push")
	resp, err := s.agentRequest(agent, http.MethodPost, url, &agentPushRequest{
		MainCf: mainCf,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("agent push rejected (%d): %s", resp.StatusCode, result.Error)
	}
	return nil
}

func (s *postfixConfigServiceImpl) applyOnAgent(agent *domain.PostfixAgent) error {
	url := s.buildAgentURL(agent, "/api/v1/agent/config/apply")
	resp, err := s.agentRequest(agent, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("agent apply rejected (%d): %s", resp.StatusCode, result.Error)
	}
	return nil
}

func (s *postfixConfigServiceImpl) rollbackOnAgent(agent *domain.PostfixAgent) error {
	url := s.buildAgentURL(agent, "/api/v1/agent/config/rollback")
	resp, err := s.agentRequest(agent, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("agent rollback rejected (%d): %s", resp.StatusCode, result.Error)
	}
	return nil
}

func (s *postfixConfigServiceImpl) queryAgentStatus(agent *domain.PostfixAgent) (*domain.AgentStatusInfo, error) {
	url := s.buildAgentURL(agent, "/api/v1/agent/status")
	resp, err := s.agentRequest(agent, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent status returned %d", resp.StatusCode)
	}
	var status agentStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &domain.AgentStatusInfo{
		PostfixRunning: status.PostfixRunning,
		ConfigHash:     status.ConfigHash,
		LastReloadAt:   status.LastReloadAt,
		PostfixVersion: status.PostfixVersion,
		AgentVersion:   status.AgentVersion,
		Uptime:         status.Uptime,
	}, nil
}

// ==================== Go Templates ====================

const mainCfTemplate = `# === EasyMail managed config ===
# Generated at: {{ .GeneratedAt }}
# Do not edit — managed by EasyMail postfix-agent.

{{range .Params}}{{if .Enabled}}{{.ParamName}} = {{.ParamValue}}
{{end}}{{end}}
# Virtual mailbox domains (auto-synced from EasyMail)
{{if .HasDomains}}virtual_mailbox_domains = {{ .DomainList | join ", " }}{{else}}# No active domains configured{{end}}
# === End EasyMail managed config ===
`

// installScriptData is the template data for the Postfix install shell script.
type installScriptData struct {
	GeneratedAt string
	MainCf      string
	Params      []string
	ConfigHash  string
}

const installScriptTemplate = `#!/bin/sh
# EasyMail Postfix auto-install script
# Generated at: {{ .GeneratedAt }}
# Usage: curl -s http://<easymail-admin>/api/v1/admin/postfix/install-script | sudo sh

set -e

POSTFIX_DIR="${POSTFIX_DIR:-/etc/postfix}"
BACKUP_DIR="${BACKUP_DIR:-/etc/postfix/backups}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"

# Ensure running as root
if [ "$(id -u)" -ne 0 ]; then
  echo "This script must be run as root. Try: sudo sh"
  exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR/$TIMESTAMP"

# Backup current easymail.cf if exists
if [ -f "$POSTFIX_DIR/easymail.cf" ]; then
  cp "$POSTFIX_DIR/easymail.cf" "$BACKUP_DIR/$TIMESTAMP/easymail.cf.bak"
  echo "Backed up current easymail.cf to $BACKUP_DIR/$TIMESTAMP/"
fi

# Write new config (heredoc with single-quoted delimiter prevents shell expansion)
cat > "$POSTFIX_DIR/easymail.cf" << 'EASYMAIL_EOF'
{{ .MainCf }}EASYMAIL_EOF
echo "Wrote easymail.cf to $POSTFIX_DIR/"

# Apply each parameter via postconf (heredoc ensures literal values, no injection risk)
{{range $i, $p := .Params}}postconf -e "$(cat <<'PARAM_{{$i}}'
{{$p}}
PARAM_{{$i}}"
{{end}}
# Validate configuration
echo "Validating Postfix configuration..."
postfix check
echo "Configuration validation passed."

# Reload Postfix
echo "Reloading Postfix..."
postfix reload
echo "Postfix reloaded successfully."

# Cleanup old backups (keep last 10)
ls -t "$BACKUP_DIR" 2>/dev/null | tail -n +11 | while read d; do
  rm -rf "$BACKUP_DIR/$d"
done

echo "=== EasyMail Postfix configuration applied successfully ==="
echo "Config hash: {{ .ConfigHash }}"
`

// ==================== Queue Management ====================

func (s *postfixConfigServiceImpl) ListQueue(ctx context.Context, agentID shared.GlobalID, filter *domain.QueueFilter) (*domain.QueueListResponse, error) {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Build query parameters
	url := s.buildAgentURL(agent, "/api/v1/agent/queue/list")
	
	// Create request with filter parameters
	reqURL := url + "?"
	if filter != nil {
		if filter.Status != "" {
			reqURL += "status=" + filter.Status + "&"
		}
		if filter.Sender != "" {
			reqURL += "sender=" + filter.Sender + "&"
		}
		if filter.Recipient != "" {
			reqURL += "recipient=" + filter.Recipient + "&"
		}
		if filter.QueueID != "" {
			reqURL += "queueId=" + filter.QueueID + "&"
		}
		if filter.Page > 0 {
			reqURL += fmt.Sprintf("page=%d&", filter.Page)
		}
		if filter.PageSize > 0 {
			reqURL += fmt.Sprintf("pageSize=%d&", filter.PageSize)
		}
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Agent-Token", agent.Token)

	resp, err := s.httpClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return nil, fmt.Errorf("agent rejected (%d): %s", resp.StatusCode, result.Error)
	}

	var queueResp domain.QueueListResponse
	if err := json.NewDecoder(resp.Body).Decode(&queueResp); err != nil {
		return nil, err
	}

	return &queueResp, nil
}

func (s *postfixConfigServiceImpl) GetQueueStats(ctx context.Context, agentID shared.GlobalID) (*domain.QueueStats, error) {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	url := s.buildAgentURL(agent, "/api/v1/agent/queue/stats")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Agent-Token", agent.Token)

	resp, err := s.httpClient.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return nil, fmt.Errorf("agent rejected (%d): %s", resp.StatusCode, result.Error)
	}

	var stats domain.QueueStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

func (s *postfixConfigServiceImpl) DeleteQueueMessages(ctx context.Context, agentID shared.GlobalID, messageIDs []string) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}

	url := s.buildAgentURL(agent, "/api/v1/agent/queue/delete")
	body := map[string][]string{
		"messageIds": messageIDs,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", agent.Token)

	resp, err := s.httpClient.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("agent rejected (%d): %s", resp.StatusCode, result.Error)
	}

	return nil
}

func (s *postfixConfigServiceImpl) ResendQueueMessages(ctx context.Context, agentID shared.GlobalID, messageIDs []string) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}

	url := s.buildAgentURL(agent, "/api/v1/agent/queue/resend")
	body := map[string][]string{
		"messageIds": messageIDs,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Token", agent.Token)

	resp, err := s.httpClient.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("agent rejected (%d): %s", resp.StatusCode, result.Error)
	}

	return nil
}

func (s *postfixConfigServiceImpl) FlushQueue(ctx context.Context, agentID shared.GlobalID) error {
	agent, err := s.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return err
	}

	url := s.buildAgentURL(agent, "/api/v1/agent/queue/flush")
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Agent-Token", agent.Token)

	resp, err := s.httpClient.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("agent rejected (%d): %s", resp.StatusCode, result.Error)
	}

	return nil
}

var _ PostfixConfigService = (*postfixConfigServiceImpl)(nil)

