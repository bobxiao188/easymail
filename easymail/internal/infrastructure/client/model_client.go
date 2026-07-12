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

package client

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"unicode/utf8"

	filtermodelv1 "easymail/internal/api/filtermodel/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	classifyModelMu       sync.Mutex
	classifyModelConn     *grpc.ClientConn
	classifyModelEndpoint string
)

// CloseClassifyModelClient closes the cached gRPC connection (e.g. tests).
func CloseClassifyModelClient() error {
	classifyModelMu.Lock()
	defer classifyModelMu.Unlock()
	if classifyModelConn == nil {
		return nil
	}
	err := classifyModelConn.Close()
	classifyModelConn = nil
	classifyModelEndpoint = ""
	return err
}

func classifyModelConnCached(endpoint string) (*grpc.ClientConn, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, ErrClassifyModelEndpoint
	}
	classifyModelMu.Lock()
	defer classifyModelMu.Unlock()
	if classifyModelConn != nil && classifyModelEndpoint == endpoint {
		return classifyModelConn, nil
	}
	if classifyModelConn != nil {
		slog.Info("classify_model grpc closing previous connection", "old_endpoint", classifyModelEndpoint, "new_endpoint", endpoint)
		_ = classifyModelConn.Close()
		classifyModelConn = nil
	}
	slog.Info("classify_model grpc dialing", "endpoint", endpoint)
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Warn("classify_model grpc dial_failed", "endpoint", endpoint, "err", err.Error())
		return nil, err
	}
	classifyModelConn = conn
	classifyModelEndpoint = endpoint
	return classifyModelConn, nil
}

// ErrClassifyModelEndpoint is returned when milter.filter.classify_model.endpoint is unset.
var ErrClassifyModelEndpoint = errors.New("classify model grpc endpoint is empty")

// InferClassifyModels calls the remote classify-model Infer RPC.
// When verbose is false, per-call tracing uses slog.Debug to reduce production noise.
func InferClassifyModels(ctx context.Context, endpoint string, req *filtermodelv1.InferRequest, verbose bool) (*filtermodelv1.InferResponse, error) {
	textRunes := 0
	if req != nil {
		textRunes = utf8.RuneCountInString(req.GetText())
	}
	conn, err := classifyModelConnCached(endpoint)
	if err != nil {
		return nil, err
	}
	cli := filtermodelv1.NewFilterModelServiceClient(conn)
	if verbose {
		slog.Info("classify_model grpc Invoking Infer",
			"endpoint", endpoint,
			"text_runes", textRunes,
			"lang_codes_count", countLangCodes(req),
		)
	} else {
		slog.Debug("classify_model grpc Invoking Infer",
			"endpoint", endpoint,
			"text_runes", textRunes,
			"lang_codes_count", countLangCodes(req),
		)
	}
	resp, err := cli.Infer(ctx, req)
	if err != nil {
		slog.Warn("classify_model grpc Infer RPC error", "endpoint", endpoint, "err", err.Error())
		return nil, err
	}
	n := 0
	if resp != nil {
		n = len(resp.GetPredictions())
	}
	if verbose {
		slog.Info("classify_model grpc Infer returned", "endpoint", endpoint, "prediction_count", n)
	} else {
		slog.Debug("classify_model grpc Infer returned", "endpoint", endpoint, "prediction_count", n)
	}
	return resp, nil
}

func countLangCodes(req *filtermodelv1.InferRequest) int {
	if req == nil {
		return 0
	}
	return len(req.GetLanguageCodes())
}
