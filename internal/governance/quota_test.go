package governance

import (
	"sync"
	"testing"
	"time"
)

// fakeSpendReader returns a fixed cost for a given key and lower-bound. It also
// holds a team-keyed map for the 2-level hierarchy's SumCostByTeam (bf-gov-1, D8).
type fakeSpendReader struct {
	mu        sync.Mutex
	values    map[string]float64
	teamSpend map[string]float64
	tokens    map[string]int64
	requests  map[string]int64
}

func newFakeSpendReader() *fakeSpendReader {
	return &fakeSpendReader{
		values:    map[string]float64{},
		teamSpend: map[string]float64{},
		tokens:    map[string]int64{},
		requests:  map[string]int64{},
	}
}

func (f *fakeSpendReader) SumCostByAPIKey(key, sinceISO string) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.values[key], nil
}

func (f *fakeSpendReader) SumCostByTeam(teamID, sinceISO string) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.teamSpend[teamID], nil
}

// SumTokensByAPIKey / SumRequestsByAPIKey are the in-memory analogues of the
// store's SQL-live token/request aggregates (bf-gov-3, D1/D3/D7). Window
// rollover is simulated by clearing the maps (the same effect the SQL lower
// bound has when prior rows fall out of the window).
func (f *fakeSpendReader) SumTokensByAPIKey(key, sinceISO string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tokens[key], nil
}

func (f *fakeSpendReader) SumRequestsByAPIKey(key, sinceISO string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[key], nil
}

func (f *fakeSpendReader) set(key string, cost float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key] = cost
}

func (f *fakeSpendReader) setTeam(teamID string, cost float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.teamSpend[teamID] = cost
}

func (f *fakeSpendReader) setTokens(key string, tokens int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens[key] = tokens
}

func (f *fakeSpendReader) setRequests(key string, count int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests[key] = count
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

// TestTeamBudgetExhaustion verifies the 2-level hierarchy (bf-gov-1, D3/D8): a VK
// whose own budget passes is still DENIED 429 "team budget exhausted" when the
// team's aggregate SumCostByTeam exceeds the team budget.
func TestTeamBudgetExhaustion(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:             "vk-team-budget",
		BudgetLimit:     10.0,
		BudgetPeriod:    "daily",
		RateLimitRPM:    1000,
		TeamID:          "team-A",
		TeamBudgetLimit: 5.0,
		TeamBudgetPeriod: "monthly",
		TeamRateLimitRPM: 1000,
	}

	// VK spend well under VK budget; team spend under team budget -> allow.
	spend.set("vk-team-budget", 1.0)
	spend.setTeam("team-A", 4.0)
	ok, status, reason := engine.Allow(vk, "gpt-4o")
	if !ok || status != 0 || reason != "" {
		t.Fatalf("team under limit: ok=%v status=%d reason=%q, want allow", ok, status, reason)
	}

	// Team spend over team budget (VK still under its own budget) -> deny.
	spend.setTeam("team-A", 6.0)
	ok, status, reason = engine.Allow(vk, "gpt-4o")
	if ok || status != 429 || reason != "team budget exhausted" {
		t.Fatalf("team over limit: ok=%v status=%d reason=%q, want deny 429 team budget exhausted", ok, status, reason)
	}
}

// TestTeamRateLimitExceeded verifies the team RPM tier denies 429 "team rate
// limit exceeded" even when the VK's own RPM passes (D3).
func TestTeamRateLimitExceeded(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:              "vk-team-rpm",
		RateLimitRPM:     1000, // VK RPM generous
		TeamID:           "team-B",
		TeamRateLimitRPM: 2, // team RPM tight
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
	if ok || status != 429 || reason != "team rate limit exceeded" {
		t.Fatalf("third request: ok=%v status=%d reason=%q, want deny 429 team rate limit exceeded", ok, status, reason)
	}
}

// TestTeamTierSkippedWhenUnteamed verifies a VK with empty TeamID is evaluated at
// the VK level only — the Team tier is skipped even if team spend would exceed a
// (zero) team budget (D3/D4).
func TestTeamTierSkippedWhenUnteamed(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:          "vk-unteamed",
		BudgetLimit:  10.0,
		BudgetPeriod: "daily",
		RateLimitRPM: 1000,
		// No TeamID, no team limits.
	}
	spend.set("vk-unteamed", 1.0)
	// Even if some team had spend, an un-teamed VK must never consult it.
	spend.setTeam("", 999.0)
	spend.setTeam("team-X", 999.0)

	ok, status, reason := engine.Allow(vk, "gpt-4o")
	if !ok || status != 0 || reason != "" {
		t.Fatalf("un-teamed VK: ok=%v status=%d reason=%q, want allow", ok, status, reason)
	}
}

// TestVKDenialPrecedesTeam verifies VK-level denial is reported before the Team
// tier is consulted (D3 precedence: VK budget -> VK RPM -> Team budget -> Team RPM).
func TestVKDenialPrecedesTeam(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:              "vk-precedence",
		BudgetLimit:      1.0,
		BudgetPeriod:     "daily",
		TeamID:           "team-C",
		TeamBudgetLimit:  5.0,
		TeamBudgetPeriod: "monthly",
	}
	// Both VK and Team are over budget; the VK reason must win.
	spend.set("vk-precedence", 2.0)
	spend.setTeam("team-C", 99.0)

	ok, status, reason := engine.Allow(vk, "gpt-4o")
	if ok || status != 429 || reason != "budget exhausted" {
		t.Fatalf("precedence: ok=%v status=%d reason=%q, want VK 'budget exhausted' first", ok, status, reason)
	}
}

// TestValidateBudgetOwner verifies the inline single-owner validation (bf-gov-1,
// D2): a budget owner may name at most one of {VirtualKeyID, TeamID}.
func TestValidateBudgetOwner(t *testing.T) {
	if err := ValidateBudgetOwner(BudgetOwner{}); err != nil {
		t.Fatalf("no owner: err=%v, want nil", err)
	}
	if err := ValidateBudgetOwner(BudgetOwner{VirtualKeyID: "vk-1"}); err != nil {
		t.Fatalf("VK-only owner: err=%v, want nil", err)
	}
	if err := ValidateBudgetOwner(BudgetOwner{TeamID: "team-1"}); err != nil {
		t.Fatalf("team-only owner: err=%v, want nil", err)
	}
	if err := ValidateBudgetOwner(BudgetOwner{VirtualKeyID: "vk-1", TeamID: "team-1"}); err == nil {
		t.Fatal("both owners: err=nil, want error")
	}
}
