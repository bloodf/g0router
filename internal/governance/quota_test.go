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

// TestEvaluateAllowAndWrapperEquivalence verifies the Decision enum + Evaluate
// sibling (bf-gov-3, D8): Evaluate returns DecisionAllow with status 0 on pass,
// and the re-expressed Allow wrapper returns the IDENTICAL (ok,status,reason)
// tuple for the shipped budget/RPM/team cases.
func TestEvaluateAllowAndWrapperEquivalence(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)

	t.Run("allow returns DecisionAllow", func(t *testing.T) {
		spend := newFakeSpendReader()
		engine := NewQuotaEngine(spend, fixedClock(now))
		vk := &VirtualKeyInfo{Key: "vk-eval-ok", BudgetLimit: 1.0, BudgetPeriod: "daily", RateLimitRPM: 1000}
		spend.set("vk-eval-ok", 0.5)

		res := engine.Evaluate(vk, "gpt-4o")
		if res.Decision != DecisionAllow {
			t.Fatalf("Evaluate decision = %v, want DecisionAllow", res.Decision)
		}
		if res.Status != 0 || res.Reason != "" {
			t.Fatalf("Evaluate status=%d reason=%q, want 0/empty", res.Status, res.Reason)
		}
		ok, status, reason := engine.Allow(vk, "gpt-4o")
		if !ok || status != 0 || reason != "" {
			t.Fatalf("Allow wrapper = %v/%d/%q, want true/0/empty", ok, status, reason)
		}
	})

	t.Run("budget denial maps to DecisionBudgetExceeded and wrapper tuple", func(t *testing.T) {
		spend := newFakeSpendReader()
		engine := NewQuotaEngine(spend, fixedClock(now))
		vk := &VirtualKeyInfo{Key: "vk-eval-budget", BudgetLimit: 1.0, BudgetPeriod: "daily"}
		spend.set("vk-eval-budget", 2.0)

		res := engine.Evaluate(vk, "gpt-4o")
		if res.Decision != DecisionBudgetExceeded || res.Status != 429 || res.Reason != "budget exhausted" {
			t.Fatalf("Evaluate budget = %v/%d/%q, want BudgetExceeded/429/budget exhausted", res.Decision, res.Status, res.Reason)
		}
		ok, status, reason := engine.Allow(vk, "gpt-4o")
		if ok || status != 429 || reason != "budget exhausted" {
			t.Fatalf("Allow wrapper budget = %v/%d/%q, want false/429/budget exhausted", ok, status, reason)
		}
	})
}

// TestVKTokenLimit verifies the SQL-live token dimension (bf-gov-3, D1/D3): a VK
// is DENIED 429 DecisionTokenLimited when its live SumTokensByAPIKey over the
// window exceeds TokenMax, even though budget + RPM + request all pass. After a
// window rollover (the fake's token map cleared — the analogue of the windowStart
// lower bound excluding prior rows) it is re-allowed: the reset is INHERENT, with
// no counter and no worker.
func TestVKTokenLimit(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:             "vk-token",
		BudgetLimit:     1000, // generous, passes
		BudgetPeriod:    "daily",
		RateLimitRPM:    1000, // generous, passes
		TokenMax:        100,
		TokenResetPeriod: "daily",
	}

	// Under the token limit -> allow.
	spend.setTokens("vk-token", 50)
	if res := engine.Evaluate(vk, "gpt-4o"); res.Decision != DecisionAllow {
		t.Fatalf("under token limit: decision=%v, want DecisionAllow", res.Decision)
	}

	// At the token limit (100) still allowed (deny only when it EXCEEDS).
	spend.setTokens("vk-token", 100)
	if res := engine.Evaluate(vk, "gpt-4o"); res.Decision != DecisionAllow {
		t.Fatalf("at token limit: decision=%v, want DecisionAllow", res.Decision)
	}

	// Over the token limit -> deny 429 TokenLimited.
	spend.setTokens("vk-token", 101)
	res := engine.Evaluate(vk, "gpt-4o")
	if res.Decision != DecisionTokenLimited || res.Status != 429 || res.Reason == "" {
		t.Fatalf("over token limit: decision=%v status=%d reason=%q, want TokenLimited/429", res.Decision, res.Status, res.Reason)
	}

	// Window rollover: clearing the token map (prior rows fall out of window) re-allows.
	spend.setTokens("vk-token", 0)
	if res := engine.Evaluate(vk, "gpt-4o"); res.Decision != DecisionAllow {
		t.Fatalf("after rollover: decision=%v, want DecisionAllow (inherent reset)", res.Decision)
	}
}

