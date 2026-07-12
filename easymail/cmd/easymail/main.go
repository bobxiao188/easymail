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

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"easymail/internal/runtime"
	"easymail/internal/runtime/launcher"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/logger/easylog"
)

func main() {
	configFlag := flag.String("config", "", "path to YAML config; empty searches default locations")
	flag.Parse()

	configPath := *configFlag
	if configPath == "" {
		configPath = resolveConfigPath()
	}

	rt, err := runtime.Start(configPath)
	if err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	cfg := rt.Config
	log.Printf("%s", appi18n.LogMessage(appi18n.KeyLogUsingConfig, map[string]interface{}{"Path": rt.ConfigPath}))

	logger, err := easylog.New(cfg.LogFile)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}

	runners, err := launcher.BuildRunners(rt, logger)
	if err != nil {
		log.Fatalf("launcher: %v", err)
	}

	// Create manager with heartbeat support
	mgr := launcher.BuildManager(rt, runners, logger)

	// Register all heartbeat-capable runners
	mgr.RegisterHeartbeatRunners()

	// Start heartbeat manager
	if mgr.HeartbeatMgr != nil {
		mgr.HeartbeatMgr.Start()
		defer mgr.HeartbeatMgr.Stop()
	}

	if err := mgr.StartAll(); err != nil {
		log.Fatalf("launcher: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}

func resolveConfigPath() string {
	candidates := []string{
		"easymail.yaml",
		filepath.FromSlash("config/easymail.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.FromSlash("config/easymail.yaml")
}
