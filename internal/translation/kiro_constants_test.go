package translation

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"
)

func TestResolveKiroModel(t *testing.T) {
	cases := []struct {
		model           string
		wantUpstream    string
		wantAgentic     bool
		wantThinking    bool
	}{
		{"claude-sonnet-4.5-thinking-agentic", "claude-sonnet-4.5", true, true},
		{"claude-sonnet-4.5-agentic", "claude-sonnet-4.5", true, false},
		{"claude-sonnet-4.5-thinking", "claude-sonnet-4.5", false, true},
		{"claude-sonnet-4.5", "claude-sonnet-4.5", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			upstream, agentic, thinking := ResolveKiroModel(tc.model)
			if upstream != tc.wantUpstream {
				t.Errorf("upstream = %q, want %q", upstream, tc.wantUpstream)
			}
			if agentic != tc.wantAgentic {
				t.Errorf("agentic = %v, want %v", agentic, tc.wantAgentic)
			}
			if thinking != tc.wantThinking {
				t.Errorf("thinking = %v, want %v", thinking, tc.wantThinking)
			}
		})
	}
}

func TestIsThinkingEnabled(t *testing.T) {
	// Anthropic-Beta header
	t.Run("header_interleaved_thinking", func(t *testing.T) {
		if !isThinkingEnabled(nil, map[string]string{"anthropic-beta": "interleaved-thinking"}, "") {
			t.Error("expected true for anthropic-beta header")
		}
	})
	t.Run("header_case_insensitive_key", func(t *testing.T) {
		if !isThinkingEnabled(nil, map[string]string{"Anthropic-Beta": "interleaved-thinking"}, "") {
			t.Error("expected true for Anthropic-Beta header (case-insensitive key)")
		}
	})
	t.Run("header_no_match", func(t *testing.T) {
		if isThinkingEnabled(nil, map[string]string{"anthropic-beta": "other"}, "") {
			t.Error("expected false for unrelated beta header")
		}
	})

	// Claude thinking object
	t.Run("claude_thinking_enabled", func(t *testing.T) {
		body := map[string]any{"thinking": map[string]any{"type": "enabled", "budget_tokens": 1000}}
		if !isThinkingEnabled(body, nil, "") {
			t.Error("expected true for thinking.type=enabled")
		}
	})
	t.Run("claude_thinking_enabled_infinite_budget", func(t *testing.T) {
		body := map[string]any{"thinking": map[string]any{"type": "enabled", "budget_tokens": math.Inf(1)}}
		if !isThinkingEnabled(body, nil, "") {
			t.Error("expected true for thinking.type=enabled with infinite budget")
		}
	})
	t.Run("claude_thinking_disabled_zero_budget", func(t *testing.T) {
		body := map[string]any{"thinking": map[string]any{"type": "enabled", "budget_tokens": 0}}
		if isThinkingEnabled(body, nil, "") {
			t.Error("expected false for thinking.type=enabled with budget=0")
		}
	})

	// OpenAI reasoning_effort
	t.Run("reasoning_effort_low", func(t *testing.T) {
		if !isThinkingEnabled(map[string]any{"reasoning_effort": "low"}, nil, "") {
			t.Error("expected true for reasoning_effort=low")
		}
	})
	t.Run("reasoning_effort_medium", func(t *testing.T) {
		if !isThinkingEnabled(map[string]any{"reasoning_effort": "medium"}, nil, "") {
			t.Error("expected true for reasoning_effort=medium")
		}
	})
	t.Run("reasoning_effort_high", func(t *testing.T) {
		if !isThinkingEnabled(map[string]any{"reasoning_effort": "high"}, nil, "") {
			t.Error("expected true for reasoning_effort=high")
		}
	})
	t.Run("reasoning_effort_auto", func(t *testing.T) {
		if !isThinkingEnabled(map[string]any{"reasoning_effort": "auto"}, nil, "") {
			t.Error("expected true for reasoning_effort=auto")
		}
	})
	t.Run("reasoning_effort_none", func(t *testing.T) {
		if isThinkingEnabled(map[string]any{"reasoning_effort": "none"}, nil, "") {
			t.Error("expected false for reasoning_effort=none")
		}
	})

	// OpenAI reasoning.effort
	t.Run("reasoning_dot_effort", func(t *testing.T) {
		body := map[string]any{"reasoning": map[string]any{"effort": "high"}}
		if !isThinkingEnabled(body, nil, "") {
			t.Error("expected true for reasoning.effort=high")
		}
	})

	// thinking_mode tag in messages
	t.Run("thinking_mode_tag_in_system", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{"role": "system", "content": "<thinking_mode>enabled</thinking_mode> do x"},
			},
		}
		if !isThinkingEnabled(body, nil, "") {
			t.Error("expected true for thinking_mode tag in system message")
		}
	})
	t.Run("thinking_mode_tag_in_user", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{"role": "user", "content": "<thinking_mode>interleaved</thinking_mode> do x"},
			},
		}
		if !isThinkingEnabled(body, nil, "") {
			t.Error("expected true for thinking_mode tag in user message")
		}
	})
	t.Run("thinking_mode_tag_in_body_system", func(t *testing.T) {
		body := map[string]any{"system": "<thinking_mode>enabled</thinking_mode> do x"}
		if !isThinkingEnabled(body, nil, "") {
			t.Error("expected true for thinking_mode tag in body.system")
		}
	})
	t.Run("thinking_mode_tag_in_array_content", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "<thinking_mode>enabled</thinking_mode>"}}},
			},
		}
		if !isThinkingEnabled(body, nil, "") {
			t.Error("expected true for thinking_mode tag in array content")
		}
	})

	// Model name hints
	t.Run("model_contains_thinking", func(t *testing.T) {
		if !isThinkingEnabled(nil, nil, "claude-thinking") {
			t.Error("expected true for model containing thinking")
		}
	})
	t.Run("model_contains_reason", func(t *testing.T) {
		if !isThinkingEnabled(nil, nil, "o3-reason") {
			t.Error("expected true for model containing -reason")
		}
	})

	// Negative: no triggers
	t.Run("no_triggers", func(t *testing.T) {
		if isThinkingEnabled(map[string]any{"temperature": 0.7}, map[string]string{"content-type": "application/json"}, "gpt-4") {
			t.Error("expected false with no triggers")
		}
	})
}

