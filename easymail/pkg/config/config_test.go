package config

import (
	"os"
	"strings"
	"testing"
)

func TestExpandEnvVars_NoPlaceholder(t *testing.T) {
	s := "mysql://localhost:3306/db"
	got := expandEnvVars(s)
	if got != s {
		t.Errorf("expandEnvVars(%q) = %q, want unchanged", s, got)
	}
}

func TestExpandEnvVars_EnvSet(t *testing.T) {
	os.Setenv("EASYMAIL_TEST_DSN", "mysql://prod:3306/db")
	defer os.Unsetenv("EASYMAIL_TEST_DSN")

	got := expandEnvVars("${EASYMAIL_TEST_DSN:mysql://local:3306/db}")
	if got != "mysql://prod:3306/db" {
		t.Errorf("expandEnvVars() = %q, want env value", got)
	}
}

func TestExpandEnvVars_EnvNotSet_Fallback(t *testing.T) {
	os.Unsetenv("EASYMAIL_MISSING_VAR")
	got := expandEnvVars("${EASYMAIL_MISSING_VAR:default-value}")
	if got != "default-value" {
		t.Errorf("expandEnvVars() = %q, want default-value", got)
	}
}

func TestExpandEnvVars_MultiplePlaceholders(t *testing.T) {
	os.Setenv("EASYMAIL_USER", "alice")
	os.Setenv("EASYMAIL_PASS", "s3cret")
	defer os.Unsetenv("EASYMAIL_USER")
	defer os.Unsetenv("EASYMAIL_PASS")

	s := "${EASYMAIL_USER:guest}:${EASYMAIL_PASS:pass}@localhost/${EASYMAIL_DB:mail}"
	got := expandEnvVars(s)
	if got != "alice:s3cret@localhost/mail" {
		t.Errorf("expandEnvVars() = %q, want alice:s3cret@localhost/mail", got)
	}
}

func TestExpandEnvVars_NoColonDefault(t *testing.T) {
	// Should not match - requires colon separator
	os.Setenv("EASYMAIL_KEY", "env-val")
	defer os.Unsetenv("EASYMAIL_KEY")
	s := "${EASYMAIL_KEY}"
	got := expandEnvVars(s)
	if got != "${EASYMAIL_KEY}" {
		t.Errorf("expandEnvVars(%q) = %q, want unchanged (no colon)", s, got)
	}
}

func TestExpandEnvVars_EmptyDefault(t *testing.T) {
	os.Unsetenv("EASYMAIL_EMPTY")
	got := expandEnvVars("${EASYMAIL_EMPTY:}")
	if got != "" {
		t.Errorf("expandEnvVars() = %q, want empty", got)
	}
}

func TestExpandEnvVars_EmptyString(t *testing.T) {
	got := expandEnvVars("")
	if got != "" {
		t.Errorf("expandEnvVars() = %q, want empty", got)
	}
}

func TestApplyDefaults_Nil(t *testing.T) {
	applyDefaults(nil) // should not panic
}

func TestApplyDefaults_StorageDefaults(t *testing.T) {
	cfg := &AppConfig{}
	applyDefaults(cfg)
	if cfg.MailStorage.Driver != "local" {
		t.Errorf("Driver = %q, want local", cfg.MailStorage.Driver)
	}
	if len(cfg.MailStorage.Local) != 1 {
		t.Fatalf("Local partitions = %d, want 1", len(cfg.MailStorage.Local))
	}
	if cfg.MailStorage.Local[0].Root != "./storage" {
		t.Errorf("Root = %q, want ./storage", cfg.MailStorage.Local[0].Root)
	}
	if cfg.MailStorage.Local[0].StorageID != 0 {
		t.Errorf("StorageID = %d, want 0", cfg.MailStorage.Local[0].StorageID)
	}
}

func TestApplyDefaults_MilterDefaults(t *testing.T) {
	cfg := &AppConfig{}
	applyDefaults(cfg)
	if cfg.Milter.Filter.Rules.DefaultAction != "accept" {
		t.Errorf("DefaultAction = %q, want accept", cfg.Milter.Filter.Rules.DefaultAction)
	}
	if cfg.Milter.Filter.Rules.RejectReply.SMTPCode != "550" {
		t.Errorf("SMTPCode = %q, want 550", cfg.Milter.Filter.Rules.RejectReply.SMTPCode)
	}
	if !strings.Contains(cfg.Milter.Filter.Rules.RejectReply.Message, "Spam") {
		t.Errorf("RejectReply message should mention Spam, got %q", cfg.Milter.Filter.Rules.RejectReply.Message)
	}
}

