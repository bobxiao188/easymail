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

package milter

import (
	"context"
	redisstats "easymail/internal/infrastructure/filter/stats/redis"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	service "easymail/internal/app/filter"
	"easymail/internal/domain/filter"
	featstats "easymail/internal/domain/filter"
	"easymail/internal/domain/filter/antivirus"
	"easymail/internal/infrastructure/filter/extractors"
	"easymail/internal/pkg/rfc2047"
	"easymail/internal/protocol/milter"

	"github.com/google/uuid"
)

func (h *MilterHandler) Connect(host string, family string, port uint16, addr net.IP, m *milter.Modifier) (milter.Response, error) {
	if h.fc == nil {
		h.fc = filter.NewMilterContext()
		h.fc.TraceID = uuid.New().String()
	}
	h.fc.Stage = filter.StageConnect
	h.infof("milter connect host=%q family=%s port=%d addr=%s", host, family, port, addr.String())
	h.fc.ConnectHost = host
	h.fc.ConnectFamily = family
	h.fc.ConnectPort = port
	h.fc.ConnectIP = addr

	t0 := time.Now()
	ctx := context.Background()
	resp, err := h.finishMilterStage(ctx, filter.StageConnect, h.stageTimeout(150*time.Millisecond))
	h.infof("milter_trace connect_total elapsed_ms=%d resp=%v err=%v", time.Since(t0).Milliseconds(), resp, err)
	return resp, err
}

func (h *MilterHandler) Helo(name string, m *milter.Modifier) (milter.Response, error) {
	h.infof("milter helo name=%q", name)
	if h.fc == nil {
		h.fc = filter.NewMilterContext()
		h.fc.TraceID = uuid.New().String()
	}
	h.fc.Stage = filter.StageHelo
	h.fc.HeloName = strings.TrimSpace(name)

	ctx := context.Background()
	return h.finishMilterStage(ctx, filter.StageHelo, h.stageTimeout(80*time.Millisecond))
}

func (h *MilterHandler) MailFrom(from string, m *milter.Modifier) (milter.Response, error) {
	fromTrim := strings.TrimSpace(from)
	h.infof("milter mail from=%q (new message)", fromTrim)

	if h.fc == nil {
		h.fc = filter.NewMilterContext()
		h.fc.TraceID = uuid.New().String()
		h.fc.Rcpts = []string{}
	}
	h.fc.Stage = filter.StageMailFrom
	h.fc.QueueID = ""
	h.fc.Sender = fromTrim
	h.fc.MailFrom = fromTrim

	ctx := context.Background()
	return h.finishMilterStage(ctx, filter.StageMailFrom, h.stageTimeout(150*time.Millisecond))
}

func (h *MilterHandler) RcptTo(rcptTo string, m *milter.Modifier) (milter.Response, error) {
	if h.fc == nil {
		h.fc = filter.NewMilterContext()
		h.fc.TraceID = uuid.New().String()
		h.fc.Rcpts = []string{}
	}
	h.fc.Stage = filter.StageRcptTo
	rcptTo = strings.ToLower(strings.TrimSpace(rcptTo))
	if rcptTo != "" {
		h.fc.Rcpts = append(h.fc.Rcpts, rcptTo)
		h.infof("milter rcpt to=%q (total_rcpt=%d)", rcptTo, len(h.fc.Rcpts))
	}
	h.fc.Set("rcpt_count", float64(len(h.fc.Rcpts)))
	ctx := context.Background()
	return h.finishMilterStage(ctx, filter.StageRcptTo, h.stageTimeout(120*time.Millisecond))
}

// Header tracks per-header metadata for feature extraction (Received chain length, auth results, etc.)
func (h *MilterHandler) Header(name string, value string, m *milter.Modifier) (milter.Response, error) {
	if h.fc == nil {
		h.fc = filter.NewMilterContext()
		h.fc.TraceID = uuid.New().String()
	}
	h.fc.Stage = filter.StageHeaders

	// Track header-level features for rule evaluation
	key := "header_" + strings.ToLower(strings.ReplaceAll(name, "-", "_"))
	switch strings.ToLower(name) {
	case "received":
		count := h.fc.Features[key]
		h.fc.Set(key, count+1)
	case "authentication-results", "dkim-signature", "spf-check", "dmarc":
		h.fc.Set(key, 1)
	case "x-spam-status", "x-spam-flag", "x-spam-level":
		h.fc.Set(key, 1)
	case "list-unsubscribe", "list-id", "list-post":
		h.fc.Set("header_list_id", 1)
		h.fc.Set("header_list_unsubscribe", 1)
	case "content-type":
		if strings.Contains(strings.ToLower(value), "multipart/") {
			h.fc.Set("header_content_type_multipart", 1)
		}
	case "message-id":
		if strings.TrimSpace(value) != "" {
			h.fc.Set("header_has_message_id", 1)
		}
	case "date":
		if strings.TrimSpace(value) != "" {
			h.fc.Set("header_has_date", 1)
		}
	default:
		if strings.HasPrefix(strings.ToLower(name), "x-") {
			count := h.fc.Features["header_custom_count"]
			h.fc.Set("header_custom_count", count+1)
		}
	}

	return milter.RespContinue, nil
}

