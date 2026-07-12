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
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"easymail/pkg/config"
	"easymail/pkg/heartbeat"

	"github.com/redis/go-redis/v9"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.ReadAppConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	ctx := context.Background()
	prefix := heartbeat.ServiceStatusPrefix

	// 查找所有服务
	keys, err := rdb.Keys(ctx, prefix+"*").Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "query keys: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-12s %-25s %-12s %s\n", "Service", "Last Heartbeat", "Age", "Status")
	fmt.Println(strings.Repeat("-", 70))

	for _, key := range keys {
		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			fmt.Printf("%-12s %-25s %-12s %s\n", extractServiceName(key), "N/A", "N/A", heartbeat.StatusStopped)
			continue
		}

		t, err := time.Parse(time.RFC3339, val)
		if err != nil {
			fmt.Printf("%-12s %-25s %-12s %s\n", extractServiceName(key), val, "invalid", heartbeat.StatusStopped)
			continue
		}

		age := time.Since(t)
		status := heartbeat.StatusStopped
		if age < 30*time.Second {
			status = heartbeat.StatusRunning
		} else if age < 120*time.Second {
			status = heartbeat.StatusWarning
		}

		fmt.Printf("%-12s %-25s %-12s %s\n", extractServiceName(key), val, age.Round(time.Second), status)
	}
}

func extractServiceName(key string) string {
	// easymail:service:{name}:heartbeat
	name := key
	if idx := strings.LastIndex(name, ":heartbeat"); idx > 0 {
		name = name[:idx]
	}
	if idx := strings.LastIndex(name, ":"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}
