package governance

import (
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

// Repository is the narrow persistence interface the governance domain needs.
type Repository interface {
	GetVirtualKey(id int64) (*store.VirtualKey, error)
	GetTeam(id int64) (*store.Team, error)
	AddVirtualKeyBudgetUsed(id int64, delta float64) error
	AddTeamBudgetUsed(id int64, delta float64) error
	ResetVirtualKeyBudget(id int64, resetAt time.Time) error
	ResetTeamBudget(id int64, resetAt time.Time) error
}

// RateLimiter is the in-memory limiter interface for RPM/TPM enforcement.
type RateLimiter interface {
	AllowRequest(keyID string, rpm *int) bool
	AllowTokens(keyID string, tpm *int) bool
	AddTokens(keyID string, tokens int)
}

// CheckResult is the outcome of a pre-request governance check.
type CheckResult struct {
	Allowed bool
	Status  int
	Reason  string
	KeyID   int64
	TeamID  *int64
}

// Governance enforces hierarchical budgets and rate limits for virtual keys.
type Governance struct {
	repo    Repository
	limiter RateLimiter
	clock   func() time.Time
}

// New returns a Governance using the real wall clock.
func New(repo Repository, limiter RateLimiter) *Governance {
	return NewWithClock(repo, limiter, time.Now)
}

// NewWithClock returns a Governance driven by the supplied clock.
func NewWithClock(repo Repository, limiter RateLimiter, clock func() time.Time) *Governance {
	return &Governance{repo: repo, limiter: limiter, clock: clock}
}

// Check performs a pre-request governance evaluation for a virtual key.
// It applies lazy budget reset, then checks active status, budget, and rate limits
// hierarchically (key limits and team limits must both pass).
func (g *Governance) Check(key *store.VirtualKey) CheckResult {
	now := g.clock().UTC().Truncate(time.Second)

	if err := g.lazyResetKey(key, now); err != nil {
		return CheckResult{Allowed: false, Status: 500, Reason: "budget reset error"}
	}

	if !key.IsActive {
		return CheckResult{Allowed: false, Status: 403, Reason: "virtual key inactive"}
	}

	if key.BudgetUSD != nil && *key.BudgetUSD > 0 && key.BudgetUsedUSD >= *key.BudgetUSD {
		return CheckResult{Allowed: false, Status: 403, Reason: "virtual key budget exhausted"}
	}

	var team *store.Team
	if key.TeamID != nil {
		t, err := g.repo.GetTeam(*key.TeamID)
		if err != nil {
			return CheckResult{Allowed: false, Status: 403, Reason: "team not found"}
		}
		team = t

		if err := g.lazyResetTeam(team, now); err != nil {
			return CheckResult{Allowed: false, Status: 500, Reason: "team budget reset error"}
		}

		if team.BudgetUSD != nil && *team.BudgetUSD > 0 && team.BudgetUsedUSD >= *team.BudgetUSD {
			return CheckResult{Allowed: false, Status: 403, Reason: "team budget exhausted"}
		}
	}

	keyLimiterID := limiterIDForKey(key.ID)

	// Check team RPM before key RPM so a team failure does not consume a key slot.
	if team != nil {
		teamLimiterID := limiterIDForTeam(team.ID)
		if !g.limiter.AllowRequest(teamLimiterID, team.RateLimitRPM) {
			return CheckResult{Allowed: false, Status: 429, Reason: "team rate limit exceeded"}
		}
	}

	if !g.limiter.AllowRequest(keyLimiterID, key.RateLimitRPM) {
		return CheckResult{Allowed: false, Status: 429, Reason: "virtual key rate limit exceeded"}
	}

	if !g.limiter.AllowTokens(keyLimiterID, key.RateLimitTPM) {
		return CheckResult{Allowed: false, Status: 429, Reason: "virtual key token limit exceeded"}
	}

	return CheckResult{Allowed: true, Status: 200, KeyID: key.ID, TeamID: key.TeamID}
}

// RecordUsage accumulates budget_used_usd on the virtual key and its team (if any)
// from computed request cost, and records tokens for TPM tracking.
func (g *Governance) RecordUsage(keyID int64, teamID *int64, provider providers.ModelProvider, model string, u usage.Usage) error {
	cost, err := usage.CalculateCostUSDWithOverrides(modelcatalog.NewCatalog(), nil, provider, model, &u)
	if err != nil {
		cost = 0
	}

	keyLimiterID := limiterIDForKey(keyID)
	g.limiter.AddTokens(keyLimiterID, u.TotalTokens)

	if cost > 0 {
		if err := g.repo.AddVirtualKeyBudgetUsed(keyID, cost); err != nil {
			return fmt.Errorf("add virtual key budget used: %w", err)
		}
		if teamID != nil {
			if err := g.repo.AddTeamBudgetUsed(*teamID, cost); err != nil {
				return fmt.Errorf("add team budget used: %w", err)
			}
		}
	}

	return nil
}

func limiterIDForKey(keyID int64) string {
	return fmt.Sprintf("vkey-%d", keyID)
}

func limiterIDForTeam(teamID int64) string {
	return fmt.Sprintf("team-%d", teamID)
}
