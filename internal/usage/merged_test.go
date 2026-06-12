package usage

import (
	"testing"
)

type fakeOverrideStore struct {
	calls int
	data  map[string]map[string]map[string]float64
}

func (f *fakeOverrideStore) UserPricing() (map[string]map[string]map[string]float64, error) {
	f.calls++
	return f.data, nil
}

func TestMergedPricingAndCache(t *testing.T) {
	store := &fakeOverrideStore{
		data: map[string]map[string]map[string]float64{
			"gh": {
				"gpt-5.3-codex": {"input": 1.00},
			},
			"custom": {
				"custom-model": {"input": 0.10, "output": 0.20},
			},
		},
	}
	var now int64 = 0
	clock := func() int64 { return now }
	r := NewResolver(store, clock)

	// First call reads the store.
	merged, err := r.Merged()
	if err != nil {
		t.Fatalf("Merged #1: %v", err)
	}
	if store.calls != 1 {
		t.Errorf("after first Merged, store.calls = %d, want 1", store.calls)
	}

	// gh/gpt-5.3-codex: input overridden, output stays canonical.
	gh, ok := merged["gh"]
	if !ok {
		t.Fatal("missing gh provider in merged view")
	}
	ghModel, ok := gh["gpt-5.3-codex"]
	if !ok {
		t.Fatal("missing gh/gpt-5.3-codex in merged view")
	}
	if ghModel.Input != 1.00 {
		t.Errorf("merged gh/gpt-5.3-codex input = %v, want 1.00", ghModel.Input)
	}
	if ghModel.Output != 14.00 {
		t.Errorf("merged gh/gpt-5.3-codex output = %v, want 14.00", ghModel.Output)
	}

	// user-only provider appears.
	custom, ok := merged["custom"]
	if !ok {
		t.Fatal("missing custom provider in merged view")
	}
	customModel, ok := custom["custom-model"]
	if !ok {
		t.Fatal("missing custom/custom-model in merged view")
	}
	if customModel.Input != 0.10 || customModel.Output != 0.20 {
		t.Errorf("merged custom/custom-model = %+v, want input=0.10 output=0.20", customModel)
	}
	// absent rates are zero for user-only entries
	if customModel.Cached != 0 || customModel.Reasoning != 0 {
		t.Errorf("merged custom/custom-model absent rates = cached=%v reasoning=%v, want 0", customModel.Cached, customModel.Reasoning)
	}

	// Second call within TTL does not re-read store.
	now += 1000
	_, err = r.Merged()
	if err != nil {
		t.Fatalf("Merged #2: %v", err)
	}
	if store.calls != 1 {
		t.Errorf("after cached Merged, store.calls = %d, want 1", store.calls)
	}

	// After TTL expiry, re-reads.
	now += 5000
	_, err = r.Merged()
	if err != nil {
		t.Fatalf("Merged #3 after TTL: %v", err)
	}
	if store.calls != 2 {
		t.Errorf("after expired Merged, store.calls = %d, want 2", store.calls)
	}

	// Invalidate forces re-read.
	now += 1000
	r.Invalidate()
	_, err = r.Merged()
	if err != nil {
		t.Fatalf("Merged #4 after Invalidate: %v", err)
	}
	if store.calls != 3 {
		t.Errorf("after Invalidate, store.calls = %d, want 3", store.calls)
	}
}
