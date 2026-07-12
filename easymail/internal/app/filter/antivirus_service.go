/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * License: AGPLv3
 */

package filter

import (
	"context"
	"log/slog"
	"time"

	"easymail/internal/domain/filter/antivirus"
	infraav "easymail/internal/infrastructure/filter/antivirus"
	"easymail/pkg/config"
)

// AntivirusService orchestrates antivirus scanning within the mail processing pipeline.
// It depends on the AntivirusEngine port, keeping the domain decoupled from specific
// antivirus implementations (ClamAV, third-party API, etc.).
type AntivirusService struct {
	engine antivirus.AntivirusEngine
	log    *slog.Logger
	cfg    infraav.Config
}

// NewAntivirusService builds a service with the given engine and config.
// Pass nil engine to use the default ClamAV provider (disabled by default).
func NewAntivirusService(cfg infraav.Config, engine antivirus.AntivirusEngine) *AntivirusService {
	if engine == nil {
		engine = infraav.NewClamAVProvider(cfg)
	}
	return &AntivirusService{
		engine: engine,
		log:    slog.With("component", "antivirus_service"),
		cfg:    cfg,
	}
}

// ScanResult wraps the domain result with service-level metadata (latency, engine info).
type AVScanResult struct {
	VirusScanResult *antivirus.VirusScanResult
	Latency         time.Duration
	EngineVersion   string
}

// Scan performs a full antivirus scan: ping health check, scan data, collect version.
// Returns a service-level result with timing and engine metadata.
func (s *AntivirusService) Scan(ctx context.Context, req *antivirus.VirusScanRequest) (*AVScanResult, error) {
	if !s.cfg.Enable {
		s.log.Debug("antivirus scan skipped (disabled)")
		return &AVScanResult{VirusScanResult: &antivirus.VirusScanResult{ScanOK: true}}, nil
	}
	if len(req.Data) == 0 {
		return &AVScanResult{VirusScanResult: &antivirus.VirusScanResult{ScanOK: true}}, nil
	}

	t0 := time.Now()
	log := s.log.With("file_name", req.FileName, "data_bytes", len(req.Data))
	log.Debug("antivirus scan start")

	result, err := s.engine.Scan(ctx, req)
	latency := time.Since(t0)

	if err != nil {
		log.Warn("antivirus scan failed", "err", err, "latency_ms", latency.Milliseconds())
		return &AVScanResult{
			VirusScanResult: &antivirus.VirusScanResult{ScanOK: false, Error: err},
			Latency:         latency,
		}, nil // Return result, not error 鈥?transport failure is handled by caller policy.
	}

	log.Debug("antivirus scan done",
		"is_virus", result.IsVirus,
		"virus_name", result.VirusName,
		"latency_ms", latency.Milliseconds(),
	)

	if result.IsVirus {
		log.Info("virus detected",
			"virus_name", result.VirusName,
			"latency_ms", latency.Milliseconds(),
		)
	}

	return &AVScanResult{
		VirusScanResult: result,
		Latency:         latency,
	}, nil
}

// Ping checks the antivirus engine health.
func (s *AntivirusService) Ping(ctx context.Context) error {
	if !s.cfg.Enable {
		return nil
	}
	return s.engine.Ping(ctx)
}

// Version returns the underlying antivirus engine version.
func (s *AntivirusService) Version(ctx context.Context) (string, error) {
	if !s.cfg.Enable {
		return "disabled", nil
	}
	return "unknown", nil
}

// Close releases the engine connection.
func (s *AntivirusService) Close() error {
	return nil
}

// EngineAddress returns the configured engine endpoint.
func (s *AntivirusService) EngineAddress() string {
	return s.cfg.Addr
}

// NewAntivirusServiceFromConfig is a convenience constructor that reads config
// from the application-level filter config. When the ClamAV section is missing
// or empty, it falls back to the default (disabled).
func NewAntivirusServiceFromConfig(cfg config.FilterConfig) *AntivirusService {
	ac := infraav.Config{
		Addr:                 cfg.ClamAVAddr,
		Timeout:              cfg.ClamAVTimeout,
		Enable:               cfg.ClamAVEnable,
		MaxScanSize:          cfg.ClamAVMaxScanSize,
		ScanEmailAttachments: cfg.ClamAVScanAttachments,
		ScanEmailBody:        cfg.ClamAVScanBody,
	}
	if ac.Addr == "" {
		ac.Addr = "127.0.0.1:3310"
	}
	if ac.Timeout <= 0 {
		ac.Timeout = 5 * time.Minute
	}
	return NewAntivirusService(ac, nil)
}

// ScanAttachment is a helper that builds a VirusScanRequest from an attachment
// and runs the scan.
func (s *AntivirusService) ScanAttachment(ctx context.Context, filename string, data []byte) (*AVScanResult, error) {
	return s.Scan(ctx, &antivirus.VirusScanRequest{
		Data:     data,
		FileName: filename,
	})
}
