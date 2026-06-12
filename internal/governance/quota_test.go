package governance

import (
	"sync"
	"testing"
	"time"
)

// fakeSpendReader returns a fixed cost for a given key and lower-bound.
type fakeSpendReader struct {
	mu     sync.Mutex
	values map[string]float64
}

func newFakeSpendReader() *fakeSpendReader {
	return &fakeSpendReader{values: map[string]float64{}}
}

func (f *fakeSpendReader) SumCostByAPIKey(key, sinceISO string) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.values[key], nil
}

func (f *fakeSpendReader) set(key string, cost float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key] = cost
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestVKBudgetExhaustion(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:            "vk-budget",
		BudgetLimit:    1.00,
		BudgetPeriod:   "daily",
		RateLimitRPM:   1000,
	}

	// Under limit.
	spend.set("vk-budget", 0.50)
	ok, status, reason := engine.Allow(vk, "gpt-4o")
	if !ok || status != 0 || reason != "" {
		t.Fatalf("under limit: ok=%v status=%d reason=%q, want ok=true", ok, status, reason)
	}

	// Exactly at limit still allowed (Usage.Used semantics match spend exactly).
	spend.set("vk-budget", 1.00)
	ok, status, reason = engine.Allow(vk, "gpt-4o")
	if !ok || status != 0 || reason != "" {
		t.Fatalf("at limit: ok=%v status=%d reason=%q, want ok=true", ok, status, reason)
	}

	// Over limit.
	spend.set("vk-budget", 1.10)
	ok, status, reason = engine.Allow(vk, "gpt-4o")
	if ok || status != 429 || reason == "" {
		t.Fatalf("over limit: ok=%v status=%d reason=%q, want ok=false status=429", ok, status, reason)
	}
	if reason != "budget exhausted" {
		t.Fatalf("reason = %q, want %q", reason, "budget exhausted")
	}
}

func TestVKRateLimitRPM(t *testing.T) {
	base := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	now := base
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, func() time.Time { return now })

	vk := &VirtualKeyInfo{
		Key:          "vk-rpm",
		RateLimitRPM: 2,
	}

	ok, _, _ := engine.Allow(vk, "gpt-4o")
	if !ok {
		t.Fatal("first request denied")
	}
	ok, _, _ = engine.Allow(vk, "gpt-4o")
	if !ok {
		t.Fatal("second request denied")
	}
	ok, status, reason := engine.Allow(vk, "gpt-4o")
	if ok || status != 429 || reason == "" {
		t.Fatalf("third request: ok=%v status=%d reason=%q, want ok=false status=429", ok, status, reason)
	}
	if reason != "rate limit exceeded" {
		t.Fatalf("reason = %q, want %q", reason, "rate limit exceeded")
	}

	// Advance to the next minute and try again.
	now = base.Add(time.Minute)
	ok, _, _ = engine.Allow(vk, "gpt-4o")
	if !ok {
		t.Fatal("next-minute request denied")
	}
}

func TestVKQuotaConcurrent(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:          "vk-concurrent",
		RateLimitRPM: 2,
	}

	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, _ := engine.Allow(vk, "gpt-4o")
			if ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 2 {
		t.Fatalf("allowed = %d, want 2", allowed)
	}
}
