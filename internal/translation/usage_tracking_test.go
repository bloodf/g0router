package translation

import (
	"math"
	"testing"
)

// TestNormalizeUsage verifies the numeric-coercion rules from
// open-sse/utils/usageTracking.js:115-150: assignNumber drops non-finite values,
// preserves details objects, and returns nil when no numeric field survives.
func TestNormalizeUsage(t *testing.T) {
	t.Run("numeric_fields_kept", func(t *testing.T) {
		got := NormalizeUsage(map[string]any{
			"prompt_tokens":     100,
			"completion_tokens": 50,
			"total_tokens":      150,
		})
		if got == nil {
			t.Fatal("normalize returned nil for valid usage")
		}
		if got["prompt_tokens"] != 100 || got["completion_tokens"] != 50 || got["total_tokens"] != 150 {
			t.Errorf("got %+v, want all numeric fields preserved", got)
		}
	})

	t.Run("string_numbers_coerced", func(t *testing.T) {
		got := NormalizeUsage(map[string]any{
			"prompt_tokens":     "100",
			"completion_tokens": "50",
		})
		if got["prompt_tokens"] != 100 || got["completion_tokens"] != 50 {
			t.Errorf("got %+v, want string numbers coerced", got)
		}
	})

	t.Run("non_finite_dropped", func(t *testing.T) {
		got := NormalizeUsage(map[string]any{
			"prompt_tokens":     math.Inf(1),
			"completion_tokens": math.NaN(),
		})
		if got != nil {
			t.Errorf("expected nil for all non-finite values, got %+v", got)
		}
	})

	t.Run("empty_returns_nil", func(t *testing.T) {
		if got := NormalizeUsage(map[string]any{}); got != nil {
			t.Errorf("expected nil for empty input, got %+v", got)
		}
	})

	t.Run("nil_returns_nil", func(t *testing.T) {
		if got := NormalizeUsage(nil); got != nil {
			t.Errorf("expected nil for nil input, got %+v", got)
		}
	})

	t.Run("details_preserved", func(t *testing.T) {
		details := map[string]any{"cached_tokens": 10}
		got := NormalizeUsage(map[string]any{
			"prompt_tokens":         100,
			"prompt_tokens_details": details,
		})
		if got["prompt_tokens"] != 100 {
			t.Errorf("prompt_tokens = %v, want 100", got["prompt_tokens"])
		}
		if got["prompt_tokens_details"] == nil {
			t.Error("prompt_tokens_details should be preserved")
		}
	})
}

