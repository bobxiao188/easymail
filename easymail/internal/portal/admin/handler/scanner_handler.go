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

package handler

import (
	appAdmin "easymail/internal/app/admin"
	filtersvc "easymail/internal/app/filter"
	"easymail/internal/domain/filter"
	"easymail/internal/domain/filter/rule"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"
	"easymail/pkg/types"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var recordNotFound = errors.New("record not found")

func normalizeFilterRuleAction(a string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "accept", "spam", "quarantine", "reject":
		return strings.ToLower(strings.TrimSpace(a)), true
	default:
		return "", false
	}
}

var customFeatureKeyRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{1,127}$`)

func normalizeCustomFeatureType(t string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "meta_regex", "composite":
		return strings.ToLower(strings.TrimSpace(t)), true
	default:
		return "", false
	}
}

func normalizeValueType(t string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "bool", "number":
		return strings.ToLower(strings.TrimSpace(t)), true
	default:
		return "", false
	}
}

type metaRegexSpec struct {
	Sources []string `json:"sources"`
	Pattern string   `json:"pattern"`
	Flags   string   `json:"flags"`
	Mode    string   `json:"mode"` // any | all
	Emit    string   `json:"emit"` // bool_hit | count
}

type compositeSpec struct {
	ConditionJSON string `json:"conditionJson"`
	Emit          string `json:"emit"` // bool
}

func validateCustomFeatureSpec(t string, specJSON string) error {
	specJSON = strings.TrimSpace(specJSON)
	if specJSON == "" {
		return errors.New("spec_json is empty")
	}
	switch t {
	case "meta_regex":
		var s metaRegexSpec
		if err := json.Unmarshal([]byte(specJSON), &s); err != nil {
			return err
		}
		if len(s.Sources) == 0 {
			return errors.New("meta_regex.sources is required")
		}
		if strings.TrimSpace(s.Pattern) == "" {
			return errors.New("meta_regex.pattern is required")
		}
		// compile regex with flags best-effort (Go doesn't support inline flag string directly)
		pat := s.Pattern
		flags := strings.ToLower(strings.TrimSpace(s.Flags))
		if flags != "" {
			if strings.Contains(flags, "i") {
				pat = "(?i)" + pat
			}
			if strings.Contains(flags, "m") {
				pat = "(?m)" + pat
			}
			if strings.Contains(flags, "s") {
				pat = "(?s)" + pat
			}
		}
		if _, err := regexp.Compile(pat); err != nil {
			return errors.New("invalid regex: " + err.Error())
		}
		mode := strings.ToLower(strings.TrimSpace(s.Mode))
		if mode != "" && mode != "any" && mode != "all" {
			return errors.New("meta_regex.mode must be any or all")
		}
		emit := strings.ToLower(strings.TrimSpace(s.Emit))
		if emit != "" && emit != "bool_hit" && emit != "count" {
			return errors.New("meta_regex.emit must be bool_hit or count")
		}
		return nil
	case "composite":
		var s compositeSpec
		if err := json.Unmarshal([]byte(specJSON), &s); err != nil {
			return err
		}
		if err := filtersvc.ValidateConditionJSON(s.ConditionJSON); err != nil {
			return errors.New("composite.condition_json invalid: " + err.Error())
		}
		return nil
	default:
		return errors.New("unknown custom feature type")
	}
}

// ListFilterFeaturesHandler GET /api/filter/features
func (h *Handler) ListFilterFeaturesHandler(c *gin.Context) {
	type featureRow struct {
		ID          int64             `json:"id"`
		FeatureKey  string            `json:"featureKey"`
		Label       string            `json:"label"`
		ValueType   string            `json:"valueType"`
		Description string            `json:"description"`
		Unit        string            `json:"unit"`
		CreatedAt   string            `json:"createdAt"`
		UpdatedAt   string            `json:"updatedAt"`
		ModelSource *rule.ModelSource `json:"modelSource,omitempty"`
	}

	// Merged list: built-in + enabled custom + enabled classify models (Body-stage numeric scores).
	builtin, err := h.scannerService.ListBuiltinFeatures(c.Request.Context())
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	custom, err := h.scannerService.ListCustomFeatures(c.Request.Context())
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}

	all := make([]featureRow, 0, len(builtin)+len(custom))
	for _, r := range builtin {
		all = append(all, featureRow{
			ID:          r.ID,
			FeatureKey:  r.FeatureKey,
			Label:       r.Label,
			ValueType:   r.ValueType,
			Description: r.Description,
			Unit:        r.Unit,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
			ModelSource: r.ModelSource,
		})
	}
	for _, r := range custom {
		all = append(all, featureRow{
			ID:          r.ID,
			FeatureKey:  r.FeatureKey,
			Label:       r.Label,
			ValueType:   r.ValueType,
			Description: r.Description,
			Unit:        r.Unit,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
		})
	}

	seenKeys := make(map[string]struct{}, len(all))
	for _, r := range all {
		seenKeys[strings.ToLower(strings.TrimSpace(r.FeatureKey))] = struct{}{}
	}
	enabled := 1
	cmRows, _, err := h.spamModelService.List(c.Request.Context(), "", "", &enabled, 1, 1000)
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	descCM := appi18n.Message(c, appi18n.KeyFilterClassifyModelFeatureDesc)
	for _, m := range cmRows {
		entries := rule.RuleFeatureEntries(m)
		for i, ent := range entries {
			if ent.Key == "" {
				continue
			}
			lk := strings.ToLower(ent.Key)
			if _, dup := seenKeys[lk]; dup {
				continue
			}
			seenKeys[lk] = struct{}{}
			// Build BuiltinFeature with ModelSource
			modelSource := &rule.ModelSource{
				ModelID:       uint(m.ID),
				ModelName:     m.Name,
				FeatureOrigin: ent.ModelSource.FeatureOrigin,
			}
			all = append(all, featureRow{
				ID:          -(int64(m.ID)*1000 + int64(i)),
				FeatureKey:  ent.Key,
				Label:       ent.DisplayLabel,
				ValueType:   "number",
				Description: descCM,
				Unit:        "",
				CreatedAt:   m.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
				ModelSource: modelSource,
			})
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].FeatureKey < all[j].FeatureKey })

	// Backward compatible:
	// - No page params: return full list.
	// - With page/pageSize: return paginated response.
	page := c.Query("page")
	pageSize := c.Query("pageSize")
	if pageSize == "" {
		pageSize = c.Query("page_size") // backward compatibility
	}
	if strings.TrimSpace(page) == "" && strings.TrimSpace(pageSize) == "" {
		response.Success(c, all)
		return
	}
	var q types.PaginationRequest
	if err := q.FromGinContext(c); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	total := int64(len(all))
	start := (q.Page - 1) * q.PageSize
	if start < 0 {
		start = 0
	}
	end := start + q.PageSize
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}
	pageRows := all[start:end]
	meta := types.NewPaginationMeta(q.Page, q.PageSize, total)
	response.SuccessWithPagination(c, pageRows, meta)
}

type customFeatureWriteRequest struct {
	FeatureKey  string `json:"featureKey" binding:"required"`
	Label       string `json:"label" binding:"required"`
	Type        string `json:"type" binding:"required"`      // meta_regex | composite
	ValueType   string `json:"valueType" binding:"required"` // bool | number
	Enabled     bool   `json:"enabled"`
	SpecJSON    string `json:"specJson" binding:"required"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
}

