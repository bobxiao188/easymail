# =============================================================================
# EasyMail Build Makefile
# Produces a self-contained release/ directory for Linux deployment.
#
# Targets:
#   all       - build everything (default)
#   backend   - compile Go binary
#   frontend  - build admin UI (Vue)
#   dirs      - create required directories
#   config    - copy default config + TLS certs
#   scripts   - copy deploy scripts
#   clean     - remove release/
# =============================================================================

SHELL := /bin/bash

# ---------------------------------------------------------------------------
# Tool detection (allow override with GO=/path/to/go, NPM=/path/to/npm)
# ---------------------------------------------------------------------------
GO  ?= go
NPM ?= npm
CP  ?= cp

# ---------------------------------------------------------------------------
# Platform detection — binary extension and cross-compilation
# ---------------------------------------------------------------------------
# Override with:  make GOOS=linux GOARCH=amd64
#                  make GOOS=windows GOARCH=amd64
#                  make GOOS=darwin  GOARCH=arm64
GOOS   ?= $(strip $(shell $(GO) env GOOS 2>/dev/null || echo linux))
GOARCH ?= $(strip $(shell $(GO) env GOARCH 2>/dev/null || echo amd64))

# .exe on Windows, no extension on Unix
# findstring does pure Make substring matching — immune to shell \r issues
BIN_EXT := $(if $(findstring windows,$(GOOS)),.exe,)

# fasttext binary — filename determined by wildcard (e.g. fasttext, fasttext.exe)
FASTTEXT_GLOB := fasttext*

# ONNX Runtime library — filename varies by platform (onnxruntime.dll, libonnxruntime.so, etc.)
ONNXRUNTIME_GLOB := onnxruntime*

# Control script — easymail.sh on Unix, easymail.ps1 on Windows
CTLSCRIPT_GLOB := easymail$(if $(findstring windows,$(GOOS)),.ps1,.sh)

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
RELEASE_DIR   := release

BACKEND_SRC   := easymail
BACKEND_MAIN  := $(BACKEND_SRC)/cmd/easymail
BACKEND_BIN   := $(RELEASE_DIR)/bin/easymail$(BIN_EXT)

POSTFIX_AGENT_MAIN := $(BACKEND_SRC)/cmd/postfix-agent

FRONTEND_SRC  := frontend/admin
FRONTEND_DIST := $(RELEASE_DIR)/frontend/admin/dist

WEBMAIL_SRC   := frontend/webmail
WEBMAIL_DIST  := $(RELEASE_DIR)/frontend/webmail/dist

CONFIG_SRC    := $(BACKEND_SRC)/config/easymail.yaml
CONFIG_DST    := $(RELEASE_DIR)/config/easymail.yaml
CERT_SRC      := $(BACKEND_SRC)/config

SCRIPTS_SRC   := $(BACKEND_SRC)/scripts
SCRIPTS_DST   := $(RELEASE_DIR)/scripts

# ---------------------------------------------------------------------------
# Default target
# ---------------------------------------------------------------------------
.PHONY: all release
all release: dirs postfix-agent backend fasttext lib ctlscript frontend webmail config scripts
	@echo ""
	@echo "===================================================================="
	@echo "  EasyMail release built at $(RELEASE_DIR)/"
	@echo "===================================================================="
	@echo "  bin/easymail$(BIN_EXT)      - backend binary ($(GOOS)/$(GOARCH))"
	@echo "  bin/$(FASTTEXT_GLOB)         - fasttext binary ($(GOOS))"
	@echo "  $(CTLSCRIPT_GLOB)               - control script ($(GOOS))"
	@echo "  lib/$(ONNXRUNTIME_GLOB)     - ONNX Runtime library ($(GOOS))"
	@echo "  config/easymail.yaml    - default configuration"
	@echo "  scripts/                - deployment scripts (init.sql, etc.)"
	@echo "  frontend/admin/dist/    - admin UI (Vue SPA)"
	@echo "  frontend/webmail/dist/  - webmail UI (Vue SPA)"
	@echo "  logs/                   - runtime logs directory"
	@echo "  storage/                - mail storage directory"
	@echo "===================================================================="

