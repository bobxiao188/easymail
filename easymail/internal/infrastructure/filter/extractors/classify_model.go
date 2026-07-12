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

// Classify-model feature path: build input text from mail fields, run FastText in-process on the milter and
// DistilBERT (and other non-FastText) models via the classify-model gRPC service, then merge scores into features.

package extractors

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	filtermodelv1 "easymail/internal/api/filtermodel/v1"
	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/classifier"
	"easymail/internal/domain/filter/feature"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/client"
	grpc "easymail/internal/infrastructure/filter/classifier/grpc"
	modelcache "easymail/internal/infrastructure/filter/classifier/modelcache"

	"easymail/pkg/config"
	"easymail/pkg/database"
)

// --- Input field defaults & language hints ---

var defaultClassifyModelInputEmailFields = []string{
	string(classifier.EmailSubject),
	string(classifier.EmailPlainTextBody),
	string(classifier.EmailFromName),
}

func defaultClassifyModelInputFields() []string {
	out := make([]string, len(defaultClassifyModelInputEmailFields))
	copy(out, defaultClassifyModelInputEmailFields)
	return out
}

func buildClassifyModelTextFromNames(fc *filter.MilterContext, fieldNames []string) string {
	if fc == nil || len(fieldNames) == 0 {
		return ""
	}
	fields := make(classifier.EmailFields, 0, len(fieldNames))
	for _, n := range fieldNames {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		fields = append(fields, n)
	}
	return buildClassifyModelText(fc, fields)
}