// Headers is called when all message headers have been processed.
func (h *MilterHandler) Headers(hdr textproto.MIMEHeader, m *milter.Modifier) (milter.Response, error) {
	if h.fc == nil {
		h.fc = filter.NewMilterContext()
		h.fc.TraceID = uuid.New().String()
	}
	h.fc.Stage = filter.StageHeaders
	n := 0
	if hdr != nil {
		n = len(hdr)
	}
	h.infof("milter end_of_headers fields=%d", n)
	if h.fc != nil {
		h.fc.Headers = hdr
		h.fc.Set("header_fields", float64(n))
		if raw := strings.TrimSpace(hdr.Get("Subject")); raw != "" {
			h.fc.Subject = rfc2047.DecodeHeader(raw)
		}

		if raw := strings.TrimSpace(hdr.Get("From")); raw != "" {
			h.fc.SenderName = rfc2047.DecodeHeader(raw)
		}
	}

	h.stripInboundEasymailPolicyHeaders(m)

	ctx := context.Background()
	return h.finishMilterStage(ctx, filter.StageHeaders, h.stageTimeout(200*time.Millisecond))
}

// BodyChunk accumulates body chunks; called for each chunk up to 64KB.
func (h *MilterHandler) BodyChunk(chunk []byte, m *milter.Modifier) (milter.Response, error) {
	if len(chunk) > 0 {
		newLen := len(h.bodyAccum) + len(chunk)
		if newLen > maxMilterBodyAccum {
			h.bodyAccum = h.bodyAccum[:maxMilterBodyAccum]
			return milter.RespContinue, nil
		}
		h.bodyAccum = append(h.bodyAccum, chunk...)
	}
	return milter.RespContinue, nil
}

func (h *MilterHandler) Body(m *milter.Modifier) (milter.Response, error) {
	t0 := time.Now()
	if h.fc == nil {
		h.fc = filter.NewMilterContext()
		h.fc.TraceID = uuid.New().String()
	}
	h.fc.Stage = filter.StageBody

	// read body from milter modifier
	body := h.bodyAccum
	bodyLen := 0
	if len(body) > 0 {
		bodyLen = len(body)
		h.fc.BodyBytes = body
	}

	ctx := context.Background()
	// Apply stage timeout to the entire body processing (scanVirus + evalAndTrack)
	// so that a slow DB or AV scan cannot block the milter response indefinitely.
	bodyCtx, bodyCancel := context.WithTimeout(ctx, h.stageTimeout(30*time.Second))
	defer bodyCancel()

	// extract macros from milter session
	if h.fc.QueueID == "" {
		if q := strings.TrimSpace(m.Macros["i"]); q != "" {
			h.fc.QueueID = q
		}
	}

	// transfer email fields and body to readable
	_ = service.ApplyBody(h.fc)

	// extract features in stage of body
	h.fe.Extract(bodyCtx, filter.StageBody, h.fc, h.stageTimeout(1000*time.Millisecond))

	// infer classify models using config from handler struct
	inferCtx := ctx
	cfg := h.Config
	d := cfg.InferDeadlineMs
	if d <= 0 {
		d = cfg.StageTimeoutMs
	}
	if d > 0 {
		var inferCancel context.CancelFunc
		inferCtx, inferCancel = context.WithTimeout(ctx, time.Duration(d)*time.Millisecond)
		defer inferCancel()
	}
	clsOut := extractors.InferClassifyModels(inferCtx, h.fc)
	h.infof("milter classify_model done trace_id=%s queue_id=%q feature_rows=%d",
		h.fc.TraceID, h.fc.QueueID, len(clsOut))

	// Antivirus scan: scan attachments, merge results as features for rule evaluation.
	h.infof("antivirus check: av=%v", h.av != nil)
	if h.av != nil {
		h.scanVirus(bodyCtx)
	}

	// eval all features
	res, evalErr := h.evalAndTrack(bodyCtx, filter.StageBody)
	if evalErr != nil {
		h.warnf("milter evalAndTrack error: %v", evalErr)
	}
	if res == nil {
		res = &service.MatchResult{Action: "accept", RuleID: 0, FeatureSnap: "{}", TraceJSON: "{}"}
	}

	if err := m.AddHeader(service.HeaderFilterAction, res.Action); err != nil {
		h.warnf("milter AddHeader %s: %v", service.HeaderFilterAction, err)
	}
	if res.RuleID > 0 {
		if err := m.AddHeader(service.HeaderFilterRuleID, strconv.FormatInt(res.RuleID, 10)); err != nil {
			h.warnf("milter AddHeader %s: %v", service.HeaderFilterRuleID, err)
		}
	}
	if err := m.AddHeader(service.HeaderFilterTraceID, h.fc.TraceID); err != nil {
		h.warnf("milter AddHeader %s: %v", service.HeaderFilterTraceID, err)
	}

	engineOn := h.eng != nil && h.eng.DB != nil && h.eng.Config.Enable
	h.infof("milter body_end queue_id=%q mail_from=%q rcpts=%v body_bytes=%d header_fields=%d subject=%q engine_enabled=%v evaluate_ms=%d rule_id=%d action=%s trace_id=%s feature_snap_len=%d",
		h.fc.QueueID, strings.TrimSpace(h.fc.MailFrom), h.fc.Rcpts, bodyLen, len(h.fc.Headers), h.fc.Subject, engineOn, time.Since(t0).Milliseconds(), res.RuleID, res.Action, h.fc.TraceID, len(res.FeatureSnap))

	{
		o := featstats.NormalizeOutcome(res.Action)
		if h.fc != nil && h.fc.ConnectIP != nil {
			redisstats.RecordIPOutcome(ctx, h.fc.ConnectIP.String(), o)
		}
		if h.fc != nil {
			redisstats.RecordSenderOutcome(ctx, h.fc.MailFrom, o)
			for _, r := range h.fc.Rcpts {
				r = strings.ToLower(strings.TrimSpace(r))
				if i := strings.LastIndex(r, "@"); i >= 0 && i < len(r)-1 {
					redisstats.RecordSenderRcptDomainOutcome(ctx, h.fc.MailFrom, r[i+1:], o)
				}
			}
		}
	}

	if strings.ToLower(strings.TrimSpace(res.Action)) == "reject" {
		code, enh, msg := "550", "5.7.1", "Spam detected by EasyMail"
		if h.eng != nil {
			if s := strings.TrimSpace(h.eng.Config.RejectSMTPCode); s != "" {
				code = s
			}
			if s := strings.TrimSpace(h.eng.Config.RejectEnhancedCode); s != "" {
				enh = s
			}
			if s := strings.TrimSpace(h.eng.Config.RejectMessage); s != "" {
				msg = s
			}
		}
		if err := m.CustomReply(code, enh, msg); err != nil {
			h.warnf("milter CustomReply: %v", err)
		}
		return milter.RespReject, nil
	}
	h.asyncStageFilterLog(coalesceStageMatchResult(res), time.Since(t0))
	return milter.RespContinue, nil
}