# ---------------------------------------------------------------------------
# Directories
# ---------------------------------------------------------------------------
.PHONY: dirs
dirs:
	@mkdir -p $(RELEASE_DIR)/bin
	@mkdir -p $(RELEASE_DIR)/lib
	@mkdir -p $(RELEASE_DIR)/config
	@mkdir -p $(RELEASE_DIR)/logs
	@mkdir -p $(RELEASE_DIR)/storage
	@mkdir -p $(RELEASE_DIR)/scripts
	@mkdir -p $(RELEASE_DIR)/frontend/admin
	@mkdir -p $(RELEASE_DIR)/frontend/webmail
	@echo "[dirs]    release directories created"

# ---------------------------------------------------------------------------
# Backend — Go build (auto-detects platform)
# ---------------------------------------------------------------------------
.PHONY: backend
backend:
	@echo "[backend] building easymail for $(GOOS)/$(GOARCH) ..."
	cd $(BACKEND_MAIN) && GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		-o ../../../$(BACKEND_BIN) \
		-ldflags="-s -w" \
		.
	@echo "[backend] binary ready: $(BACKEND_BIN)"

# ---------------------------------------------------------------------------
# postfix-agent — standalone Postfix management agent (cross-compile all platforms)
# Destination: scripts/<os>/postfix-agent  (copied to release by `scripts` target)
# ---------------------------------------------------------------------------
.PHONY: postfix-agent
postfix-agent:
	@echo "[postfix-agent] building for linux/amd64 ..."
	@cd $(POSTFIX_AGENT_MAIN) && GOOS=linux GOARCH=amd64 $(GO) build -o ../../scripts/linux/postfix-agent .
	@echo "[postfix-agent] building for windows/amd64 ..."
	@cd $(POSTFIX_AGENT_MAIN) && GOOS=windows GOARCH=amd64 $(GO) build -o ../../scripts/windows/postfix-agent.exe .
	@echo "[postfix-agent] building for darwin/amd64 ..."
	@cd $(POSTFIX_AGENT_MAIN) && GOOS=darwin GOARCH=amd64 $(GO) build -o ../../scripts/darwin/postfix-agent .
	@chmod +x $(SCRIPTS_SRC)/linux/postfix-agent $(SCRIPTS_SRC)/darwin/postfix-agent 2>/dev/null || true
	@echo "[postfix-agent] binaries ready: scripts/{linux,windows,darwin}/postfix-agent*"

# ---------------------------------------------------------------------------
# fasttext — supervised training binary (platform-specific)
# Source: scripts/$(GOOS)/fasttext*  →  Destination: release/bin/ (preserves filename)
# ---------------------------------------------------------------------------
.PHONY: fasttext
fasttext:
	@echo "[fasttext] copying fasttext binary for $(GOOS) ..."
	@src=$$(ls "$(SCRIPTS_SRC)/$(GOOS)"/$(FASTTEXT_GLOB) 2>/dev/null | head -1); \
	if [ -n "$$src" ]; then \
		dst="$(RELEASE_DIR)/bin/$$(basename "$$src")"; \
		$(CP) "$$src" "$$dst"; \
		chmod +x "$$dst"; \
		echo "[fasttext] $$dst"; \
	else \
		echo "[fasttext] WARNING: $(SCRIPTS_SRC)/$(GOOS)/$(FASTTEXT_GLOB) not found, skipping"; \
	fi

# ---------------------------------------------------------------------------
# lib — ONNX Runtime per-platform shared library
# Source: scripts/$(GOOS)/onnxruntime*  →  Destination: release/lib/ (preserves filename)
# ---------------------------------------------------------------------------
.PHONY: lib
lib:
	@echo "[lib]      copying ONNX Runtime library for $(GOOS) ..."
	@src=$$(ls "$(SCRIPTS_SRC)/$(GOOS)"/$(ONNXRUNTIME_GLOB) 2>/dev/null | head -1); \
	if [ -n "$$src" ]; then \
		dst="$(RELEASE_DIR)/lib/$$(basename "$$src")"; \
		$(CP) "$$src" "$$dst"; \
		echo "[lib]      $$dst"; \
	else \
		echo "[lib]      WARNING: $(SCRIPTS_SRC)/$(GOOS)/$(ONNXRUNTIME_GLOB) not found, skipping"; \
	fi