// ListFilterCustomFeaturesHandler GET /api/filter/custom-features
func (h *Handler) ListFilterCustomFeaturesHandler(c *gin.Context) {
	page := c.Query("page")
	pageSize := c.Query("pageSize")
	if pageSize == "" {
		pageSize = c.Query("page_size") // backward compatibility
	}
	if strings.TrimSpace(page) == "" && strings.TrimSpace(pageSize) == "" {
		rows, err := h.scannerService.ListCustomFeatures(c.Request.Context())
		if err != nil {
			response.InternalError(c, messageInternalError(c))
			return
		}
		response.Success(c, rows)
		return
	}
	var q types.PaginationRequest
	if err := q.FromGinContext(c); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	rows, err := h.scannerService.ListCustomFeatures(c.Request.Context())
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	meta := types.NewPaginationMeta(q.Page, q.PageSize, int64(len(rows)))
	response.SuccessWithPagination(c, rows, meta)
}

// GetFilterCustomFeatureHandler GET /api/filter/custom-features/:id
func (h *Handler) GetFilterCustomFeatureHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	row, err := h.scannerService.GetCustomFeature(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, recordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundFeature))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	response.Success(c, row)
}

// CreateFilterCustomFeatureHandler POST /api/filter/custom-features
func (h *Handler) CreateFilterCustomFeatureHandler(c *gin.Context) {
	var req customFeatureWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	key := strings.TrimSpace(req.FeatureKey)
	if !customFeatureKeyRe.MatchString(key) {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterFeatureKeyInvalid))
		return
	}
	reserved, err := h.scannerService.FeatureKeyReserved(c.Request.Context(), key)
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	if reserved {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterFeatureKeyClassifyModelReserved))
		return
	}
	tp, ok := normalizeCustomFeatureType(req.Type)
	if !ok {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterTypeInvalid))
		return
	}
	vt, ok := normalizeValueType(req.ValueType)
	if !ok {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterValueTypeInvalid))
		return
	}
	if err := validateCustomFeatureSpec(tp, req.SpecJSON); err != nil {
		response.Fail(c, appi18n.MessageWith(c, appi18n.KeyFilterCustomSpecInvalid, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}
	row := &rule.CustomFeature{
		FeatureKey:  key,
		Label:       strings.TrimSpace(req.Label),
		Type:        tp,
		ValueType:   vt,
		Enabled:     req.Enabled,
		SpecJSON:    strings.TrimSpace(req.SpecJSON),
		Description: strings.TrimSpace(req.Description),
		Unit:        strings.TrimSpace(req.Unit),
	}
	if _, err := h.scannerService.CreateCustomFeature(c.Request.Context(), row.FeatureKey, row.Label, row.Type, nil); err != nil { // row.Fields omitted, handled via SpecJSON
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, row)
}

// UpdateFilterCustomFeatureHandler PUT /api/filter/custom-features/:id
func (h *Handler) UpdateFilterCustomFeatureHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	var req customFeatureWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	existing, err := h.scannerService.GetCustomFeature(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, recordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundFeature))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	key := strings.TrimSpace(req.FeatureKey)
	if !customFeatureKeyRe.MatchString(key) {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterFeatureKeyInvalid))
		return
	}
	reserved, resErr := h.scannerService.FeatureKeyReserved(c.Request.Context(), key)
	if resErr != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	if reserved {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterFeatureKeyClassifyModelReserved))
		return
	}
	tp, ok := normalizeCustomFeatureType(req.Type)
	if !ok {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterTypeInvalid))
		return
	}
	vt, ok := normalizeValueType(req.ValueType)
	if !ok {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterValueTypeInvalid))
		return
	}
	if err := validateCustomFeatureSpec(tp, req.SpecJSON); err != nil {
		response.Fail(c, appi18n.MessageWith(c, appi18n.KeyFilterCustomSpecInvalid, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}
	existing.FeatureKey = key
	existing.Label = strings.TrimSpace(req.Label)
	existing.Type = tp
	existing.ValueType = vt
	existing.Enabled = req.Enabled
	existing.SpecJSON = strings.TrimSpace(req.SpecJSON)
	existing.Description = strings.TrimSpace(req.Description)
	existing.Unit = strings.TrimSpace(req.Unit)
	if err := h.scannerService.UpdateCustomFeature(c.Request.Context(), existing.ID, existing.FeatureKey, existing.Label, existing.Type, nil); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, existing)
}

