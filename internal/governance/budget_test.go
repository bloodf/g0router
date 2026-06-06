package governance

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

type errorRepo struct {
	*fakeGovernanceRepo
	resetKeyErr   error
	resetTeamErr  error
	getTeamErr    error
	getKeyErr     error
}

func (e *errorRepo) ResetVirtualKeyBudget(id int64, resetAt time.Time) error {
	if e.resetKeyErr != nil {
		return e.resetKeyErr
	}
	return e.fakeGovernanceRepo.ResetVirtualKeyBudget(id, resetAt)
}

func (e *errorRepo) ResetTeamBudget(id int64, resetAt time.Time) error {
	if e.resetTeamErr != nil {
		return e.resetTeamErr
	}
	return e.fakeGovernanceRepo.ResetTeamBudget(id, resetAt)
}

func (e *errorRepo) GetTeam(id int64) (*store.Team, error) {
	if e.getTeamErr != nil {
		return nil, e.getTeamErr
	}
	return e.fakeGovernanceRepo.GetTeam(id)
}

func (e *errorRepo) GetVirtualKey(id int64) (*store.VirtualKey, error) {
	if e.getKeyErr != nil {
		return nil, e.getKeyErr
	}
	return e.fakeGovernanceRepo.GetVirtualKey(id)
}

func TestLazyResetKeyReturnsErrorOnResetFailure(t *testing.T) {
	repo := newFakeGovernanceRepo()
	budget := 10.0
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, BudgetUSD: &budget, BudgetUsedUSD: 5.0, BudgetPeriod: "monthly"}
	errRepo := &errorRepo{fakeGovernanceRepo: repo, resetKeyErr: errors.New("db error")}
	g := New(errRepo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 500 {
		t.Errorf("status = %d, want 500", res.Status)
	}
	if res.Reason != "budget reset error" {
		t.Errorf("reason = %q, want budget reset error", res.Reason)
	}
}

func TestLazyResetKeyNoBudget(t *testing.T) {
	repo := newFakeGovernanceRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if !res.Allowed {
		t.Fatalf("expected allowed, got %d %s", res.Status, res.Reason)
	}
}

func TestLazyResetTeamReturnsErrorOnResetFailure(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(10)
	budget := 50.0
	pastReset := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.teams[10] = &store.Team{ID: 10, Name: "eng", BudgetUSD: &budget, BudgetUsedUSD: 50.0, BudgetPeriod: "daily", BudgetResetAt: &pastReset}
	errRepo := &errorRepo{fakeGovernanceRepo: repo, resetTeamErr: errors.New("db error")}
	g := New(errRepo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if res.Allowed {
		t.Fatal("expected not allowed")
	}
	if res.Status != 500 {
		t.Errorf("status = %d, want 500", res.Status)
	}
}

func TestLazyResetTeamNoBudget(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(10)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.teams[10] = &store.Team{ID: 10, Name: "eng"}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if !res.Allowed {
		t.Fatalf("expected allowed, got %d %s", res.Status, res.Reason)
	}
}

func TestCheckTeamBudgetResetsAfterPeriodRollover(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(10)
	budget := 50.0
	pastReset := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.teams[10] = &store.Team{ID: 10, Name: "eng", BudgetUSD: &budget, BudgetUsedUSD: 50.0, BudgetPeriod: "daily", BudgetResetAt: &pastReset}
	g := New(repo, newFakeGovernanceLimiter())

	res := g.Check(repo.keys[1])
	if !res.Allowed {
		t.Fatalf("expected allowed after reset, got status %d reason %s", res.Status, res.Reason)
	}
	if repo.teams[10].BudgetUsedUSD != 0 {
		t.Errorf("BudgetUsedUSD = %f, want 0", repo.teams[10].BudgetUsedUSD)
	}
	if repo.teams[10].BudgetResetAt == nil || !repo.teams[10].BudgetResetAt.After(pastReset) {
		t.Error("BudgetResetAt did not roll forward")
	}
}

func TestRecordUsageAddKeyBudgetError(t *testing.T) {
	repo := newFakeGovernanceRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true}
	errRepo := &errorRepo{fakeGovernanceRepo: repo}

	// Use a store that returns error for AddVirtualKeyBudgetUsed
	badRepo := &badBudgetRepo{errorRepo: errRepo}
	g2 := New(badRepo, newFakeGovernanceLimiter())

	u := usage.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	if err := g2.RecordUsage(1, nil, "openai", "gpt-4o", u); err == nil {
		t.Fatal("expected error")
	}
}

type badBudgetRepo struct {
	*errorRepo
}

func (b *badBudgetRepo) AddVirtualKeyBudgetUsed(id int64, delta float64) error {
	return errors.New("budget write failed")
}

func (b *badBudgetRepo) AddTeamBudgetUsed(id int64, delta float64) error {
	return errors.New("team budget write failed")
}

func TestRecordUsageAddTeamBudgetError(t *testing.T) {
	repo := newFakeGovernanceRepo()
	teamID := int64(10)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.teams[10] = &store.Team{ID: 10, Name: "eng"}
	badRepo := &badBudgetRepo{errorRepo: &errorRepo{fakeGovernanceRepo: repo}}
	g := New(badRepo, newFakeGovernanceLimiter())

	u := usage.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	if err := g.RecordUsage(1, &teamID, "openai", "gpt-4o", u); err == nil {
		t.Fatal("expected error")
	}
}
