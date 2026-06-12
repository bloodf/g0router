package translation

import "encoding/json"

// BufferTokens is the constant buffer added to input/prompt token counts to
// prevent context-length errors after translation. Port of
// open-sse/utils/usageTracking.js:BUFFER_TOKENS.
const BufferTokens = 2000

// NormalizeUsage returns a copy of the usage map containing only numerically
// valid fields and the original details objects. It mirrors
// open-sse/utils/usageTracking.js:115-150: non-finite numbers are dropped, the
// seven core token fields are coerced to int, prompt_tokens_details /
// completion_tokens_details are preserved when present, and an empty result
// returns nil.
func NormalizeUsage(usage map[string]any) map[string]any {
	if usage == nil {
		return nil
	}
	if _, isArr := usage["_"]; isArr { // cheap array check via type guard below
		// unreachable: arrays can't be top-level keys; defensive.
	}
	// Arrays are not handled in the JS reference either; reject them.
	if isArrayish(usage) {
		return nil
	}

	normalized := map[string]any{}
	assignNumber := func(key string, value any) {
		if value == nil {
			return
		}
		switch v := value.(type) {
		case float64:
			if v == v && v != v-1 { // not NaN
				normalized[key] = int(v)
			}
		case float32:
			if v == v && v != v-1 {
				normalized[key] = int(v)
			}
		case int:
			normalized[key] = v
		case int32:
			normalized[key] = int(v)
		case int64:
			normalized[key] = int(v)
		case uint:
			normalized[key] = int(v)
		case uint32:
			normalized[key] = int(v)
		case uint64:
			normalized[key] = int(v)
		case string:
			// Port of Number(value) coercion: only finite values survive.
			f, ok := parseFiniteNumber(v)
			if !ok {
				return
			}
			normalized[key] = int(f)
		}
	}

	assignNumber("prompt_tokens", usage["prompt_tokens"])
	assignNumber("completion_tokens", usage["completion_tokens"])
	assignNumber("total_tokens", usage["total_tokens"])
	assignNumber("cache_read_input_tokens", usage["cache_read_input_tokens"])
	assignNumber("cache_creation_input_tokens", usage["cache_creation_input_tokens"])
	assignNumber("cached_tokens", usage["cached_tokens"])
	assignNumber("reasoning_tokens", usage["reasoning_tokens"])

	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		normalized["prompt_tokens_details"] = details
	}
	if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
		normalized["completion_tokens_details"] = details
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// HasValidUsage reports whether the usage object has at least one known token
// field with a value greater than zero. Mirrors
// open-sse/utils/usageTracking.js:150-170.
func HasValidUsage(usage map[string]any) bool {
	if usage == nil {
		return false
	}
	fields := []string{
		"prompt_tokens", "completion_tokens", "total_tokens",
		"input_tokens", "output_tokens",
		"promptTokenCount", "candidatesTokenCount",
	}
	for _, f := range fields {
		v, ok := usage[f]
		if !ok {
			continue
		}
		n, ok := numericPositive(v)
		if ok && n > 0 {
			return true
		}
	}
	return false
}

