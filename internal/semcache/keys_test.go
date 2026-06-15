package semcache

import "testing"

// TestCacheKeyDeterministic verifies the exact-key hash is stable for the same
// (model, prompt) pair: CacheKey is a pure sha256 over the normalized input
// (D1), so identical inputs must always produce the identical key.
func TestCacheKeyDeterministic(t *testing.T) {
	a := CacheKey("gpt-4", "hello world")
	b := CacheKey("gpt-4", "hello world")
	if a != b {
		t.Fatalf("CacheKey not deterministic: %q != %q", a, b)
	}
	if a == "" {
		t.Fatal("CacheKey returned empty string")
	}
	// sha256 hex is 64 chars.
	if len(a) != 64 {
		t.Fatalf("CacheKey len = %d, want 64 (sha256 hex)", len(a))
	}
}

// TestCacheKeyDiffersByModel verifies the model is part of the key: the same
// prompt under a different model must not collide (the cache is per-model).
func TestCacheKeyDiffersByModel(t *testing.T) {
	a := CacheKey("gpt-4", "hello world")
	b := CacheKey("gpt-3.5", "hello world")
	if a == b {
		t.Fatal("CacheKey collided across models")
	}
}

// TestCacheKeyDiffersByPrompt verifies the prompt is part of the key: a
// different prompt under the same model must not collide.
func TestCacheKeyDiffersByPrompt(t *testing.T) {
	a := CacheKey("gpt-4", "hello world")
	b := CacheKey("gpt-4", "goodbye world")
	if a == b {
		t.Fatal("CacheKey collided across prompts")
	}
}

// TestCacheKeyNormalizationStable verifies normalization folds insignificant
// surrounding whitespace so semantically identical prompts hit the same key
// (the documented prompt-normalization, D1/keys.go).
func TestCacheKeyNormalizationStable(t *testing.T) {
	a := CacheKey("gpt-4", "  hello world  ")
	b := CacheKey("gpt-4", "hello world")
	if a != b {
		t.Fatalf("CacheKey did not normalize surrounding whitespace: %q != %q", a, b)
	}
}

// TestCacheKeyModelPromptNoMerge verifies the model/prompt boundary cannot be
// forged by shifting characters across it (a separator keeps the fields
// distinct so ("ab","c") and ("a","bc") never collide).
func TestCacheKeyModelPromptNoMerge(t *testing.T) {
	a := CacheKey("ab", "c")
	b := CacheKey("a", "bc")
	if a == b {
		t.Fatal("CacheKey merged model/prompt boundary")
	}
}