func TestBuildThinkingSystemPrefix(t *testing.T) {
	t.Run("default_budget", func(t *testing.T) {
		got := buildThinkingSystemPrefix(0)
		want := "<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>16000</max_thinking_length>"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("clamp_low", func(t *testing.T) {
		got := buildThinkingSystemPrefix(-5)
		want := "<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>1</max_thinking_length>"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("clamp_high", func(t *testing.T) {
		got := buildThinkingSystemPrefix(50000)
		want := "<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>32000</max_thinking_length>"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("custom_budget", func(t *testing.T) {
		got := buildThinkingSystemPrefix(8000)
		want := "<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>8000</max_thinking_length>"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestKiroAgenticPromptByteExact(t *testing.T) {
	hash := sha256.Sum256([]byte(kiroAgenticSystemPrompt))
	got := hex.EncodeToString(hash[:])
	want := "df38d752b7913306e1d8885a32134e9ce214cb5b7303c979852a23e5e6080f6a"
	if got != want {
		t.Errorf("sha256 = %s, want %s", got, want)
	}
	if len(kiroAgenticSystemPrompt) != 1864 {
		t.Errorf("len = %d, want 1864", len(kiroAgenticSystemPrompt))
	}
	// Derivation: length=1864, sha256=df38d752b7913306e1d8885a32134e9ce214cb5b7303c979852a23e5e6080f6a
}