// ExtractUsage pulls a normalized usage payload out of a stream chunk in any
// of the supported formats: Claude message_delta, OpenAI Responses
// response.completed, OpenAI chunk.usage, Gemini usageMetadata, and Ollama
// done=true. Mirrors open-sse/utils/usageTracking.js:172-238.
func ExtractUsage(chunk map[string]any) map[string]any {
	if chunk == nil {
		return nil
	}

	// Claude format (message_delta event).
	if typ, _ := chunk["type"].(string); typ == "message_delta" {
		if usage, ok := chunk["usage"].(map[string]any); ok {
			return NormalizeUsage(map[string]any{
				"prompt_tokens":                 getAny(usage, "input_tokens"),
				"completion_tokens":             getAny(usage, "output_tokens"),
				"cache_read_input_tokens":       usage["cache_read_input_tokens"],
				"cache_creation_input_tokens":   usage["cache_creation_input_tokens"],
			})
		}
	}

	// OpenAI Responses API format (response.completed or response.done).
	if typ, _ := chunk["type"].(string); typ == "response.completed" || typ == "response.done" {
		if resp, ok := chunk["response"].(map[string]any); ok {
			if usage, ok := resp["usage"].(map[string]any); ok {
				var cachedTokens any
				if details, ok := usage["input_tokens_details"].(map[string]any); ok {
					cachedTokens = details["cached_tokens"]
				}
				return NormalizeUsage(map[string]any{
					"prompt_tokens":     firstNonNil(usage["input_tokens"], usage["prompt_tokens"]),
					"completion_tokens": firstNonNil(usage["output_tokens"], usage["completion_tokens"]),
					"cached_tokens":     cachedTokens,
					"reasoning_tokens":  getNestedAny(usage, "output_tokens_details", "reasoning_tokens"),
					"prompt_tokens_details": func() any {
						if cachedTokens == nil {
							return nil
						}
						return map[string]any{"cached_tokens": cachedTokens}
					}(),
				})
			}
		}
	}

	// OpenAI format (also covers DeepSeek which uses prompt_cache_hit_tokens).
	if usage, ok := chunk["usage"].(map[string]any); ok {
		if _, hasPrompt := usage["prompt_tokens"]; hasPrompt {
			var cachedTokens any
			if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
				cachedTokens = details["cached_tokens"]
			}
			if cachedTokens == nil {
				cachedTokens = usage["prompt_cache_hit_tokens"]
			}
			var reasoningTokens any
			if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
				reasoningTokens = details["reasoning_tokens"]
			}
			var promptDetails any = usage["prompt_tokens_details"]
			if promptDetails == nil {
				promptDetails = nil
			}
			return NormalizeUsage(map[string]any{
				"prompt_tokens":              usage["prompt_tokens"],
				"completion_tokens":          usage["completion_tokens"],
				"cached_tokens":              cachedTokens,
				"reasoning_tokens":           reasoningTokens,
				"prompt_tokens_details":      promptDetails,
				"completion_tokens_details":  usage["completion_tokens_details"],
			})
		}
	}

	// Gemini format (Antigravity) — usageMetadata may be at top level or
	// nested under response.
	var usageMeta map[string]any
	if m, ok := chunk["usageMetadata"].(map[string]any); ok {
		usageMeta = m
	} else if resp, ok := chunk["response"].(map[string]any); ok {
		if m, ok := resp["usageMetadata"].(map[string]any); ok {
			usageMeta = m
		}
	}
	if usageMeta != nil {
		return NormalizeUsage(map[string]any{
			"prompt_tokens":     usageMeta["promptTokenCount"],
			"completion_tokens": usageMeta["candidatesTokenCount"],
			"total_tokens":      usageMeta["totalTokenCount"],
			"cached_tokens":     usageMeta["cachedContentTokenCount"],
			"reasoning_tokens":  usageMeta["thoughtsTokenCount"],
		})
	}

	// Ollama NDJSON (raw from provider, before translation).
	if done, _ := chunk["done"].(bool); done {
		if _, ok := chunk["prompt_eval_count"]; ok {
			return NormalizeUsage(map[string]any{
				"prompt_tokens":     chunk["prompt_eval_count"],
				"completion_tokens": chunk["eval_count"],
				"total_tokens":      totalFromOllama(chunk),
			})
		}
	}

	return nil
}

// AddBufferToUsage adds BufferTokens to input/prompt fields and recomputes
// total_tokens to match. Mirrors open-sse/utils/usageTracking.js:19,31-55.
func AddBufferToUsage(usage map[string]any) map[string]any {
	if usage == nil {
		return usage
	}
	result := make(map[string]any, len(usage))
	for k, v := range usage {
		result[k] = v
	}

	if v, ok := result["input_tokens"]; ok {
		result["input_tokens"] = toInt(v) + BufferTokens
	}
	if v, ok := result["prompt_tokens"]; ok {
		result["prompt_tokens"] = toInt(v) + BufferTokens
	}

	if v, ok := result["total_tokens"]; ok {
		result["total_tokens"] = toInt(v) + BufferTokens
	} else if pt, okPt := result["prompt_tokens"]; okPt {
		if ct, okCt := result["completion_tokens"]; okCt {
			result["total_tokens"] = toInt(pt) + toInt(ct)
		}
	}

	return result
}

