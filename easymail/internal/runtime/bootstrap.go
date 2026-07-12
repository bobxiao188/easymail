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

package runtime

import (
	"fmt"
	"path/filepath"
	"time"

	"easymail/internal/domain/management"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/easydns"
	"easymail/internal/infrastructure/filter/tokenizer"
	"easymail/internal/infrastructure/migrate"
	"easymail/internal/infrastructure/persistence/mysql"
	"easymail/pkg/config"
	"easymail/pkg/database"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Runtime struct {
	ConfigPath  string
	Config      *config.AppConfig
	DB          *gorm.DB
	RedisClient *redis.Client

	// Repositories ??initialized in Start(), cached via Redis when available
	MailDomainRepo management.MailDomainRepository
	MailUserRepo   management.MailUserRepository
	AdminUserRepo  management.AdminUserRepository
}

func Start(configPath string) (*Runtime, error) {
	cfg, err := database.Initialize(configPath)
	if err != nil {
		return nil, err
	}

	if cfg.AutoMigrate {
		db := database.GetDB()
		if db == nil {
			return nil, fmt.Errorf("auto_migrate enabled but db is nil")
		}
		if err := migrate.AutoMigrate(db); err != nil {
			return nil, fmt.Errorf("auto migrate failed: %w", err)
		}
		if err := migrate.AdminBootstrap(db); err != nil {
			return nil, fmt.Errorf("admin bootstrap failed: %w", err)
		}
		if err := migrate.FilterBootstrap(db); err != nil {
			return nil, fmt.Errorf("filter bootstrap: %w", err)
		}
		if err := migrate.PostfixBootstrap(db); err != nil {
			return nil, fmt.Errorf("postfix bootstrap: %w", err)
		}
	}

	db := database.GetDB()
	redisClient := database.GetRedisClient()

	// Build persistence layer and cached repositories
	dbp := mysql.NewStaticDB(db)
	pdbp := mysql.NewPersistenceDBProvider(dbp)

	var cacheBackend cache.CacheBackend
	if redisClient != nil {
		cacheBackend = cache.NewRedisBackend(redisClient)
	} else {
		cacheBackend = cache.NewMemoryBackend()
	}
	repoTTL := 5 * time.Minute

	// Initialize DNS resolver from config.
	if len(cfg.DNS.Servers) > 0 {
		easydns.SetDefault(easydns.NewResolver(
			easydns.Config{Servers: cfg.DNS.Servers},
		))
	}

	// Initialize GSE tokenizer with dictionaries from config/dict/
	dictDir := filepath.Join(filepath.Dir(configPath), "dict")
	if err := tokenizer.InitGSE(dictDir); err != nil {
		return nil, fmt.Errorf("init gse tokenizer: %w", err)
	}

	return &Runtime{

		ConfigPath:  configPath,
		Config:      cfg,
		DB:          db,
		RedisClient: redisClient,

		MailDomainRepo: cache.NewCachedMailDomainRepository(
			mysql.NewMailDomainRepository(pdbp), cacheBackend, repoTTL),
		MailUserRepo: cache.NewCachedMailUserRepository(
			mysql.NewMailUserRepository(pdbp), cacheBackend, repoTTL),
		AdminUserRepo: cache.NewCachedAdminUserRepository(
			mysql.NewAdminUserRepository(pdbp), cacheBackend, repoTTL),
	}, nil
}