func TestApplyDefaults_ClassifierDefaults(t *testing.T) {
	cfg := &AppConfig{Classifier: ClassifierConfig{Enable: true}}
	applyDefaults(cfg)
	if cfg.Classifier.MaxConcurrent != 4 {
		t.Errorf("MaxConcurrent = %d, want 4", cfg.Classifier.MaxConcurrent)
	}
	if cfg.Classifier.InferTimeoutMs != 30000 {
		t.Errorf("InferTimeoutMs = %d, want 30000", cfg.Classifier.InferTimeoutMs)
	}
	if cfg.Classifier.Listen != "127.0.0.1:50051" {
		t.Errorf("Listen = %q, want 127.0.0.1:50051", cfg.Classifier.Listen)
	}
}

func TestApplyDefaults_CacheDefaults(t *testing.T) {
	cfg := &AppConfig{}
	applyDefaults(cfg)
	if cfg.Cache.MailDomainTTL != "120s" {
		t.Errorf("MailDomainTTL = %q, want 120s", cfg.Cache.MailDomainTTL)
	}
	if cfg.Cache.MailUserTTL != "60s" {
		t.Errorf("MailUserTTL = %q, want 60s", cfg.Cache.MailUserTTL)
	}
	if cfg.Cache.AdminUserTTL != "300s" {
		t.Errorf("AdminUserTTL = %q, want 300s", cfg.Cache.AdminUserTTL)
	}
}

func TestValidateConfig_Nil(t *testing.T) {
	err := ValidateConfig(nil)
	if err == nil {
		t.Fatal("ValidateConfig(nil) should error")
	}
}

func TestValidateConfig_WeakAdminSecret(t *testing.T) {
	cfg := &AppConfig{Admin: AdminConfig{
		Enable: true,
		JWT:    JWTConfig{Secret: "short"},
	}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("ValidateConfig() with short secret should error")
	}
}

func TestValidateConfig_PlaceholderAdminSecret(t *testing.T) {
	tests := []string{
		"your_jwt_secret_key",
		"change-me-1234567890123456789012",
		"change_in_production_1234567890abc",
	}
	for _, s := range tests {
		t.Run(s[:15], func(t *testing.T) {
			cfg := &AppConfig{Admin: AdminConfig{
				Enable: true,
				JWT:    JWTConfig{Secret: s},
			}}
			err := ValidateConfig(cfg)
			if err == nil {
				t.Errorf("ValidateConfig() with placeholder %q should error", s)
			}
		})
	}
}

func TestValidateConfig_StrongAdminSecret(t *testing.T) {
	cfg := &AppConfig{Admin: AdminConfig{
		Enable: true,
		JWT:    JWTConfig{Secret: "this-is-a-strong-secret-key-32-chars!!"},
	}}
	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("ValidateConfig() error = %v, want nil", err)
	}
}

func TestValidateConfig_DisabledAdminSkipsCheck(t *testing.T) {
	cfg := &AppConfig{Admin: AdminConfig{Enable: false}}
	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("ValidateConfig() with admin disabled should not error: %v", err)
	}
}

func TestValidateConfig_WeakWebmailSecret(t *testing.T) {
	cfg := &AppConfig{Webmail: WebmailConfig{
		Enable: true,
		JWT:    JWTConfig{Secret: "weak"},
	}}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("ValidateConfig() with weak webmail secret should error")
	}
}

func TestMailStorageConfig_RootForStorage(t *testing.T) {
	m := MailStorageConfig{
		Local: []LocalStoragePartition{
			{Root: "/data/mail1", StorageID: 1},
		},
	}
	if got := m.RootForStorage(1); got != "/data/mail1" {
		t.Errorf("RootForStorage(1) = %q, want /data/mail1", got)
	}
}

func TestMailStorageConfig_RootForStorage_Missing(t *testing.T) {
	m := MailStorageConfig{
		Local: []LocalStoragePartition{
			{Root: "/data/mail1", StorageID: 1},
		},
	}
	if got := m.RootForStorage(99); got != "/data/mail1" {
		t.Errorf("RootForStorage(99) = %q, want fallback /data/mail1", got)
	}
}

func TestMailStorageConfig_RootForStorage_Empty(t *testing.T) {
	m := MailStorageConfig{}
	if got := m.RootForStorage(0); got != "./storage" {
		t.Errorf("RootForStorage(0) with empty partitions = %q, want ./storage", got)
	}
}
