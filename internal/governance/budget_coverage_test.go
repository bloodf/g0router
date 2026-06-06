package governance

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

type fakeBudgetRepo struct {
	keys             map[int64]*store.VirtualKey
	teams            map[int64]*store.Team
	resetKeyErr      error
	resetTeamErr     error
	budgetDeltas     map[string]float64
}

func newFakeBudgetRepo() *fakeBudgetRepo {
	return &fakeBudgetRepo{
		keys:         make(map[int64]*store.VirtualKey),
		teams:        make(map[int64]*store.Team),
		budgetDeltas: make(map[string]float64),
	}
}

func (f *fakeBudgetRepo) GetVirtualKey(id int64) (*store.VirtualKey, error) {
	key, ok := f.keys[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return key, nil
}

func (f *fakeBudgetRepo) GetTeam(id int64) (*store.Team, error) {
	team, ok := f.teams[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return team, nil
}

func (f *fakeBudgetRepo) AddVirtualKeyBudgetUsed(id int64, delta float64) error {
	f.budgetDeltas["key-"+string(rune(id))] += delta
	return nil
}

func (f *fakeBudgetRepo) AddTeamBudgetUsed(id int64, delta float64) error {
	f.budgetDeltas["team-"+string(rune(id))] += delta
	return nil
}

func (f *fakeBudgetRepo) ResetVirtualKeyBudget(id int64, resetAt time.Time) error {
	return f.resetKeyErr
}

func (f *fakeBudgetRepo) ResetTeamBudget(id int64, resetAt time.Time) error {
	return f.resetTeamErr
}

func TestLazyResetKeyNilBudget(t *testing.T) {
	repo := newFakeBudgetRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetKey(repo.keys[1], time.Now())
	if err != nil {
		t.Fatalf("expected nil for nil budget, got %v", err)
	}
}

func TestLazyResetKeyZeroBudget(t *testing.T) {
	repo := newFakeBudgetRepo()
	budget := 0.0
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, BudgetUSD: &budget}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetKey(repo.keys[1], time.Now())
	if err != nil {
		t.Fatalf("expected nil for zero budget, got %v", err)
	}
}

func TestLazyResetKeyResetError(t *testing.T) {
	repo := newFakeBudgetRepo()
	repo.resetKeyErr = errors.New("reset error")
	budget := 10.0
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, BudgetUSD: &budget, BudgetPeriod: "monthly"}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetKey(repo.keys[1], time.Now())
	if err == nil {
		t.Fatal("expected error from reset")
	}
}

func TestLazyResetKeyRolloverError(t *testing.T) {
	repo := newFakeBudgetRepo()
	repo.resetKeyErr = errors.New("reset error")
	budget := 10.0
	pastReset := time.Now().UTC().Add(-24 * time.Hour)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, BudgetUSD: &budget, BudgetPeriod: "daily", BudgetResetAt: &pastReset}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetKey(repo.keys[1], time.Now())
	if err == nil {
		t.Fatal("expected error from reset")
	}
}

func TestLazyResetTeamNilBudget(t *testing.T) {
	repo := newFakeBudgetRepo()
	repo.teams[1] = &store.Team{ID: 1, Name: "eng"}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetTeam(repo.teams[1], time.Now())
	if err != nil {
		t.Fatalf("expected nil for nil budget, got %v", err)
	}
}

func TestLazyResetTeamZeroBudget(t *testing.T) {
	repo := newFakeBudgetRepo()
	budget := 0.0
	repo.teams[1] = &store.Team{ID: 1, Name: "eng", BudgetUSD: &budget}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetTeam(repo.teams[1], time.Now())
	if err != nil {
		t.Fatalf("expected nil for zero budget, got %v", err)
	}
}

func TestLazyResetTeamResetError(t *testing.T) {
	repo := newFakeBudgetRepo()
	repo.resetTeamErr = errors.New("reset error")
	budget := 10.0
	repo.teams[1] = &store.Team{ID: 1, Name: "eng", BudgetUSD: &budget, BudgetPeriod: "monthly"}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetTeam(repo.teams[1], time.Now())
	if err == nil {
		t.Fatal("expected error from reset")
	}
}

func TestLazyResetTeamRolloverError(t *testing.T) {
	repo := newFakeBudgetRepo()
	repo.resetTeamErr = errors.New("reset error")
	budget := 10.0
	pastReset := time.Now().UTC().Add(-24 * time.Hour)
	repo.teams[1] = &store.Team{ID: 1, Name: "eng", BudgetUSD: &budget, BudgetPeriod: "daily", BudgetResetAt: &pastReset}
	g := New(repo, newFakeGovernanceLimiter())

	err := g.lazyResetTeam(repo.teams[1], time.Now())
	if err == nil {
		t.Fatal("expected error from reset")
	}
}