// TestHasValidUsage mirrors usageTracking.js:150-170: any of the listed token
// fields with value > 0 is valid.
func TestHasValidUsage(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		want bool
	}{
		{"nil", nil, false},
		{"empty", map[string]any{}, false},
		{"all_zero", map[string]any{"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0}, false},
		{"openai_prompt", map[string]any{"prompt_tokens": 1}, true},
		{"claude_input", map[string]any{"input_tokens": 1}, true},
		{"gemini_candidates", map[string]any{"candidatesTokenCount": 1}, true},
		{"only_string", map[string]any{"prompt_tokens": "1"}, false}, // strings don't count
		{"negative", map[string]any{"prompt_tokens": -1}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasValidUsage(c.in); got != c.want {
				t.Errorf("HasValidUsage(%+v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// TestExtractUsageFormats exercises each source shape from
// usageTracking.js:172-238: Claude message_delta, OpenAI Responses response.completed,
// OpenAI chunk.usage, Gemini usageMetadata, Ollama done=true.
func TestExtractUsageFormats(t *testing.T) {
	t.Run("claude_message_delta", func(t *testing.T) {
		chunk := map[string]any{
			"type": "message_delta",
			"usage": map[string]any{
				"input_tokens":  120,
				"output_tokens": 30,
			},
		}
		got := ExtractUsage(chunk)
		if got == nil {
			t.Fatal("expected non-nil usage")
		}
		if got["prompt_tokens"] != 120 || got["completion_tokens"] != 30 {
			t.Errorf("got %+v, want prompt=120 completion=30", got)
		}
	})

	t.Run("openai_responses_completed", func(t *testing.T) {
		chunk := map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"usage": map[string]any{
					"input_tokens":  200,
					"output_tokens": 80,
					"input_tokens_details": map[string]any{
						"cached_tokens": 5,
					},
				},
			},
		}
		got := ExtractUsage(chunk)
		if got == nil {
			t.Fatal("expected non-nil usage")
		}
		if got["prompt_tokens"] != 200 || got["completion_tokens"] != 80 {
			t.Errorf("got %+v, want prompt=200 completion=80", got)
		}
		if got["cached_tokens"] != 5 {
			t.Errorf("got %+v, want cached_tokens=5", got)
		}
	})

	t.Run("openai_chunk_usage", func(t *testing.T) {
		chunk := map[string]any{
			"id": "c1",
			"usage": map[string]any{
				"prompt_tokens":     15,
				"completion_tokens": 7,
				"total_tokens":      22,
			},
		}
		got := ExtractUsage(chunk)
		if got == nil {
			t.Fatal("expected non-nil usage")
		}
		if got["prompt_tokens"] != 15 || got["completion_tokens"] != 7 {
			t.Errorf("got %+v, want prompt=15 completion=7", got)
		}
	})

	t.Run("gemini_usage_metadata", func(t *testing.T) {
		chunk := map[string]any{
			"usageMetadata": map[string]any{
				"promptTokenCount":     9,
				"candidatesTokenCount": 4,
				"totalTokenCount":      13,
			},
		}
		got := ExtractUsage(chunk)
		if got == nil {
			t.Fatal("expected non-nil usage")
		}
		if got["prompt_tokens"] != 9 || got["completion_tokens"] != 4 {
			t.Errorf("got %+v, want prompt=9 completion=4", got)
		}
	})

	t.Run("ollama_done", func(t *testing.T) {
		chunk := map[string]any{
			"done":               true,
			"prompt_eval_count":  11,
			"eval_count":         3,
		}
		got := ExtractUsage(chunk)
		if got == nil {
			t.Fatal("expected non-nil usage")
		}
		if got["prompt_tokens"] != 11 || got["completion_tokens"] != 3 {
			t.Errorf("got %+v, want prompt=11 completion=3", got)
		}
	})

	t.Run("no_usage_returns_nil", func(t *testing.T) {
		if got := ExtractUsage(map[string]any{"id": "c1"}); got != nil {
			t.Errorf("expected nil for chunk without usage, got %+v", got)
		}
	})
}

// TestAddBufferToUsage covers the BUFFER_TOKENS = 2000 add + total recompute
// rules from usageTracking.js:31-55.
func TestAddBufferToUsage(t *testing.T) {
	t.Run("openai_format", func(t *testing.T) {
		got := AddBufferToUsage(map[string]any{
			"prompt_tokens":     1000,
			"completion_tokens": 500,
		})
		if got["prompt_tokens"] != 3000 {
			t.Errorf("prompt_tokens = %v, want 3000 (1000+2000)", got["prompt_tokens"])
		}
		if got["completion_tokens"] != 500 {
			t.Errorf("completion_tokens = %v, want 500 (untouched)", got["completion_tokens"])
		}
		if got["total_tokens"] != 3500 {
			t.Errorf("total_tokens = %v, want 3500 (recomputed)", got["total_tokens"])
		}
	})

	t.Run("claude_format", func(t *testing.T) {
		got := AddBufferToUsage(map[string]any{
			"input_tokens":  100,
			"output_tokens": 50,
		})
		if got["input_tokens"] != 2100 {
			t.Errorf("input_tokens = %v, want 2100 (100+2000)", got["input_tokens"])
		}
		if got["output_tokens"] != 50 {
			t.Errorf("output_tokens = %v, want 50 (untouched)", got["output_tokens"])
		}
	})

	t.Run("existing_total_incremented", func(t *testing.T) {
		got := AddBufferToUsage(map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 5,
			"total_tokens":      15,
		})
		if got["total_tokens"] != 2015 {
			t.Errorf("total_tokens = %v, want 2015 (15+2000)", got["total_tokens"])
		}
	})

	t.Run("nil_passthrough", func(t *testing.T) {
		if got := AddBufferToUsage(nil); got != nil {
			t.Errorf("nil in should pass through, got %+v", got)
		}
	})
}

// TestEstimateUsage covers the golden cases: ceil(len(JSON(body))/4) for input,
// max(1, floor(contentLen/4)) for output, claude vs openai shape, estimated flag.
func TestEstimateUsage(t *testing.T) {
	// The body marshals to {"messages":[{"role":"user","content":"0123456789"}]}
	// = 55 chars → ceil(55/4) = 14 input tokens before buffer.
	body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "0123456789"}}}

	t.Run("claude_format", func(t *testing.T) {
		// contentLength 10 → floor(10/4) = 2 output tokens.
		// Buffer (+2000) is applied to input only.
		got := EstimateUsage(body, 10, FormatClaude)
		if got == nil {
			t.Fatal("expected non-nil usage")
		}
		if got["input_tokens"] != 2014 {
			t.Errorf("input_tokens = %v, want 2014 (14+2000 buffer)", got["input_tokens"])
		}
		if got["output_tokens"] != 2 {
			t.Errorf("output_tokens = %v, want 2 (estimated)", got["output_tokens"])
		}
		if got["estimated"] != true {
			t.Error("estimated flag should be true")
		}
	})

	t.Run("openai_format_has_total", func(t *testing.T) {
		got := EstimateUsage(body, 10, FormatOpenAI)
		if got == nil {
			t.Fatal("expected non-nil usage")
		}
		// 14 input + 2 output = 16, +2000 buffer on prompt = 2014+2 = 2016
		if got["prompt_tokens"] != 2014 {
			t.Errorf("prompt_tokens = %v, want 2014 (14+2000 buffer)", got["prompt_tokens"])
		}
		if got["completion_tokens"] != 2 {
			t.Errorf("completion_tokens = %v, want 2 (estimated)", got["completion_tokens"])
		}
		if got["total_tokens"] != 2016 {
			// After buffer: 2014 + 2 = 2016
			t.Errorf("total_tokens = %v, want 2016 (2014+2)", got["total_tokens"])
		}
		if got["estimated"] != true {
			t.Error("estimated flag should be true")
		}
	})

	t.Run("zero_content_returns_zero", func(t *testing.T) {
		// contentLength 0 → ref returns 0 (ref guards with `if (!contentLength)`).
		got := EstimateUsage(map[string]any{}, 0, FormatOpenAI)
		if got["completion_tokens"] != 0 {
			t.Errorf("completion_tokens = %v, want 0 (ref guards with !contentLength)", got["completion_tokens"])
		}
	})

	t.Run("min_output_one_with_content", func(t *testing.T) {
		// contentLength 1..3 → floor(n/4) = 0, max(1, 0) = 1 output token.
		got := EstimateUsage(map[string]any{}, 1, FormatOpenAI)
		if got["completion_tokens"] != 1 {
			t.Errorf("completion_tokens = %v, want 1 (max(1, floor(1/4)))", got["completion_tokens"])
		}
	})

	t.Run("no_body_input_zero", func(t *testing.T) {
		got := EstimateUsage(nil, 8, FormatOpenAI)
		if got["prompt_tokens"] != 2000 {
			// 0 + 2000 buffer
			t.Errorf("prompt_tokens = %v, want 2000 (0+2000)", got["prompt_tokens"])
		}
	})
}