func collectLanguageCodes(fc *filter.MilterContext) []string {
	if fc == nil || fc.Headers == nil {
		return nil
	}
	raw := strings.TrimSpace(fc.Headers.Get("Content-Language"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.Split(p, ";")[0])
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

// --- Text assembly from MilterContext ---

func buildClassifyModelText(fc *filter.MilterContext, fields classifier.EmailFields) string {
	if fc == nil || len(fields) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fields {
		switch classifier.EmailField(strings.TrimSpace(string(f))) {
		case classifier.EmailFromName:
			parts = append(parts, fc.SenderName)
		case classifier.EmailSubject:
			parts = append(parts, fc.Subject)
		case classifier.EmailHtmlBody:
			parts = append(parts, fc.HTMLBody)
		case classifier.EmailPlainTextBody:
			parts = append(parts, fc.TextBody)
		case classifier.EmailAttachmentNames:
			parts = append(parts, strings.Join(fc.AttachmentNames, "\n"))
		case classifier.EmailAttachBody:
			parts = append(parts, concatAttachmentBodies(fc, 32<<10))
		default:
			continue
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func concatAttachmentBodies(fc *filter.MilterContext, maxPerPart int) string {
	if fc == nil || len(fc.Attachments) == 0 || maxPerPart <= 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range fc.Attachments {
		ct := strings.ToLower(strings.TrimSpace(p.ContentType))
		if ct != "" && !strings.HasPrefix(ct, "text/") && ct != "message/rfc822" {
			continue
		}
		raw := p.Content
		if len(raw) > maxPerPart {
			raw = raw[:maxPerPart]
		}
		if len(raw) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.Write(raw)
	}
	return b.String()
}

// truncateHeadTailRunes keeps the beginning and end of s in UTF-8 runes; middle replaced with "\n...\n".
func truncateHeadTailRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 512
	}
	const sep = "\n...\n"
	rs := []rune(s)
	if len(rs) <= maxLen {
		return s
	}
	sepR := []rune(sep)
	avail := maxLen - len(sepR)
	if avail < 2 {
		return string(rs[:maxLen])
	}
	head := avail / 2
	tail := avail - head
	return string(rs[:head]) + sep + string(rs[len(rs)-tail:])
}

// --- gRPC infer + feature merge ---

func probForCatalogLabel(dist []classifier.LabelScore, catalogLabel string) float64 {
	want := strings.TrimSpace(catalogLabel)
	want = strings.TrimPrefix(want, "__label__")
	want = strings.TrimSpace(want)
	if want == "" {
		return 0
	}
	wantLower := strings.ToLower(want)
	for i := range dist {
		raw := strings.TrimSpace(dist[i].Label)
		raw = strings.TrimPrefix(raw, "__label__")
		raw = strings.TrimSpace(raw)
		if raw == want || strings.ToLower(raw) == wantLower {
			return dist[i].Probability
		}
	}
	return 0
}

// batchesFromPrediction merges scores into feature keys. When catalogLabels is non-empty
// (persisted ClassifyModel.classLabels), one feature per catalog label is emitted as baseKey_sanitized(label)
// with probability matched from the model distribution; missing labels get 0.
func batchesFromPrediction(baseKey string, p classifier.Prediction, catalogLabels []string) []filter.FeatureBatch {
	if baseKey == "" || len(p.Distribution) == 0 {
		return nil
	}
	if len(catalogLabels) > 0 {
		b := make(filter.FeatureBatch)
		for _, raw := range catalogLabels {
			lab := strings.TrimSpace(raw)
			if lab == "" {
				continue
			}
			sub := classifier.SanitizeFeatureKey(lab)
			if sub == "" {
				continue
			}
			fk := classifier.SanitizeFeatureKey(baseKey + "_" + sub)
			if fk == "" {
				continue
			}
			b[fk] = probForCatalogLabel(p.Distribution, lab)
		}
		if len(b) == 0 {
			return nil
		}
		return []filter.FeatureBatch{b}
	}

	lps := p.Distribution
	switch {
	case len(lps) > 2:
		b := make(filter.FeatureBatch)
		for i := range lps {
			lp := lps[i]
			sub := classifier.SanitizeFeatureKey(strings.TrimSpace(lp.Label))
			if sub == "" {
				continue
			}
			fk := classifier.SanitizeFeatureKey(baseKey + "_" + sub)
			if fk == "" {
				continue
			}
			b[fk] = lp.Probability
		}
		if len(b) == 0 {
			return nil
		}
		return []filter.FeatureBatch{b}
	case len(lps) == 2:
		return []filter.FeatureBatch{{baseKey: lps[1].Probability}}
	case len(lps) == 1:
		return []filter.FeatureBatch{{baseKey: lps[0].Probability}}
	default:
		return nil
	}
}

func classifyOutputRecord(baseKey string, p classifier.Prediction, batches []filter.FeatureBatch) filter.ModelOutput {
	out := filter.ModelOutput{
		ModelKey: baseKey,
		Name:     p.ModelName,
		Label:    strings.TrimSpace(p.TopLabel),
		Prob:     p.TopProbability,
	}
	lps := p.Distribution
	combined := make(map[string]float64)
	for _, b := range batches {
		for k, v := range b {
			combined[k] = v
		}
	}
	if len(combined) > 0 {
		out.MultiLabelScores = combined
	}
	switch {
	case len(combined) >= 2:
		// per-class features (catalog or native multi-head)
	case len(combined) == 1:
		for _, v := range combined {
			out.ProbClass1 = v
			break
		}
	case len(lps) == 2:
		out.ProbClass1 = lps[1].Probability
	case len(lps) == 1:
		out.ProbClass1 = lps[0].Probability
	}
	return out
}

func mergeOnePrediction(fc *filter.MilterContext, p classifier.Prediction, modelByID map[string]classifier.Model, log *slog.Logger) (feature.Result, bool) {
	featKey := classifier.SanitizeFeatureKey(strings.TrimSpace(p.ModelName))
	if featKey == "" {
		featKey = classifier.SanitizeFeatureKey(p.ModelID)
	}
	if featKey == "" {
		log.Info("classify_model skip_prediction", "reason", "empty_feature_key", "model_id", p.ModelID, "model_name", p.ModelName)
		return feature.Result{}, false
	}
	var catalog []string
	if mdef, ok := modelByID[strings.TrimSpace(p.ModelID)]; ok && len(mdef.ClassLabels) > 0 {
		catalog = []string(mdef.ClassLabels)
	}
	if msg := strings.TrimSpace(p.Err); msg != "" {
		batch := filter.FeatureBatch{featKey: 0}
		fc.AppendModelOutput(filter.ModelOutput{
			ModelKey: featKey,
			Name:     p.ModelName,
			Err:      msg,
		})
		fc.Merge(batch)
		log.Info("classify_model prediction_error", "model_id", p.ModelID, "model_name", p.ModelName, "err", msg)
		return feature.Result{Key: featKey, Features: batch, Err: fmt.Errorf("%s", msg)}, true
	}
	if len(p.Distribution) == 0 {
		msg := "missing label_probabilities"
		batch := filter.FeatureBatch{featKey: 0}
		fc.AppendModelOutput(filter.ModelOutput{
			ModelKey: featKey,
			Name:     p.ModelName,
			Err:      msg,
		})
		fc.Merge(batch)
		log.Info("classify_model prediction_error", "model_id", p.ModelID, "model_name", p.ModelName, "err", msg)
		return feature.Result{Key: featKey, Features: batch, Err: fmt.Errorf("%s", msg)}, true
	}
	batches := batchesFromPrediction(featKey, p, catalog)
	combined := filter.FeatureBatch{}
	for _, b := range batches {
		for k, v := range b {
			combined[k] = v
		}
		fc.Merge(b)
	}
	fc.AppendModelOutput(classifyOutputRecord(featKey, p, batches))
	return feature.Result{Key: featKey, Features: combined, Err: nil}, true
}

// runClassifyModels runs FastText in the milter process and other algorithms via the classify-model gRPC worker.
func runClassifyModels(ctx context.Context, fc *filter.MilterContext) []feature.Result {
	traceID := ""
	queueID := ""
	if fc != nil {
		traceID = fc.TraceID
		queueID = fc.QueueID
	}
	log := slog.With("component", "classify_model_client", "trace_id", traceID, "queue_id", queueID)

	if fc == nil {
		log.Info("classify_model skip", "reason", "nil_scan_context")
		return nil
	}
	app := database.GetAppConfig()
	if app == nil {
		log.Info("classify_model skip", "reason", "nil_app_config")
		return nil
	}
	if !app.Milter.Filter.ClassifyModel.Enable {
		log.Info("classify_model skip", "reason", "milter.filter.classify_model.enable_false",
			"hint", "set milter.filter.classify_model.enable: true in easymail.yaml")
		return nil
	}
	flat := app.FilterEngineConfig()
	verbose := flat.VerboseInferLogs
	ep := strings.TrimSpace(flat.ModelEndpoint)

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		d := config.ClassifyModelClientInferDeadline(app)
		if d > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		}
	}

	fields := flat.ModelInputFields
	if len(fields) == 0 {
		fields = defaultClassifyModelInputFields()
	}
	text := buildClassifyModelTextFromNames(fc, fields)
	maxLen := flat.InputMaxLength
	if maxLen <= 0 {
		maxLen = 8192
	}
	text = truncateHeadTailRunes(text, maxLen)
	textRunes := utf8.RuneCountInString(text)
	langs := collectLanguageCodes(fc)
	in := classifier.BuildPredictorInput(text, langs)

	_ = app.Classifier.InferTimeout()
	maxConc := app.Classifier.MaxConcurrent
	if maxConc <= 0 {
		maxConc = 4
	}

	var ftPreds []classifier.Prediction
	if mc := modelcache.MilterCache(); mc != nil {
		ftPreds = mc.PredictAll(ctx, in, func(m classifier.Model) bool {
			return m.Algorithm == classifier.AlgorithmFastText
		})
	}

	if verbose {
		log.Info("classify_model infer_request",
			"grpc_endpoint", ep,
			"fasttext_local_count", len(ftPreds),
			"text_runes", textRunes,
			"input_fields", fields,
			"lang_codes", langs,
		)
	} else {
		log.Debug("classify_model infer_request",
			"grpc_endpoint", ep,
			"fasttext_local_count", len(ftPreds),
			"text_runes", textRunes,
			"input_fields", fields,
			"lang_codes", langs,
		)
	}

	var combined []classifier.Prediction
	combined = append(combined, ftPreds...)

	if ep != "" {
		req := &filtermodelv1.InferRequest{
			Text:          text,
			LanguageCodes: langs,
		}
		// Ensure gRPC call has enough time even if FastText consumed most of ctx.
		grpcCtx := ctx
		if deadline, ok := ctx.Deadline(); ok {
			const minGRPCTimeout = 15 * time.Second
			if time.Until(deadline) < minGRPCTimeout {
				var cancel context.CancelFunc
				grpcCtx, cancel = context.WithTimeout(context.Background(), minGRPCTimeout)
				defer cancel()
			}
		}
		resp, err := client.InferClassifyModels(grpcCtx, ep, req, verbose)
		if err != nil {
			log.Warn("classify_model infer_rpc_failed", "endpoint", ep, "err", err.Error())
		} else if resp == nil {
			log.Warn("classify_model infer_nil_response", "endpoint", ep)
		} else {
			preds := resp.GetPredictions()
			if verbose {
				log.Info("classify_model infer_response", "endpoint", ep, "prediction_count", len(preds))
			} else {
				log.Debug("classify_model infer_response", "endpoint", ep, "prediction_count", len(preds))
			}
			for _, pProto := range preds {
				if pProto == nil {
					continue
				}
				combined = append(combined, grpc.PredictionFromProto(pProto))
			}
		}
	} else if len(ftPreds) == 0 {
		log.Debug("classify_model grpc skipped", "reason", "empty_grpc_endpoint",
			"hint", "FastText-only: ensure milter started the in-process FastText pool; DistilBERT needs classify_model_service + endpoint")
	}

	if len(combined) == 0 {
		log.Info("classify_model skip", "reason", "no_predictions",
			"hint", "enable models, configure gRPC for non-FastText, or wait for pool sync")
		return nil
	}

	modelByID := map[string]classifier.Model{}
	if all, err := cache.CachedClassifyModels(ctx, nil); err != nil {
		log.Warn("classify_model label_catalog_cache_failed", "err", err.Error())
	} else {
		for i := range all {
			m := all[i]
			if m.IsDeleted {
				continue
			}
			id := strconv.FormatUint(uint64(m.ID), 10)
			modelByID[id] = m
		}
	}

	var out []feature.Result
	for i := range combined {
		if r, ok := mergeOnePrediction(fc, combined[i], modelByID, log); ok {
			out = append(out, r)
		}
	}
	if verbose {
		log.Info("classify_model merged_feature_results", "count", len(out))
	} else {
		log.Debug("classify_model merged_feature_results", "count", len(out))
	}
	return out
}

// InferClassifyModels is the milter Body-stage entry: in-process FastText + optional gRPC for other algorithms.
func InferClassifyModels(ctx context.Context, fc *filter.MilterContext) []feature.Result {
	return runClassifyModels(ctx, fc)
}
