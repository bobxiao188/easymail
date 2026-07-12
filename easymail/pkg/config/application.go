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

package config

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// expandEnvVars replaces ${ENV:default} patterns in value strings.
func expandEnvVars(s string) string {
	re := regexp.MustCompile(`\$\{(\w+):([^}]*)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		envKey := parts[1]
		defaultVal := parts[2]
		if v := os.Getenv(envKey); v != "" {
			return v
		}
		return defaultVal
	})
}

func ReadAppConfig(filename string) (*AppConfig, error) {
	appConfig := AppConfig{}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Println("Error reading file:", err)
		return nil, err
	}
	dataStr := expandEnvVars(string(data))
	err = yaml.Unmarshal([]byte(dataStr), &appConfig)
	if err != nil {
		return nil, err
	}

	applyDefaults(&appConfig)
	applyEnvOverrides(&appConfig)
	return &appConfig, nil
}

// applyDefaults fills unset fields with sensible defaults.
func applyDefaults(cfg *AppConfig) {
	if cfg == nil {
		return
	}

	// --- Milter filter defaults ---
	if strings.TrimSpace(cfg.Milter.Filter.Rules.DefaultAction) == "" {
		cfg.Milter.Filter.Rules.DefaultAction = "accept"
	}
	rr := &cfg.Milter.Filter.Rules.RejectReply
	if strings.TrimSpace(rr.SMTPCode) == "" {
		rr.SMTPCode = "550"
	}
	if strings.TrimSpace(rr.EnhancedCode) == "" {
		rr.EnhancedCode = "5.7.1"
	}
	if strings.TrimSpace(rr.Message) == "" {
		rr.Message = "Spam detected by EasyMail"
	}

	// --- Classifier defaults ---
	w := &cfg.Classifier
	if w.MaxConcurrent <= 0 {
		w.MaxConcurrent = 4
	}
	if w.InferTimeoutMs <= 0 {
		w.InferTimeoutMs = 30000
	}
	if w.Enable {
		if strings.TrimSpace(w.Listen) == "" {
			w.Listen = "127.0.0.1:50051"
		}
		if strings.TrimSpace(w.Family) == "" {
			w.Family = "tcp"
		}
	}
	if cfg.Milter.Filter.ClassifyModel.Enable && strings.TrimSpace(cfg.Milter.Filter.ClassifyModel.Endpoint) == "" && w.Enable {
		cfg.Milter.Filter.ClassifyModel.Endpoint = w.Listen
	}

	// --- ClamAV defaults ---
	cv := &cfg.Milter.Filter.ClamAV
	if cv.TimeoutMs <= 0 {
		cv.TimeoutMs = 300000
	}
	if strings.TrimSpace(cv.Addr) == "" {
		cv.Addr = "127.0.0.1:3310"
	}

	// --- Mail storage defaults ---
	ms := &cfg.MailStorage
	if ms.Driver == "" {
		ms.Driver = "local"
	}
	if len(ms.Local) == 0 {
		ms.Local = []LocalStoragePartition{
			{Root: "./storage", StorageID: 0},
		}
	}
	if ms.SQLite.BusyTimeoutMs == 0 {
		ms.SQLite.BusyTimeoutMs = 5000
	}
	if ms.SQLite.MaxOpenConns == 0 {
		ms.SQLite.MaxOpenConns = 1
	}

	// --- Admin defaults ---
	if strings.TrimSpace(cfg.Admin.Listen) == "" {
		cfg.Admin.Listen = "0.0.0.0:8000"
	}
	if strings.TrimSpace(cfg.Admin.JWT.Secret) == "" {
		cfg.Admin.JWT.Secret = "change-me-your_jwt_secret_key_change_in_production"
	}
	if cfg.Admin.JWT.ExpireHours <= 0 {
		cfg.Admin.JWT.ExpireHours = 72
	}
	if strings.TrimSpace(cfg.Admin.StaticPath) == "" {
		cfg.Admin.StaticPath = "static/admin"
	}

	// --- Webmail defaults ---
	if strings.TrimSpace(cfg.Webmail.Listen) == "" {
		cfg.Webmail.Listen = "0.0.0.0:8080"
	}
	if strings.TrimSpace(cfg.Webmail.JWT.Secret) == "" {
		cfg.Webmail.JWT.Secret = "change-me-your_jwt_secret_key_change_in_production"
	}
	if cfg.Webmail.JWT.ExpireHours <= 0 {
		cfg.Webmail.JWT.ExpireHours = 72
	}
	if strings.TrimSpace(cfg.Webmail.StaticPath) == "" {
		cfg.Webmail.StaticPath = "static/webmail"
	}

	// --- Cache defaults ---
	c := &cfg.Cache
	if c.MailDomainTTL == "" {
		c.MailDomainTTL = "120s"
	}
	if c.MailUserTTL == "" {
		c.MailUserTTL = "60s"
	}
	if c.AdminUserTTL == "" {
		c.AdminUserTTL = "300s"
	}
}

func applyEnvOverrides(cfg *AppConfig) {
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_MYSQL_DSN")); v != "" {
		cfg.DSN = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_DB_DRIVER")); v != "" {
		cfg.DBDriver = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_REDIS_ADDR")); v != "" {
		cfg.Redis.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_REDIS_PASSWORD")); v != "" {
		cfg.Redis.Password = v
	}

	if v := strings.TrimSpace(os.Getenv("EASYMAIL_DNS_SERVERS")); v != "" {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.DNS.Servers = append(cfg.DNS.Servers, p)
			}
		}
	}

	if v := strings.TrimSpace(os.Getenv("EASYMAIL_REDIS_REQUIRED")); v != "" {

		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Redis.Required = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_LOG_FILE")); v != "" {
		cfg.LogFile = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_AUTO_MIGRATE")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.AutoMigrate = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_ADMIN_ENABLE")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Admin.Enable = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_ADMIN_LISTEN")); v != "" {
		cfg.Admin.Listen = v
	}
	if strings.TrimSpace(cfg.Admin.Listen) == "" {
		cfg.Admin.Listen = "0.0.0.0:8000"
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_ADMIN_JWT_SECRET")); v != "" {
		cfg.Admin.JWT.Secret = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_ADMIN_JWT_EXPIRE_HOURS")); v != "" {
		if h, err := strconv.Atoi(v); err == nil {
			cfg.Admin.JWT.ExpireHours = h
		}
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_ADMIN_CORS_ALLOWED_ORIGINS")); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		cfg.Admin.CORSAllowedOrigins = out
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_ADMIN_STATIC_PATH")); v != "" {
		cfg.Admin.StaticPath = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_WEBMAIL_ENABLE")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Webmail.Enable = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_WEBMAIL_LISTEN")); v != "" {
		cfg.Webmail.Listen = v
	}
	if strings.TrimSpace(cfg.Webmail.Listen) == "" {
		cfg.Webmail.Listen = "0.0.0.0:8080"
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_WEBMAIL_JWT_SECRET")); v != "" {
		cfg.Webmail.JWT.Secret = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_WEBMAIL_JWT_EXPIRE_HOURS")); v != "" {
		if h, err := strconv.Atoi(v); err == nil {
			cfg.Webmail.JWT.ExpireHours = h
		}
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_WEBMAIL_CORS_ALLOWED_ORIGINS")); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		cfg.Webmail.CORSAllowedOrigins = out
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_WEBMAIL_STATIC_PATH")); v != "" {
		cfg.Webmail.StaticPath = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_IMAP_ENABLE")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.IMAP.Enable = b
		}
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_IMAP_LISTEN")); v != "" {
		cfg.IMAP.Listen = v
	}
	if v := strings.TrimSpace(os.Getenv("EASYMAIL_IMAP_DEBUG")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.IMAP.Debug = b
		}
	}
}

func ValidateConfig(cfg *AppConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Admin.Enable {
		secret := strings.TrimSpace(cfg.Admin.JWT.Secret)
		placeholder := secret == "" ||
			strings.Contains(secret, "your_jwt_secret_key") ||
			strings.HasPrefix(secret, "change-me") ||
			strings.Contains(secret, "change_in_production")
		if placeholder || len(secret) < 32 {
			return fmt.Errorf("admin.jwt.secret is not set or too weak (must be >= 32 chars and not a placeholder)")
		}
	}
	if cfg.Webmail.Enable {
		secret := strings.TrimSpace(cfg.Webmail.JWT.Secret)
		placeholder := secret == "" ||
			strings.Contains(secret, "your_jwt_secret_key") ||
			strings.HasPrefix(secret, "change-me") ||
			strings.Contains(secret, "change_in_production")
		if placeholder || len(secret) < 32 {
			return fmt.Errorf("webmail.jwt.secret is not set or too weak (must be >= 32 chars and not a placeholder)")
		}
	}
	return nil
}
