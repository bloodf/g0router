package translation

import "testing"

func TestAdjustMaxTokensDefault(t *testing.T) {
	if got := AdjustMaxTokens(map[string]any{}); got != 64000 {
		t.Errorf("missing max_tokens = %d, want 64000", got)
	}
	if got := AdjustMaxTokens(map[string]any{"max_tokens": 0}); got != 64000 {
		t.Errorf("zero max_tokens = %d, want 64000", got)
	}
	if got := AdjustMaxTokens(map[string]any{"max_tokens": nil}); got != 64000 {
		t.Errorf("nil max_tokens = %d, want 64000", got)
	}
}

func TestAdjustMaxTokensToolBoost(t *testing.T) {
	body := map[string]any{
		"max_tokens": 1000,
		"tools":      []any{map[string]any{"name": "test"}},
	}
	if got := AdjustMaxTokens(body); got != 32000 {
		t.Errorf("tools with low max_tokens = %d, want 32000", got)
	}

	// No tools - keep value
	body2 := map[string]any{"max_tokens": 1000}
	if got := AdjustMaxTokens(body2); got != 1000 {
		t.Errorf("no tools = %d, want 1000", got)
	}

	// Tools but value already >= 32000
	body3 := map[string]any{
		"max_tokens": 50000,
		"tools":      []any{map[string]any{"name": "test"}},
	}
	if got := AdjustMaxTokens(body3); got != 50000 {
		t.Errorf("tools with high max_tokens = %d, want 50000", got)
	}
}

func TestAdjustMaxTokensThinkingBudget(t *testing.T) {
	body := map[string]any{
		"max_tokens": 5000,
		"thinking":   map[string]any{"budget_tokens": 6000},
	}
	if got := AdjustMaxTokens(body); got != 7024 {
		t.Errorf("thinking budget > max_tokens = %d, want 7024", got)
	}

	// max_tokens > budget - keep value
	body2 := map[string]any{
		"max_tokens": 10000,
		"thinking":   map[string]any{"budget_tokens": 6000},
	}
	if got := AdjustMaxTokens(body2); got != 10000 {
		t.Errorf("thinking budget < max_tokens = %d, want 10000", got)
	}
}

func TestAdjustMaxTokensExplicitValueKept(t *testing.T) {
	body := map[string]any{"max_tokens": 50000}
	if got := AdjustMaxTokens(body); got != 50000 {
		t.Errorf("explicit value = %d, want 50000", got)
	}
}
