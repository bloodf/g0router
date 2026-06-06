package governance

import (
	"fmt"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

type fakeGovernanceRepo struct {
	keys         map[int64]*store.VirtualKey
	teams        map[int64]*store.Team
	budgetDeltas map[string]float64
}

func newFakeGovernanceRepo() *fakeGovernanceRepo {
	return &fakeGovernanceRepo{
		keys:         make(map[int64]*store.VirtualKey),
		teams:        make(map[int64]*store.Team),
		budgetDeltas: make(map[string]float64),
	}
}

func (f *fakeGovernanceRepo) GetVirtualKey(id int64) (*store.VirtualKey, error) {
	key, ok := f.keys[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return key, nil
}

func (f *fakeGovernanceRepo) GetTeam(id int64) (*store.Team, error) {
	team, ok := f.teams[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return team, nil
}

func (f *fakeGovernanceRepo) AddVirtualKeyBudgetUsed(id int64, delta float64) error {
	f.budgetDeltas[fmt.Sprintf("key-%d", id)] += delta
	if key, ok := f.keys[id]; ok {
		key.BudgetUsedUSD += delta
	}
	return nil
}

func (f *fakeGovernanceRepo) AddTeamBudgetUsed(id int64, delta float64) error {
	f.budgetDeltas[fmt.Sprintf("team-%d", id)] += delta
	if team, ok := f.teams[id]; ok {
		team.BudgetUsedUSD += delta
	}
	return nil
}

func (f *fakeGovernanceRepo) ResetVirtualKeyBudget(id int64, resetAt time.Time) error {
	if key, ok := f.keys[id]; ok {
		key.BudgetUsedUSD = 0
		key.BudgetResetAt = &resetAt
	}
	return nil
}

func (f *fakeGovernanceRepo) ResetTeamBudget(id int64, resetAt time.Time) error {
	if team, ok := f.teams[id]; ok {
		team.BudgetUsedUSD = 0
		team.BudgetResetAt = &resetAt
	}
	return nil
}

type fakeGovernanceLimiter struct {
	allowedReq  map[string]bool
	allowedTok  map[string]bool
	tokensAdded map[string]int
}

func newFakeGovernanceLimiter() *fakeGovernanceLimiter {
	return &fakeGovernanceLimiter{
		allowedReq:  make(map[string]bool),
		allowedTok:  make(map[string]bool),
		tokensAdded: make(map[string]int),
	}
}

func (f *fakeGovernanceLimiter) AllowRequest(keyID string, rpm *int) bool {
	if rpm == nil || *rpm <= 0 {
		return true
	}
	return f.allowedReq[keyID]
}

func (f *fakeGovernanceLimiter) AllowTokens(keyID string, tpm *int) bool {
	if tpm == nil || *tpm <= 0 {
		return true
	}
	return f.allowedTok[keyID]
}

func (f *fakeGovernanceLimiter) AddTokens(keyID string, tokens int) {
	f.tokensAdded[keyID] += tokens
}

func TestCheckInactiveKeyReturns403(t *testing.T) {
	repo := newFakeGovernanceRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: false}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 403 {
		t.Errorf("status = %d, want 403", res.Status)
	}
}

func TestCheckBudgetExhaustedReturns403(t *testing.T) {
	repo := newFakeGovernanceRepo()
	budget := 10.0
	futureReset := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, BudgetUSD: &budget, BudgetUsedUSD: 10.0, BudgetPeriod: "monthly", BudgetResetAt: &futureReset}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 403 {
		t.Errorf("status = %d, want 403", res.Status)
	}
	if res.Reason != "virtual key budget exhausted" {
		t.Errorf("reason = %q, want budget exhausted", res.Reason)
	}
}

func TestCheckBudgetResetsAfterPeriodRollover(t *testing.T) {
	repo := newFakeGovernanceRepo()
	budget := 10.0
	pastReset := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, BudgetUSD: &budget, BudgetUsedUSD: 10.0, BudgetPeriod: "daily", BudgetResetAt: &pastReset}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if !res.Allowed {
		t.Fatalf("expected allowed after reset, got status %d reason %s", res.Status, res.Reason)
	}
	if repo.keys[1].BudgetUsedUSD != 0 {
		t.Errorf("BudgetUsedUSD = %f, want 0", repo.keys[1].BudgetUsedUSD)
	}
	if repo.keys[1].BudgetResetAt == nil || !repo.keys[1].BudgetResetAt.After(pastReset) {
		t.Error("BudgetResetAt did not roll forward")
	}
}

func TestCheckKeyRPMLimitReturns429(t *testing.T) {
	repo := newFakeGovernanceRepo()
	rpm := 10
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, RateLimitRPM: &rpm}
	lim := newFakeGovernanceLimiter()
	lim.allowedReq["vkey-1"] = false
	g := New(repo, lim)

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 429 {
		t.Errorf("status = %d, want 429", res.Status)
	}
	if res.Reason != "virtual key rate limit exceeded" {
		t.Errorf("reason = %q, want rate limit exceeded", res.Reason)
	}
}