// FilterUsageForFormat keeps only the fields valid for the target format.
// Mirrors open-sse/utils/usageTracking.js:57-113.
func FilterUsageForFormat(usage map[string]any, targetFormat Format) map[string]any {
	if usage == nil {
		return usage
	}

	fields := openaiFields // default = OpenAI
	switch targetFormat {
	case FormatClaude:
		fields = claudeFields
	case FormatGemini, FormatGeminiCLI, FormatAntigravity, FormatVertex:
		fields = geminiFields
	case FormatOpenAIResponses, FormatOpenAIResponse:
		fields = openaiResponsesFields
	}

	filtered := make(map[string]any, len(fields))
	for _, f := range fields {
		if v, ok := usage[f]; ok {
			filtered[f] = v
		}
	}
	return filtered
}

// EstimateInputTokens estimates input tokens from the request body by
// JSON-encoding it and dividing the byte length by 4. Mirrors
// open-sse/utils/usageTracking.js:240-255.
func EstimateInputTokens(body map[string]any) int {
	if body == nil {
		return 0
	}
	b, err := json.Marshal(body)
	if err != nil {
		return 0
	}
	// ceil(len / 4)
	return (len(b) + 3) / 4
}

// EstimateOutputTokens estimates output tokens from accumulated content
// length. Mirrors open-sse/utils/usageTracking.js:259-262.
func EstimateOutputTokens(contentLength int) int {
	if contentLength <= 0 {
		return 0
	}
	if contentLength/4 < 1 {
		return 1
	}
	return contentLength / 4
}

// FormatUsage builds an estimated usage payload for the target format. The
// returned map is in the shape expected by the client and already has the
// buffer applied. Mirrors open-sse/utils/usageTracking.js:270-285.
func FormatUsage(inputTokens, outputTokens int, targetFormat Format) map[string]any {
	if targetFormat == FormatClaude {
		return AddBufferToUsage(map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"estimated":     true,
		})
	}
	return AddBufferToUsage(map[string]any{
		"prompt_tokens":     inputTokens,
		"completion_tokens": outputTokens,
		"total_tokens":      inputTokens + outputTokens,
		"estimated":         true,
	})
}

// EstimateUsage builds a FormatUsage from the request body and accumulated
// content length. Mirrors open-sse/utils/usageTracking.js:295-305.
func EstimateUsage(body map[string]any, contentLength int, targetFormat Format) map[string]any {
	return FormatUsage(
		EstimateInputTokens(body),
		EstimateOutputTokens(contentLength),
		targetFormat,
	)
}

// --- helpers ---

var (
	claudeFields = []string{
		"input_tokens", "output_tokens",
		"cache_read_input_tokens", "cache_creation_input_tokens",
		"estimated",
	}
	geminiFields = []string{
		"promptTokenCount", "candidatesTokenCount", "totalTokenCount",
		"cachedContentTokenCount", "thoughtsTokenCount",
		"estimated",
	}
	openaiResponsesFields = []string{
		"input_tokens", "output_tokens",
		"input_tokens_details", "output_tokens_details",
		"estimated",
	}
	openaiFields = []string{
		"prompt_tokens", "completion_tokens", "total_tokens",
		"cached_tokens", "reasoning_tokens",
		"prompt_tokens_details", "completion_tokens_details",
		"estimated",
	}
)

func isArrayish(m map[string]any) bool {
	// Defensive: maps in Go cannot be arrays; the JS reference rejects
	// array inputs. Keep this hook so future ref changes can be ported.
	return false
}

func parseFiniteNumber(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	var f float64
	var saw bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9') || c == 'e' || c == 'E' {
			if c >= '0' && c <= '9' {
				saw = true
			}
		} else {
			return 0, false
		}
	}
	if !saw {
		return 0, false
	}
	if err := json.Unmarshal([]byte(s), &f); err != nil {
		return 0, false
	}
	if f != f { // NaN
		return 0, false
	}
	if f > 1e308 || f < -1e308 { // Inf
		return 0, false
	}
	return f, true
}

func numericPositive(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		if x != x {
			return 0, false
		}
		return int(x), true
	case float32:
		if x != x {
			return 0, false
		}
		return int(x), true
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		return int(x), true
	}
	return 0, false
}

func getAny(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func getNestedAny(m map[string]any, keys ...string) any {
	cur := any(m)
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

func firstNonNil(vals ...any) any {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func totalFromOllama(chunk map[string]any) any {
	p := toInt(chunk["prompt_eval_count"])
	c := toInt(chunk["eval_count"])
	return p + c
}