// TestVKRequestLimit verifies the SQL-live request dimension (bf-gov-3, D3): a VK
// is DENIED 429 DecisionRequestLimited when its live SumRequestsByAPIKey (COUNT)
// over the window REACHES RequestMax.
func TestVKRequestLimit(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:                "vk-request",
		BudgetLimit:        1000,
		BudgetPeriod:       "daily",
		RateLimitRPM:       1000,
		RequestMax:         5,
		RequestResetPeriod: "1h",
	}

	// Under the request limit -> allow.
	spend.setRequests("vk-request", 4)
	if res := engine.Evaluate(vk, "gpt-4o"); res.Decision != DecisionAllow {
		t.Fatalf("under request limit: decision=%v, want DecisionAllow", res.Decision)
	}

	// At the request limit (count >= RequestMax) -> deny.
	spend.setRequests("vk-request", 5)
	res := engine.Evaluate(vk, "gpt-4o")
	if res.Decision != DecisionRequestLimited || res.Status != 429 || res.Reason == "" {
		t.Fatalf("at request limit: decision=%v status=%d reason=%q, want RequestLimited/429", res.Decision, res.Status, res.Reason)
	}

	// Rollover re-allows.
	spend.setRequests("vk-request", 0)
	if res := engine.Evaluate(vk, "gpt-4o"); res.Decision != DecisionAllow {
		t.Fatalf("after rollover: decision=%v, want DecisionAllow", res.Decision)
	}
}

// TestRateLimitPrecedence verifies the deterministic fail-closed order (D3):
// budget -> RPM -> request-limit -> token-limit -> team. When request AND token
// limits would both deny, request wins (it runs first).
func TestRateLimitPrecedence(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	spend := newFakeSpendReader()
	engine := NewQuotaEngine(spend, fixedClock(now))

	vk := &VirtualKeyInfo{
		Key:                "vk-prec",
		RequestMax:         5,
		RequestResetPeriod: "daily",
		TokenMax:           100,
		TokenResetPeriod:   "daily",
	}
	// Both dimensions over limit; request runs first, so RequestLimited wins.
	spend.setRequests("vk-prec", 5)
	spend.setTokens("vk-prec", 999)
	res := engine.Evaluate(vk, "gpt-4o")
	if res.Decision != DecisionRequestLimited {
		t.Fatalf("precedence: decision=%v, want DecisionRequestLimited (request before token)", res.Decision)
	}
}

// TestWindowStartDurationParsing verifies windowStart's additive default-branch
// rolling-duration parsing (bf-gov-3, D2): the shipped daily/weekly/monthly cases
// are UNCHANGED; rolling tokens (1h/1d/1M) yield a now.Add(-d) lower bound.
func TestWindowStartDurationParsing(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	engine := NewQuotaEngine(newFakeSpendReader(), fixedClock(now))

	// Shipped calendar cases unchanged.
	if got := engine.windowStart("daily"); !got.Equal(time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("windowStart(daily) = %v, want midnight UTC", got)
	}
	if got := engine.windowStart("monthly"); !got.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("windowStart(monthly) = %v, want month start", got)
	}

	// Rolling durations: now minus the parsed duration.
	if got := engine.windowStart("1h"); !got.Equal(now.Add(-time.Hour)) {
		t.Errorf("windowStart(1h) = %v, want now-1h (%v)", got, now.Add(-time.Hour))
	}
	if got := engine.windowStart("1d"); !got.Equal(now.Add(-24 * time.Hour)) {
		t.Errorf("windowStart(1d) = %v, want now-24h", got)
	}
	if got := engine.windowStart("1M"); !got.Equal(now.AddDate(0, -1, 0)) {
		t.Errorf("windowStart(1M) = %v, want now-1month", got)
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
