/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * License: AGPLv3
 */
package antivirus

import "context"

// VirusScanRequest wraps the data to be filterd by an antivirus engine.
type VirusScanRequest struct {
	Data     []byte
	FileName string
}

// VirusScanResult represents the outcome of a single antivirus scan.
type VirusScanResult struct {
	IsVirus   bool
	VirusName string
	ScanOK    bool
	Error     error
	RawReply  string
}

func (r *VirusScanResult) IsClean() bool {
	return r != nil && r.ScanOK && !r.IsVirus && r.Error == nil
}
func (r *VirusScanResult) IsFailed() bool {
	return r != nil && r.Error != nil
}

// AntivirusEngine is the port for pluggable antivirus backends.
type AntivirusEngine interface {
	Ping(ctx context.Context) error
	Scan(ctx context.Context, req *VirusScanRequest) (*VirusScanResult, error)
}
