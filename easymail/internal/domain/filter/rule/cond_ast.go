package rule

import (
	"encoding/json"
	"fmt"
	"strings"
)

const maxCondRecursionDepth = 64

// CollectConditionFeatureKeys returns feature keys referenced by feat/cmp nodes (for stage resolution).
func CollectConditionFeatureKeys(jsonStr string) ([]string, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return nil, nil
	}
	var root CondNode
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var walk func(*CondNode, int) error
	walk = func(n *CondNode, depth int) error {
		if depth > maxCondRecursionDepth {
			return fmt.Errorf("condition recursion depth exceeded %d", maxCondRecursionDepth)
		}
		if n == nil {
			return nil
		}
		switch strings.ToLower(strings.TrimSpace(n.Op)) {
		case "feat", "cmp":
			if k := strings.TrimSpace(n.Feature); k != "" {
				seen[k] = struct{}{}
			}
		case "and", "or":
			for _, c := range n.Children {
				if err := walk(c, depth+1); err != nil {
					return err
				}
			}
		case "not":
			if len(n.Children) == 1 {
				if err := walk(n.Children[0], depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(&root, 0); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out, nil
}

// CondNode matches the admin rule JSON AST format (subset).
type CondNode struct {
	Op       string      `json:"op"`
	Feature  string      `json:"feature"`
	Kind     string      `json:"kind"`
	Value    float64     `json:"value"`
	Children []*CondNode `json:"children"`
}

func evalCond(n *CondNode, feat map[string]float64, depth int) (bool, error) {
	if n == nil {
		return false, nil
	}
	if depth > maxCondRecursionDepth {
		return false, fmt.Errorf("condition recursion depth exceeded %d", maxCondRecursionDepth)
	}
	switch strings.ToLower(strings.TrimSpace(n.Op)) {
	case "and":
		for _, c := range n.Children {
			v, err := evalCond(c, feat, depth+1)
			if err != nil {
				return false, err
			}
			if !v {
				return false, nil
			}
		}
		return true, nil
	case "or":
		for _, c := range n.Children {
			v, err := evalCond(c, feat, depth+1)
			if err != nil {
				return false, err
			}
			if v {
				return true, nil
			}
		}
		return false, nil
	case "not":
		if len(n.Children) != 1 {
			return false, fmt.Errorf("not expects 1 child")
		}
		v, err := evalCond(n.Children[0], feat, depth+1)
		if err != nil {
			return false, err
		}
		return !v, nil
	case "feat":
		key := strings.TrimSpace(n.Feature)
		val, ok := feat[key]
		if !ok {
			return false, nil
		}
		return val != 0, nil
	case "cmp":
		key := strings.TrimSpace(n.Feature)
		val, ok := feat[key]
		if !ok {
			return false, nil
		}
		k := strings.ToLower(strings.TrimSpace(n.Kind))
		switch k {
		case "eq":
			return val == n.Value, nil
		case "ne":
			return val != n.Value, nil
		case "gt":
			return val > n.Value, nil
		case "ge":
			return val >= n.Value, nil
		case "lt":
			return val < n.Value, nil
		case "le":
			return val <= n.Value, nil
		default:
			return false, fmt.Errorf("unknown cmp kind %q", k)
		}
	default:
		return false, fmt.Errorf("unknown op %q", n.Op)
	}
}

// ValidateConditionJSON checks the JSON can be parsed.
func ValidateConditionJSON(jsonStr string) error {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return fmt.Errorf("empty condition")
	}
	var root CondNode
	return json.Unmarshal([]byte(jsonStr), &root)
}

// EvalConditionJSON parses and evaluates the condition against features.
// The trace map is populated with feature keys evaluated (true means feature was present).
func EvalConditionJSON(jsonStr string, feat map[string]float64, trace map[string]bool) (bool, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return false, nil
	}
	var root CondNode
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return false, err
	}
	return evalCondWithTrace(&root, feat, trace, 0)
}

func evalCondWithTrace(n *CondNode, feat map[string]float64, trace map[string]bool, depth int) (bool, error) {
	if n == nil {
		return false, nil
	}
	if depth > maxCondRecursionDepth {
		return false, fmt.Errorf("condition recursion depth exceeded %d", maxCondRecursionDepth)
	}
	switch strings.ToLower(strings.TrimSpace(n.Op)) {
	case "and":
		for _, c := range n.Children {
			v, err := evalCondWithTrace(c, feat, trace, depth+1)
			if err != nil {
				return false, err
			}
			if !v {
				return false, nil
			}
		}
		return true, nil
	case "or":
		for _, c := range n.Children {
			v, err := evalCondWithTrace(c, feat, trace, depth+1)
			if err != nil {
				return false, err
			}
			if v {
				return true, nil
			}
		}
		return false, nil
	case "not":
		if len(n.Children) != 1 {
			return false, fmt.Errorf("not expects 1 child")
		}
		v, err := evalCondWithTrace(n.Children[0], feat, trace, depth+1)
		if err != nil {
			return false, err
		}
		trace[n.Feature] = !v
		return !v, nil
	case "feat":
		key := strings.TrimSpace(n.Feature)
		val, ok := feat[key]
		trace[key] = ok
		if !ok {
			return false, nil
		}
		return val != 0, nil
	case "cmp":
		key := strings.TrimSpace(n.Feature)
		val, ok := feat[key]
		trace[key] = ok
		if !ok {
			return false, nil
		}
		k := strings.ToLower(strings.TrimSpace(n.Kind))
		switch k {
		case "eq":
			return val == n.Value, nil
		case "ne":
			return val != n.Value, nil
		case "gt":
			return val > n.Value, nil
		case "ge":
			return val >= n.Value, nil
		case "lt":
			return val < n.Value, nil
		case "le":
			return val <= n.Value, nil
		default:
			return false, fmt.Errorf("unknown cmp kind %q", k)
		}
	default:
		return false, fmt.Errorf("unknown op %q", n.Op)
	}
}