// TestFilterUsageForFormat covers the per-format field set from
// usageTracking.js:57-113.
func TestFilterUsageForFormat(t *testing.T) {
	fullUsage := map[string]any{
		"input_tokens":              100,
		"output_tokens":             50,
		"prompt_tokens":             100,
		"completion_tokens":         50,
		"total_tokens":              150,
		"cache_read_input_tokens":   10,
		"cache_creation_input_tokens": 5,
		"cached_tokens":             7,
		"reasoning_tokens":          3,
		"prompt_tokens_details":     map[string]any{"cached_tokens": 4},
		"completion_tokens_details": map[string]any{"reasoning_tokens": 1},
		"promptTokenCount":          9,
		"candidatesTokenCount":      4,
		"estimated":                 true,
	}

	t.Run("claude_fields", func(t *testing.T) {
		got := FilterUsageForFormat(fullUsage, FormatClaude)
		for _, k := range []string{"input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "estimated"} {
			if _, ok := got[k]; !ok {
				t.Errorf("claude format should include %q, got %+v", k, got)
			}
		}
		for _, k := range []string{"prompt_tokens", "completion_tokens", "total_tokens", "cached_tokens", "reasoning_tokens", "promptTokenCount", "candidatesTokenCount"} {
			if _, ok := got[k]; ok {
				t.Errorf("claude format should not include %q, got %+v", k, got)
			}
		}
	})

	t.Run("gemini_fields", func(t *testing.T) {
		got := FilterUsageForFormat(fullUsage, FormatGemini)
		for _, k := range []string{"promptTokenCount", "candidatesTokenCount", "estimated"} {
			if _, ok := got[k]; !ok {
				t.Errorf("gemini format should include %q, got %+v", k, got)
			}
		}
		if _, ok := got["prompt_tokens"]; ok {
			t.Errorf("gemini format should not include prompt_tokens")
		}
	})

	t.Run("openai_default_fields", func(t *testing.T) {
		got := FilterUsageForFormat(fullUsage, FormatOpenAI)
		for _, k := range []string{"prompt_tokens", "completion_tokens", "total_tokens", "cached_tokens", "reasoning_tokens", "prompt_tokens_details", "completion_tokens_details", "estimated"} {
			if _, ok := got[k]; !ok {
				t.Errorf("openai format should include %q, got %+v", k, got)
			}
		}
		if _, ok := got["input_tokens"]; ok {
			t.Errorf("openai format should not include input_tokens")
		}
	})

	t.Run("openai_responses_fields", func(t *testing.T) {
		got := FilterUsageForFormat(fullUsage, FormatOpenAIResponses)
		for _, k := range []string{"input_tokens", "output_tokens", "estimated"} {
			if _, ok := got[k]; !ok {
				t.Errorf("openai-responses format should include %q, got %+v", k, got)
			}
		}
		if _, ok := got["prompt_tokens"]; ok {
			t.Errorf("openai-responses should not include prompt_tokens")
		}
	})
}