# ---------------------------------------------------------------------------
# ctlscript — platform control script (easymail.sh / easymail.ps1)
# Source: scripts/$(GOOS)/easymail(.sh|.ps1)  →  Destination: release/bin/
# ---------------------------------------------------------------------------
.PHONY: ctlscript
ctlscript:
	@echo "[ctlscript] copying control script for $(GOOS) ..."
	@src=$$(ls "$(SCRIPTS_SRC)/$(GOOS)"/$(CTLSCRIPT_GLOB) 2>/dev/null | head -1); \
	if [ -n "$$src" ]; then \
		dst="$(RELEASE_DIR)/$$(basename "$$src")"; \
		sed 's/\r$$//' "$$src" > "$$dst"; \
		chmod +x "$$dst" 2>/dev/null || true; \
		echo "[ctlscript] $$dst"; \
	else \
		echo "[ctlscript] WARNING: $(SCRIPTS_SRC)/$(GOOS)/$(CTLSCRIPT_GLOB) not found, skipping"; \
	fi

# ---------------------------------------------------------------------------
# Frontend — admin Vue SPA
# ---------------------------------------------------------------------------
.PHONY: frontend
frontend:
	@echo "[frontend] installing dependencies ..."
	cd $(FRONTEND_SRC) && $(NPM) ci --no-audit --no-fund
	@echo "[frontend] building admin UI ..."
	cd $(FRONTEND_SRC) && $(NPM) run build
	@rm -rf $(FRONTEND_DIST)
	@$(CP) -r $(FRONTEND_SRC)/dist $(FRONTEND_DIST)
	@echo "[frontend] admin UI built: $(FRONTEND_DIST)"

# ---------------------------------------------------------------------------
# Webmail frontend — Vue SPA
# ---------------------------------------------------------------------------
.PHONY: webmail
webmail:
	@echo "[webmail] installing dependencies ..."
	cd $(WEBMAIL_SRC) && $(NPM) ci --no-audit --no-fund
	@echo "[webmail] building webmail UI ..."
	cd $(WEBMAIL_SRC) && $(NPM) run build
	@rm -rf $(WEBMAIL_DIST)
	@$(CP) -r $(WEBMAIL_SRC)/dist $(WEBMAIL_DIST)
	@echo "[webmail] webmail UI built: $(WEBMAIL_DIST)"

# ---------------------------------------------------------------------------
# Config — default YAML + TLS test certs
# ---------------------------------------------------------------------------
.PHONY: config
config:
	@echo "[config]  copying default configuration ..."
	@$(CP) $(CONFIG_SRC) $(CONFIG_DST)
	@echo "[config]  $(CONFIG_DST)"
	@$(CP) $(CERT_SRC)/test.crt $(RELEASE_DIR)/config/test.crt
	@$(CP) $(CERT_SRC)/test.key $(RELEASE_DIR)/config/test.key
	@echo "[config]  TLS certs copied to $(RELEASE_DIR)/config/"
	@cp -r $(BACKEND_SRC)/config/dict $(RELEASE_DIR)/config/dict
	@echo "[config]  dict copied to $(RELEASE_DIR)/config/dict"

# ---------------------------------------------------------------------------
# Scripts — deployment helpers
# ---------------------------------------------------------------------------
.PHONY: scripts
scripts:
	@echo "[scripts] copying deployment scripts ..."
	@$(CP) $(SCRIPTS_SRC)/easymail.service $(SCRIPTS_DST)/easymail.service
	@$(CP) $(SCRIPTS_SRC)/init.sql          $(SCRIPTS_DST)/init.sql
	# Copy platform-specific binary bundles
	@for dir in linux windows darwin; do \
		if [ -d "$(SCRIPTS_SRC)/$$dir" ]; then \
			$(CP) -r "$(SCRIPTS_SRC)/$$dir" "$(SCRIPTS_DST)/$$dir"; \
		fi; \
	done
	@echo "[scripts] all scripts copied to $(SCRIPTS_DST)/"

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------
.PHONY: clean
clean:
	@rm -rf $(RELEASE_DIR)
	@echo "[clean]   $(RELEASE_DIR)/ removed"