// PatchFilterCustomFeatureHandler PATCH /api/filter/custom-features/:id
func (h *Handler) PatchFilterCustomFeatureHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	existing, err := h.scannerService.GetCustomFeature(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, recordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundFeature))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if v, ok := patch["enabled"]; ok {
		if b, ok := v.(bool); ok {
			existing.Enabled = b
		} else if s, ok := v.(string); ok {
			existing.Enabled = s == "1"
		}
	}
	if err := h.scannerService.UpdateCustomFeature(c.Request.Context(), existing.ID, existing.FeatureKey, existing.Label, existing.Type, nil); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, existing)
}

// DeleteFilterCustomFeatureHandler DELETE /api/filter/custom-features/:id
func (h *Handler) DeleteFilterCustomFeatureHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	if err := h.scannerService.DeleteCustomFeature(c.Request.Context(), id); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, nil)
}

// ListFilterRulesHandler GET /api/filter/rules
func (h *Handler) ListFilterRulesHandler(c *gin.Context) {
	// Backward compatible:
	// - No page params: return full list (old behavior).
	// - With page/pageSize: return paginated response.
	page := c.Query("page")
	pageSize := c.Query("pageSize")
	if pageSize == "" {
		pageSize = c.Query("page_size") // backward compatibility
	}
	if strings.TrimSpace(page) == "" && strings.TrimSpace(pageSize) == "" {
		_, rows, err := h.scannerService.ListRules(c.Request.Context(), "", 1, 1000)
		if err != nil {
			response.InternalError(c, messageInternalError(c))
			return
		}
		response.Success(c, rows)
		return
	}

	var q types.PaginationRequest
	if err := q.FromGinContext(c); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	rows, _, err := h.scannerService.ListRules(c.Request.Context(), q.Keyword, q.Page, q.PageSize)
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	meta := types.NewPaginationMeta(q.Page, q.PageSize, int64(len(rows)))
	response.SuccessWithPagination(c, rows, meta)
}

