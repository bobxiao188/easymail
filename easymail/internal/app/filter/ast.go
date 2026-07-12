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
	"encoding/json"
	"fmt"
	"strings"
)

// CondNode is a JSON AST node for rule conditions.
type CondNode struct {
	Op       string      `json:"op"`
	Feature  string      `json:"feature"`
	Kind     string      `json:"kind"`
	Value    float64     `json:"value"`
	Children []*CondNode `json:"children"`
}

// EvalResult holds one evaluation outcome and optional trace (JSON-pointer-like keys or short labels).
type EvalResult struct {
	OK    bool
	Trace map[string]bool
}

func evalCond(n *CondNode, feat map[string]float64, path string, trace map[string]bool) (bool, error) {
	if n == nil {
		return false, nil
	}
	if trace == nil {
		trace = make(map[string]bool)
	}
	switch strings.ToLower(strings.TrimSpace(n.Op)) {
	case "and":
		for i, c := range n.Children {
			p := fmt.Sprintf("%s/and[%d]", path, i)
			v, err := evalCond(c, feat, p, trace)
			if err != nil {
				return false, err
			}
			trace[p] = v
			if !v {
				return false, nil
			}
		}
		return true, nil
	case "or":
		if len(n.Children) == 0 {
			return false, nil
		}
		for i, c := range n.Children {
			p := fmt.Sprintf("%s/or[%d]", path, i)
			v, err := evalCond(c, feat, p, trace)
			if err != nil {
				return false, err
			}
			trace[p] = v
			if v {
				return true, nil
			}
		}
		return false, nil
	case "not":
		if len(n.Children) != 1 {
			return false, fmt.Errorf("not expects 1 child")
		}
		p := path + "/not"
		v, err := evalCond(n.Children[0], feat, p, trace)
		if err != nil {
			return false, err
		}
		trace[p] = !v
		return !v, nil
	case "feat":
		key := strings.TrimSpace(n.Feature)
		val, ok := feat[key]
		if !ok {
			trace[path+"/feat"] = false
			return false, nil
		}
		k := strings.ToLower(strings.TrimSpace(n.Kind))
		var okb bool
		if k == "false" {
			okb = val == 0
		} else {
			okb = val != 0
		}
		trace[path+"/feat"] = okb
		return okb, nil
	case "cmp":
		key := strings.TrimSpace(n.Feature)
		val, ok := feat[key]
		if !ok {
			trace[path+"/cmp"] = false
			return false, nil
		}
		k := strings.ToLower(strings.TrimSpace(n.Kind))
		var res bool
		switch k {
		case "eq":
			res = val == n.Value
		case "ne":
			res = val != n.Value
		case "gt":
			res = val > n.Value
		case "ge":
			res = val >= n.Value
		case "lt":
			res = val < n.Value
		case "le":
			res = val <= n.Value
		default:
			return false, fmt.Errorf("unknown cmp kind %q", k)
		}
		trace[path+"/cmp"] = res
		return res, nil
	default:
		return false, fmt.Errorf("unknown op %q", n.Op)
	}
}

// ValidateConditionJSON checks that condition JSON unmarshals (admin rule save).
func ValidateConditionJSON(jsonStr string) error {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return fmt.Errorf("empty condition")
	}
	var root CondNode
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return err
	}
	return nil
}

// EvalConditionJSON unmarshals and evaluates; if trace is non-nil, sub-expression truth values are recorded.
func EvalConditionJSON(jsonStr string, feat map[string]float64, trace map[string]bool) (bool, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return false, nil
	}
	var root CondNode
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return false, err
	}
	return evalCond(&root, feat, "$", trace)
}

// TraceToJSON serializes the trace map to a JSON string.
func TraceToJSON(trace map[string]bool) string {
	if len(trace) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(trace)
	return string(b)
}
