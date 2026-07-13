/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * For commercial licensing inquiries, please contact: 3680010825@qq.com
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

// Package config loads YAML application configuration and provides the AppConfig aggregate.
package config

type AppConfig struct {
	// DSN is the MySQL (or other RDBMS) connection string.
	DSN      string `yaml:"dsn"`
	DBDriver string `yaml:"db_driver"`

	// Redis groups connection settings for the Redis cache.
	Redis RedisConfig `yaml:"redis"`

	// DNS configures custom nameservers for DNS lookups (feature extraction, DKIM, SPF).
	// When empty the system-configured DNS servers are used.
	DNS DNSConfig `yaml:"dns"`

	HeartbeatIntervalSec int    `yaml:"heartbeat_second"`
	LogFile              string `yaml:"log_file"`
	AutoMigrate          bool   `yaml:"auto_migrate"`

	// MailStorage: on-disk mail layout and per-mailbox SQLite (shared by LMTP and storage layer).
	MailStorage MailStorageConfig `yaml:"storage"`

	// Dovecot: auth protocol listener (see internal/protocol/dovecot).
	Dovecot DovecotConfig `yaml:"dovecot"`

	// LMTP receives mail from Postfix (or others) via LMTP into local storage.
	LMTP LMTPConfig `yaml:"lmtp"`

	// Milter: Postfix smtpd_milters endpoint (see internal/protocol/milter).
	Milter MilterConfig `yaml:"milter"`

	// SMTP: SMTP client configuration for sending emails.
	SMTP SMTPConfig `yaml:"smtp"`

	// Classifier runs the standalone classify-model gRPC server (predictor pool syncs enabled ClassifyModel rows from DB).
	Classifier ClassifierConfig `yaml:"classifier"`

	// Admin: management HTTP API (cmd/admin or launcher).
	Admin AdminConfig `yaml:"admin"`

	// Webmail: end-user HTTP API (cmd/webmail, launcher).
	Webmail WebmailConfig `yaml:"webmail"`

	// TrialMode limits write operations for public trial (domain/user CRUD, password change, send mail).
	TrialMode bool `yaml:"trial_mode"`

	// Cache controls per-aggregate cache TTLs.
	Cache CacheConfig `yaml:"cache"`

	// IMAP: end-user protocol server (cmd/imapd, launcher).
	IMAP IMAPConfig `yaml:"imap"`

	// Postfix: configuration management and agent settings.
	Postfix PostfixConfig `yaml:"postfix"`
}

// DNSConfig controls which nameservers are used by easydns throughout the application.
type DNSConfig struct {
	// Servers is a list of nameserver addresses (e.g. "114.114.114.114", "8.8.8.8:53").
	Servers []string `yaml:"servers"`
}

// RedisConfig groups Redis connection settings.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	// Required, if true, fails process startup when Redis is unreachable (fail-fast).
	// Default false: skip Redis init when unavailable (healthz reports unhealthy).
	Required bool `yaml:"required"`
}

// CacheConfig controls per-aggregate cache TTLs.
// Zero or negative TTL disables caching for that aggregate.
type CacheConfig struct {
	MailDomainTTL string `yaml:"mail_domain_ttl"` // e.g. "120s"
	MailUserTTL   string `yaml:"mail_user_ttl"`   // e.g. "60s"
	AdminUserTTL  string `yaml:"admin_user_ttl"`  // e.g. "300s"
}

type AdminConfig struct {
	Enable             bool             `yaml:"enable"`
	Listen             string           `yaml:"listen"`
	CORSAllowedOrigins []string         `yaml:"cors_allowed_origins"`
	Logs               ServiceLogConfig `yaml:"logs"`
	JWT                JWTConfig        `yaml:"jwt"`
	StaticPath         string           `yaml:"static_path"`
	// TLS: when enabled, serve HTTPS instead of plain HTTP.
	TLSEnable       bool   `yaml:"tls_enable"`
	CertFile        string `yaml:"cert_file"`
	KeyFile         string `yaml:"key_file"`
	CertificateFile string `yaml:"certificate_file"`
	CertificateKey  string `yaml:"certificate_key"`
}

// WebmailConfig mirrors AdminConfig field layout for consistent operations.
type WebmailConfig struct {
	Enable             bool             `yaml:"enable"`
	Listen             string           `yaml:"listen"`
	CORSAllowedOrigins []string         `yaml:"cors_allowed_origins"`
	Logs               ServiceLogConfig `yaml:"logs"`
	JWT                JWTConfig        `yaml:"jwt"`
	// ContactsSQLitePath is the address book SQLite file; empty uses {storage.local}/contact.sqlite.
	ContactsSQLitePath string `yaml:"contacts_sqlite_path"`
	StaticPath         string `yaml:"static_path"`
	// TLS: when enabled, serve HTTPS instead of plain HTTP.
	TLSEnable       bool   `yaml:"tls_enable"`
	CertFile        string `yaml:"cert_file"`
	KeyFile         string `yaml:"key_file"`
	CertificateFile string `yaml:"certificate_file"`
	CertificateKey  string `yaml:"certificate_key"`
}