// GetFilterRuleHandler GET /api/filter/rules/:id
func (h *Handler) GetFilterRuleHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	r, err := h.scannerService.GetRule(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, recordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundRule))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	response.Success(c, r)
}

type filterRuleWriteRequest struct {
	Name          string `json:"name" binding:"required"`
	Enabled       bool   `json:"enabled"`
	Priority      int    `json:"priority"`
	Action        string `json:"action" binding:"required"`
	ConditionJSON string `json:"conditionJson" binding:"required"`
}

// CreateFilterRuleHandler POST /api/filter/rules
func (h *Handler) CreateFilterRuleHandler(c *gin.Context) {
	var req filterRuleWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if err := h.scannerService.ValidateConditionJSON(req.ConditionJSON); err != nil {
		response.Fail(c, appi18n.MessageWith(c, appi18n.KeyFilterConditionInvalid, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}
	act, ok := normalizeFilterRuleAction(req.Action)
	if !ok {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterActionInvalid))
		return
	}
	row := &rule.Rule{
		Name:          strings.TrimSpace(req.Name),
		Enabled:       req.Enabled,
		Priority:      req.Priority,
		Action:        filter.NormalizeOutcome(act),
		ConditionJSON: strings.TrimSpace(req.ConditionJSON),
	}
	if err := h.scannerService.SetRuleStageFromCondition(c.Request.Context(), row); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	if err := h.scannerService.CreateRule(c.Request.Context(), row); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, row)
}

// UpdateFilterRuleHandler PUT /api/filter/rules/:id
func (h *Handler) UpdateFilterRuleHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	var req filterRuleWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if err := h.scannerService.ValidateConditionJSON(req.ConditionJSON); err != nil {
		response.Fail(c, appi18n.MessageWith(c, appi18n.KeyFilterConditionInvalid, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
		return
	}
	act, ok := normalizeFilterRuleAction(req.Action)
	if !ok {
		response.Fail(c, appi18n.Message(c, appi18n.KeyFilterActionInvalid))
		return
	}
	existing, err := h.scannerService.GetRule(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, recordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundRule))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	existing.Name = strings.TrimSpace(req.Name)
	existing.Enabled = req.Enabled
	existing.Priority = req.Priority
	existing.Action = filter.NormalizeOutcome(act)
	existing.ConditionJSON = strings.TrimSpace(req.ConditionJSON)
	if err := h.scannerService.SetRuleStageFromCondition(c.Request.Context(), existing); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	if err := h.scannerService.UpdateRule(c.Request.Context(), existing); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, existing)
}

