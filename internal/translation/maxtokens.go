package translation

import (
	"encoding/json"
	"reflect"
)

const defaultMaxTokens = 64000
const defaultMinTokens = 32000

// AdjustMaxTokens adjusts the max_tokens value based on request context.
func AdjustMaxTokens(body map[string]any) int {
	raw, ok := body["max_tokens"]
	value := toInt(raw)
	if !ok || raw == nil || value <= 0 {
		value = defaultMaxTokens
	}

	if hasTools(body) && value < defaultMinTokens {
		value = defaultMinTokens
	}

	if budget := getThinkingBudget(body); budget > 0 && value <= budget {
		value = budget + 1024
	}

	return value
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func hasTools(body map[string]any) bool {
	raw, ok := body["tools"]
	if !ok || raw == nil {
		return false
	}
	v := reflect.ValueOf(raw)
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return v.Len() > 0
	}
	return false
}

func getThinkingBudget(body map[string]any) int {
	raw, ok := body["thinking"]
	if !ok || raw == nil {
		return 0
	}
	thinking, ok := raw.(map[string]any)
	if !ok {
		return 0
	}
	budgetRaw, ok := thinking["budget_tokens"]
	if !ok || budgetRaw == nil {
		return 0
	}
	return toInt(budgetRaw)
}
