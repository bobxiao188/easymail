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

package filter

import (
	"context"
	"fmt"
	"net"

	filtermodelv1 "easymail/internal/api/filtermodel/v1"
	modelcache "easymail/internal/infrastructure/filter/classifier/modelcache"
	"easymail/internal/infrastructure/filter/classifier/distilbert"
	"easymail/pkg/config"
	"easymail/pkg/database"
	"easymail/pkg/logger/easylog"

	"google.golang.org/grpc"
)

// Serve listens on ln until ctx is cancelled. A ModelPool syncs enabled ClassifyModel rows from MySQL on pool_refresh_ms and pre-opens predictors.
// log (service file logger) receives per-RPC diagnostics; may be nil.
func Serve(ctx context.Context, cfg config.ClassifierConfig, ln net.Listener, log *easylog.Logger) error {
	if database.GetDB() == nil {
		return fmt.Errorf("classifier: database not initialized (needed for classify model cache)")
	}
	distilbert.SetONNXRuntimeLib(cfg.ONNXRuntimeLib)

	mc := modelcache.New()
	defer mc.Invalidate()

	svc := NewGRPCService(log, mc, cfg.MaxConcurrent, cfg.InferTimeout())
	s := grpc.NewServer()
	filtermodelv1.RegisterFilterModelServiceServer(s, svc)
	if log != nil {
		log.Infof("[classify_model_service] gRPC classify Infer registered; infer_timeout=%s max_concurrent=%d",
			cfg.InferTimeout(), cfg.MaxConcurrent)
	}

	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	return s.Serve(ln)
}