// PatchFilterRuleHandler PATCH /api/filter/rules/:id
func (h *Handler) PatchFilterRuleHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	existing, err := h.scannerService.GetRule(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, recordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundRule))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if v, ok := patch["enabled"]; ok {
		if b, ok := v.(bool); ok {
			existing.Enabled = b
		}
	}
	if v, ok := patch["priority"]; ok {
		if p, ok := v.(float64); ok {
			existing.Priority = int(p)
		}
	}
	if v, ok := patch["action"]; ok {
		if a, ok := v.(string); ok {
			act, valid := normalizeFilterRuleAction(a)
			if valid {
				existing.Action = filter.NormalizeOutcome(act)
			}
		}
	}
	if v, ok := patch["name"]; ok {
		if n, ok := v.(string); ok {
			existing.Name = strings.TrimSpace(n)
		}
	}
	if v, ok := patch["condition_json"]; ok {
		if cj, ok := v.(string); ok {
			if err := h.scannerService.ValidateConditionJSON(cj); err != nil {
				response.Fail(c, appi18n.MessageWith(c, appi18n.KeyFilterConditionInvalid, map[string]interface{}{"Err": appi18n.PublicErrDetail(err)}))
				return
			}
			existing.ConditionJSON = strings.TrimSpace(cj)
		}
	}
	// conditionJson is an alias for condition_json (camelCase in JSON from JS client)
	if _, has := patch["conditionJson"]; has && patch["condition_json"] == nil {
		patch["condition_json"] = patch["conditionJson"]
	}
	if err := h.scannerService.SetRuleStageFromCondition(c.Request.Context(), existing); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	if err := h.scannerService.UpdateRule(c.Request.Context(), existing); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, existing)
}

// DeleteFilterRuleHandler DELETE /api/filter/rules/:id
func (h *Handler) DeleteFilterRuleHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	if err := h.scannerService.DeleteRule(c.Request.Context(), id); err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	h.scannerService.InvalidateFilterRulesCache()
	response.Success(c, nil)
}

// ListFilterFilterLogsHandler GET /api/filter/delivery-logs
// Query: page, pageSize, ip, sender, rcpt, created_from, created_to (RFC3339 or YYYY-MM-DD UTC).
func (h *Handler) ListFilterFilterLogsHandler(c *gin.Context) {
	var q struct {
		Page        int    `form:"page"`
		PageSize    int    `form:"pageSize"`
		IP          string `form:"ip"`
		Sender      string `form:"sender"`
		Rcpt        string `form:"rcpt"`
		CreatedFrom string `form:"created_from"`
		CreatedTo   string `form:"created_to"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	filter := &appAdmin.FilterLogFilter{
		IP:            q.IP,
		Sender:        q.Sender,
		Recipient:     q.Rcpt,
		CreatedAfter:  filterLogTimePtr(q.CreatedFrom),
		CreatedBefore: filterLogTimePtr(q.CreatedTo),
	}
	rows, _, err := h.scannerService.ListFilterLogs(c.Request.Context(), filter, q.Page, q.PageSize)
	if err != nil {
		response.InternalError(c, messageInternalError(c))
		return
	}
	meta := types.NewPaginationMeta(q.Page, q.PageSize, int64(len(rows)))
	response.SuccessWithPagination(c, rows, meta)
}

// GetFilterFilterLogHandler GET /api/filter/delivery-logs/:id
func (h *Handler) GetFilterFilterLogHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrInvalidID))
		return
	}
	row, err := h.scannerService.GetFilterLog(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, recordNotFound) {
			response.NotFound(c, appi18n.Message(c, appi18n.KeyErrNotFoundRecord))
			return
		}
		response.InternalError(c, messageInternalError(c))
		return
	}
	response.Success(c, row)
}

func filterLogTimePtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