func TestCheckKeyTPMLimitReturns429(t *testing.T) {
	repo := newFakeGovernanceRepo()
	tpm := 1000
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, RateLimitTPM: &tpm}
	lim := newFakeGovernanceLimiter()
	lim.allowedReq["vkey-1"] = true
	lim.allowedTok["vkey-1"] = false
	g := New(repo, lim)

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 429 {
		t.Errorf("status = %d, want 429", res.Status)
	}
	if res.Reason != "virtual key token limit exceeded" {
		t.Errorf("reason = %q, want token limit exceeded", res.Reason)
	}
}

func TestCheckTeamRPMLimitReturns429(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(10)
	teamRPM := 5
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.teams[10] = &store.Team{ID: 10, Name: "eng", RateLimitRPM: &teamRPM}
	lim := newFakeGovernanceLimiter()
	lim.allowedReq["team-10"] = false
	lim.allowedReq["vkey-1"] = true
	g := New(repo, lim)

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 429 {
		t.Errorf("status = %d, want 429", res.Status)
	}
	if res.Reason != "team rate limit exceeded" {
		t.Errorf("reason = %q, want team rate limit exceeded", res.Reason)
	}
}

func TestCheckTeamBudgetExhaustedReturns403(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(10)
	budget := 50.0
	futureReset := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.teams[10] = &store.Team{ID: 10, Name: "eng", BudgetUSD: &budget, BudgetUsedUSD: 50.0, BudgetPeriod: "monthly", BudgetResetAt: &futureReset}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 403 {
		t.Errorf("status = %d, want 403", res.Status)
	}
	if res.Reason != "team budget exhausted" {
		t.Errorf("reason = %q, want team budget exhausted", res.Reason)
	}
}

func TestCheckAllowedReturns200(t *testing.T) {
	repo := newFakeGovernanceRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true}
	lim := newFakeGovernanceLimiter()
	lim.allowedReq["vkey-1"] = true
	lim.allowedTok["vkey-1"] = true
	g := New(repo, lim)

	res := g.Check(repo.keys[1])
	if !res.Allowed {
		t.Fatalf("expected allowed, got status %d reason %s", res.Status, res.Reason)
	}
	if res.KeyID != 1 {
		t.Errorf("KeyID = %d, want 1", res.KeyID)
	}
}

func TestRecordUsageAccumulatesOnKeyAndTeam(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(10)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.teams[10] = &store.Team{ID: 10, Name: "eng"}
	lim := newFakeGovernanceLimiter()
	g := New(repo, lim)

	u := usage.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	if err := g.RecordUsage(1, &teamID, providers.ProviderOpenAI, "gpt-4o", u); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	if repo.budgetDeltas["key-1"] == 0 {
		t.Error("expected key budget delta > 0")
	}
	if repo.budgetDeltas["team-10"] == 0 {
		t.Error("expected team budget delta > 0")
	}
	if repo.budgetDeltas["key-1"] != repo.budgetDeltas["team-10"] {
		t.Errorf("key delta %f != team delta %f", repo.budgetDeltas["key-1"], repo.budgetDeltas["team-10"])
	}
	if lim.tokensAdded["vkey-1"] != 15 {
		t.Errorf("tokens added = %d, want 15", lim.tokensAdded["vkey-1"])
	}
}

func TestRecordUsageWithoutTeamOnlyAccumulatesOnKey(t *testing.T) {
	repo := newFakeGovernanceRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true}
	lim := newFakeGovernanceLimiter()
	g := New(repo, lim)

	u := usage.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	if err := g.RecordUsage(1, nil, providers.ProviderOpenAI, "gpt-4o", u); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	if repo.budgetDeltas["key-1"] == 0 {
		t.Error("expected key budget delta > 0")
	}
	if repo.budgetDeltas["team-10"] != 0 {
		t.Error("expected no team budget delta")
	}
}

func TestNextResetDaily(t *testing.T) {
	now := time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)
	got := nextReset(now, "daily")
	want := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("nextReset(daily) = %v, want %v", got, want)
	}
}

func TestNextResetWeekly(t *testing.T) {
	// June 6 2026 is a Saturday
	now := time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)
	got := nextReset(now, "weekly")
	want := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC) // Sunday
	if !got.Equal(want) {
		t.Errorf("nextReset(weekly) = %v, want %v", got, want)
	}
}

func TestNextResetWeeklyOnSunday(t *testing.T) {
	now := time.Date(2026, 6, 7, 15, 0, 0, 0, time.UTC) // Sunday
	got := nextReset(now, "weekly")
	want := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC) // next Sunday
	if !got.Equal(want) {
		t.Errorf("nextReset(weekly Sunday) = %v, want %v", got, want)
	}
}

func TestNextResetMonthly(t *testing.T) {
	now := time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)
	got := nextReset(now, "monthly")
	want := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("nextReset(monthly) = %v, want %v", got, want)
	}
}

func TestCheckNoBudgetNoLimitAlwaysAllowed(t *testing.T) {
	repo := newFakeGovernanceRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if !res.Allowed {
		t.Fatalf("expected allowed, got %d %s", res.Status, res.Reason)
	}
}

func TestCheckTeamNotFoundReturns403(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(99)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 403 {
		t.Errorf("status = %d, want 403", res.Status)
	}
	if res.Reason != "team not found" {
		t.Errorf("reason = %q, want team not found", res.Reason)
	}
}