// scanVirus runs the antivirus engine on email content and merges results as features.
func (h *MilterHandler) scanVirus(ctx context.Context) {
	if h.av == nil {
		h.infof("antivirus skip: av=nil")
		return
	}
	if h.fc == nil {
		h.infof("antivirus skip: fc=nil")
		return
	}
	fc := h.fc

	const antivirusHit = "antivirus_hit"
	var items []antivirus.VirusScanRequest
	cfg := h.eng.Config

	h.infof("antivirus config: enable=%v scan_body=%v scan_attachments=%v body_bytes=%d attachments=%d",
		cfg.ClamAVEnable, cfg.ClamAVScanBody, cfg.ClamAVScanAttachments,
		len(fc.BodyBytes), len(fc.Attachments))

	if cfg.ClamAVScanBody && len(fc.BodyBytes) > 0 {
		items = append(items, antivirus.VirusScanRequest{
			Data:     fc.BodyBytes,
			FileName: "email_body",
		})
	}
	if cfg.ClamAVScanAttachments {
		for _, att := range fc.Attachments {
			content := att.Content
			if len(content) == 0 {
				continue
			}
			items = append(items, antivirus.VirusScanRequest{
				Data:     content,
				FileName: att.FileName,
			})
		}
	}
	if len(items) == 0 {
		h.infof("antivirus skip: no items to scan")
		fc.Set(antivirusHit, 0)
		return
	}

	h.infof("antivirus scanning %d items", len(items))
	anyVirus := false
	anyError := false
	for _, req := range items {
		h.infof("antivirus scanning file=%q size=%d", req.FileName, len(req.Data))
		result, err := h.av.Scan(ctx, &req)
		if err != nil {
			h.warnf("antivirus scan error file=%q err=%v", req.FileName, err)
			anyError = true
			continue
		}
		if result.VirusScanResult.IsVirus {
			anyVirus = true
			h.infof("antivirus virus detected file=%q virus=%q", req.FileName, result.VirusScanResult.VirusName)
		} else {
			h.infof("antivirus clean file=%q", req.FileName)
		}
	}
	if !anyError {
		if anyVirus {
			fc.Set(antivirusHit, 1)
		} else {
			fc.Set(antivirusHit, 0)
		}
	} else {
		fc.Set(antivirusHit, -1)
	}
	h.infof("antivirus done hit=%v", anyVirus)
}
