package governance

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

type fakeRecordUsageRepo struct {
	keys         map[int64]*store.VirtualKey
	addKeyErr    error
	addTeamErr   error
}

func newFakeRecordUsageRepo() *fakeRecordUsageRepo {
	return &fakeRecordUsageRepo{
		keys: make(map[int64]*store.VirtualKey),
	}
}

func (f *fakeRecordUsageRepo) GetVirtualKey(id int64) (*store.VirtualKey, error) {
	key, ok := f.keys[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return key, nil
}

func (f *fakeRecordUsageRepo) GetTeam(id int64) (*store.Team, error) {
	return nil, errors.New("not found")
}

func (f *fakeRecordUsageRepo) AddVirtualKeyBudgetUsed(id int64, delta float64) error {
	return f.addKeyErr
}

func (f *fakeRecordUsageRepo) AddTeamBudgetUsed(id int64, delta float64) error {
	return f.addTeamErr
}

func (f *fakeRecordUsageRepo) ResetVirtualKeyBudget(id int64, resetAt time.Time) error {
	return nil
}

func (f *fakeRecordUsageRepo) ResetTeamBudget(id int64, resetAt time.Time) error {
	return nil
}

func TestRecordUsageAddKeyError(t *testing.T) {
	repo := newFakeRecordUsageRepo()
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true}
	repo.addKeyErr = errors.New("db error")
	g := New(repo, newFakeGovernanceLimiter())

	u := usage.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	err := g.RecordUsage(1, nil, providers.ProviderOpenAI, "gpt-4o", u)
	if err == nil {
		t.Fatal("expected error from AddVirtualKeyBudgetUsed")
	}
}

func TestRecordUsageAddTeamError(t *testing.T) {
	repo := newFakeRecordUsageRepo()
	teamID := int64(10)
	repo.keys[1] = &store.VirtualKey{ID: 1, Name: "k1", IsActive: true, TeamID: &teamID}
	repo.addTeamErr = errors.New("db error")
	g := New(repo, newFakeGovernanceLimiter())

	u := usage.Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	err := g.RecordUsage(1, &teamID, providers.ProviderOpenAI, "gpt-4o", u)
	if err == nil {
		t.Fatal("expected error from AddTeamBudgetUsed")
	}
}

func TestNextResetUnknownPeriod(t *testing.T) {
	now := time.Date(2026, 6, 6, 15, 0, 0, 0, time.UTC)
	got := nextReset(now, "yearly")
	want := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) // defaults to monthly
	if !got.Equal(want) {
		t.Errorf("nextReset(unknown) = %v, want %v", got, want)
	}
}
