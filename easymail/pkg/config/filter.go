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
	"strings"
	"time"
)

// ClamAVClientConfig is the ClamAV daemon connection configuration.
type ClamAVClientConfig struct {
	Enable          bool   `yaml:"enable"`
	Addr            string `yaml:"addr"`
	TimeoutMs       int    `yaml:"timeout_ms"`
	MaxScanSize     int    `yaml:"max_scan_size"`
	ScanAttachments bool   `yaml:"scan_attachments"`
	ScanBody        bool   `yaml:"scan_body"`
}

// MilterFilterConfig nests rule-engine options and the optional remote classify-model gRPC client (under milter).
type MilterFilterConfig struct {
	Enable        bool                            `yaml:"enable"`
	Rules         MilterFilterRules               `yaml:"rules"`
	ClassifyModel MilterFilterClassifyModelConfig `yaml:"classify_model"`
	ClamAV        ClamAVClientConfig              `yaml:"clamav"`
}

// MilterFilterRejectReply is the SMTP triplet sent with SMFIR_REPLYCODE when a rule action is reject (milter body stage).
type MilterFilterRejectReply struct {
	SMTPCode     string `yaml:"smtp_code"`
	EnhancedCode string `yaml:"enhanced_code"`
	Message      string `yaml:"message"`
}

// MilterFilterRules holds filter rule engine behavior.
type MilterFilterRules struct {
	DefaultAction          string                  `yaml:"default_action"`
	LogFeatureSnapshot     bool                    `yaml:"log_feature_snapshot"`
	LogConditionTrace      bool                    `yaml:"log_condition_trace"`
	SkipForComposeDelivery bool                    `yaml:"skip_for_compose_delivery"`
	StageTimeoutMs         int                     `yaml:"stage_timeout_ms"`
	RejectReply            MilterFilterRejectReply `yaml:"reject_reply"`
}

// MilterFilterClassifyModelConfig configures classify models at SMTP scan time.
// FastText runs in-process in the milter; DistilBERT and other algorithms use the classify-model gRPC worker when Endpoint is set.
type MilterFilterClassifyModelConfig struct {
	Enable             bool     `yaml:"enable"`
	Endpoint           string   `yaml:"endpoint"` // host:port; empty is OK for FastText-only (no gRPC hop)
	InputMaxTextLength int      `yaml:"input_max_text_length"`
	InputEmailFields   []string `yaml:"input_email_fields"`
	InferDeadlineMs    int      `yaml:"infer_deadline_ms"`
	VerboseInferLogs   bool     `yaml:"verbose_infer_logs"`
}

// ClassifierConfig is the standalone classifier gRPC server.
type ClassifierConfig struct {
	Enable              bool             `yaml:"enable"`
	Listen              string           `yaml:"listen"`
	Family              string           `yaml:"family"` // tcp | unix
	Logs                ServiceLogConfig `yaml:"logs"`
	MaxConcurrent       int              `yaml:"max_concurrent"`
	InferTimeoutMs      int              `yaml:"infer_timeout_ms"`
	ONNXRuntimeLib      string           `yaml:"onnx_runtime_lib"`
	FastTextExecutable  string           `yaml:"fasttext_executable"`
	ModelRoot           string           `yaml:"model_root"`
}

// FilterConfig is the flattened runtime view passed to the filter Engine and LMTP routing helpers.
type FilterConfig struct {
	Enable                 bool
	DefaultAction          string
	LogFeatureSnapshot     bool
	LogConditionTrace      bool
	SkipForComposeDelivery bool
	StageTimeoutMs         int

	ModelGRPCEnable  bool
	ModelEndpoint    string
	InputMaxLength   int
	ModelInputFields []string
	InferDeadlineMs  int
	VerboseInferLogs bool

	RejectSMTPCode     string
	RejectEnhancedCode string
	RejectMessage      string

	ClamAVEnable          bool
	ClamAVAddr            string
	ClamAVTimeout         time.Duration
	ClamAVMaxScanSize     int
	ClamAVScanAttachments bool
	ClamAVScanBody        bool
}

// FilterEngineConfig returns ToFilterConfig() of milter.filter for consumers that expected cfg.filter.
func (c *AppConfig) FilterEngineConfig() FilterConfig {
	if c == nil {
		return FilterConfig{}
	}
	return c.Milter.Filter.ToFilterConfig()
}

// ToFilterConfig merges nested milter.filter into the legacy FilterConfig shape.
func (s MilterFilterConfig) ToFilterConfig() FilterConfig {
	fields := append([]string(nil), s.ClassifyModel.InputEmailFields...)
	rr := s.Rules.RejectReply
	cv := s.ClamAV
	return FilterConfig{
		Enable:                 s.Enable,
		DefaultAction:          s.Rules.DefaultAction,
		LogFeatureSnapshot:     s.Rules.LogFeatureSnapshot,
		LogConditionTrace:      s.Rules.LogConditionTrace,
		SkipForComposeDelivery: s.Rules.SkipForComposeDelivery,
		StageTimeoutMs:         s.Rules.StageTimeoutMs,
		ModelGRPCEnable:        s.ClassifyModel.Enable,
		ModelEndpoint:          strings.TrimSpace(s.ClassifyModel.Endpoint),
		InputMaxLength:         s.ClassifyModel.InputMaxTextLength,
		ModelInputFields:       fields,
		InferDeadlineMs:        s.ClassifyModel.InferDeadlineMs,
		VerboseInferLogs:       s.ClassifyModel.VerboseInferLogs,
		RejectSMTPCode:         strings.TrimSpace(rr.SMTPCode),
		RejectEnhancedCode:     strings.TrimSpace(rr.EnhancedCode),
		RejectMessage:          strings.TrimSpace(rr.Message),
		ClamAVEnable:           cv.Enable,
		ClamAVAddr:             strings.TrimSpace(cv.Addr),
		ClamAVTimeout:          time.Duration(cv.TimeoutMs) * time.Millisecond,
		ClamAVMaxScanSize:      cv.MaxScanSize,
		ClamAVScanAttachments:  cv.ScanAttachments,
		ClamAVScanBody:         cv.ScanBody,
	}
}

// ClassifyModelClientInferDeadline is the milter-side timeout for the whole Body-stage classify step.
// Uses InferDeadlineMs when > 0, else Rules.StageTimeoutMs, else 30s.
func ClassifyModelClientInferDeadline(c *AppConfig) time.Duration {
	if c == nil {
		return 30 * time.Second
	}
	flat := c.FilterEngineConfig()
	ms := flat.InferDeadlineMs
	if ms <= 0 {
		ms = flat.StageTimeoutMs
	}
	if ms <= 0 {
		return 30 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

// InferTimeout returns the per-inference timeout for the classify-model worker.
func (c ClassifierConfig) InferTimeout() time.Duration {
	if c.InferTimeoutMs <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.InferTimeoutMs) * time.Millisecond
}