// IMAPConfig controls the built-in IMAP listener (plain TCP in v1; use TLS/stunnel in production).
type IMAPConfig struct {
	Enable bool             `yaml:"enable"`
	Listen string           `yaml:"listen"`
	Family string           `yaml:"family"` // tcp | unix; default tcp
	Logs   ServiceLogConfig `yaml:"logs"`
	// Debug enables verbose IMAP protocol tracing (command/response dumps).
	Debug bool `yaml:"debug"`
	// TLS: when enabled, serve direct TLS (port 993). CertFile and KeyFile are PEM paths.
	TLSEnable bool   `yaml:"tls_enable"`
	CertFile  string `yaml:"cert_file"`
	KeyFile   string `yaml:"key_file"`
	// CertificateFile and CertificateKey are aliases for CertFile and KeyFile
	CertificateFile string `yaml:"certificate_file"`
	CertificateKey  string `yaml:"certificate_key"`
	// StartTLS: when enabled, advertise STARTTLS in capability and allow upgrade on port 143.
	StartTLS bool `yaml:"starttls"`
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

// ServiceLogConfig is optional per-module log file settings; falls back to top-level log_file when disabled or empty.
type ServiceLogConfig struct {
	Enable     bool   `yaml:"enable"`
	File       string `yaml:"file"`
	Level      string `yaml:"level"` // debug, info, warn, error
	Rotate     bool   `yaml:"rotate"`
	MaxSize    string `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
}

// MailStorageConfig selects storage backend; only "local" is implemented.
type MailStorageConfig struct {
	Driver string                  `yaml:"driver"` // local (default)
	SQLite SQLiteStorageConfig     `yaml:"sqlite"`
	Local  []LocalStoragePartition `yaml:"local"`
}

// LocalStoragePartition defines one storage partition root.
type LocalStoragePartition struct {
	Root      string `yaml:"root"`
	StorageID int    `yaml:"storage_id"`
}

// RootForStorage returns the root path for the given storage ID.
func (m MailStorageConfig) RootForStorage(storageID int) string {
	for _, p := range m.Local {
		if p.StorageID == storageID {
			return p.Root
		}
	}
	if len(m.Local) > 0 {
		return m.Local[0].Root
	}
	return "./storage"
}

// SQLiteStorageConfig tunes SQLite pool options for mail index databases.
type SQLiteStorageConfig struct {
	BusyTimeoutMs int  `yaml:"busy_timeout_ms"`
	MaxOpenConns  int  `yaml:"max_open_conns"`
	WAL           bool `yaml:"wal"`
}

// DovecotConfig is the Dovecot authentication protocol listener.
type DovecotConfig struct {
	Enable               bool              `yaml:"enable"`
	Listen               string            `yaml:"listen"`
	Family               string            `yaml:"family"` // tcp | unix; default tcp
	Logs                 ServiceLogConfig  `yaml:"logs"`
	Parameter            map[string]string `yaml:"parameter"`
	HeartbeatIntervalSec int               `yaml:"heartbeat_interval_sec"`
	HeartbeatTTLSec      int               `yaml:"heartbeat_ttl_sec"`
}

// LMTPConfig receives mail from Postfix (or others) via LMTP into local storage.
type LMTPConfig struct {
	Enable bool             `yaml:"enable"`
	Listen string           `yaml:"listen"`
	Family string           `yaml:"family"` // tcp | unix; default tcp
	Logs   ServiceLogConfig `yaml:"logs"`
}

// MilterConfig is the Postfix smtpd_milters socket served by the main process / launcher.
type MilterConfig struct {
	Enable bool             `yaml:"enable"`
	Listen string           `yaml:"listen"`
	Family string           `yaml:"family"` // tcp | unix; default tcp
	Logs   ServiceLogConfig `yaml:"logs"`
	// filter nests rule-engine settings and optional remote model-filter gRPC client.
	Filter MilterFilterConfig `yaml:"filter"`
}

// SMTPConfig configures the SMTP client for sending emails.
type SMTPConfig struct {
	Enable             bool             `yaml:"enable"`
	Server             string           `yaml:"server"`               // SMTP server address (e.g., smtp.example.com:587)
	Username           string           `yaml:"username"`             // SMTP AUTH username
	Password           string           `yaml:"password"`             // SMTP AUTH password
	Port               int              `yaml:"port"`                 // SMTP port (587 for STARTTLS, 465 for SSL/TLS)
	UseTLS             bool             `yaml:"use_tls"`              // Use SSL/TLS directly (port 465)
	UseSTARTTLS        bool             `yaml:"use_starttls"`         // Use STARTTLS (port 587)
	InsecureSkipVerify bool             `yaml:"insecure_skip_verify"` // Skip TLS certificate verification
	FromAddress        string           `yaml:"from_address"`         // Default from address
	TimeoutSec         int              `yaml:"timeout_sec"`          // Connection timeout in seconds (default 30)
	Logs               ServiceLogConfig `yaml:"logs"`
}

// PostfixConfig controls Postfix configuration management features.
type PostfixConfig struct {
	// EasyMailHost is the hostname/IP that Postfix should use to reach EasyMail services
	// (milter, LMTP, dovecot auth). If empty, auto-detected from listener configs.
	EasyMailHost string `yaml:"easymail_host"`
}
