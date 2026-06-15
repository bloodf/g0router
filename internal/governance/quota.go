package governance

import (
	"fmt"
	"sync"
	"time"
)

// SpendReader returns the total cost attributed to a key or team since a
// timestamp, plus the dual-dimension token and request aggregates (bf-gov-3,
// D1/D3) the rate-limit dimensions enforce live over request_log.
type SpendReader interface {
	SumCostByAPIKey(key, sinceISO string) (float64, error)
	SumCostByTeam(teamID, sinceISO string) (float64, error)
	SumTokensByAPIKey(key, sinceISO string) (int64, error)
	SumRequestsByAPIKey(key, sinceISO string) (int64, error)
}

// VirtualKeyInfo is the subset of virtual key state needed by the quota engine.
// The Team* fields carry the optional owning team's budget/RPM for the 2-level
// hierarchy check (bf-gov-1, D3); they are zero/empty for un-teamed keys.
type VirtualKeyInfo struct {
	Key          string
	BudgetLimit  float64
	BudgetPeriod string
	RateLimitRPM int

	TeamID           string
	TeamBudgetLimit  float64
	TeamBudgetPeriod string
	TeamRateLimitRPM int
}

// BudgetOwner names the owner(s) a budget is associated with. A budget may name
// at most one owner among {VirtualKeyID, TeamID} (bf-gov-1, D2). The Customer
// tier is ESC, so it is not represented here.
type BudgetOwner struct {
	VirtualKeyID string
	TeamID       string
}

// ValidateBudgetOwner returns an error when more than one owner among
// {VirtualKeyID, TeamID} is set. It is the inline single-owner validation that
// replaces Bifrost's GORM BeforeSave hook (bf-gov-1, D2).
func ValidateBudgetOwner(owner BudgetOwner) error {
	owners := 0
	if owner.VirtualKeyID != "" {
		owners++
	}
	if owner.TeamID != "" {
		owners++
	}
	if owners > 1 {
		return fmt.Errorf("budget: at most one owner allowed among {virtual_key, team}, got %d", owners)
	}
	return nil
}

// QuotaEngine enforces per-virtual-key budget and RPM limits.
type QuotaEngine struct {
	spend SpendReader
	clock func() time.Time

	mu      sync.Mutex
	rpmHits map[string]*rpmWindow
}

// NewQuotaEngine creates a quota engine with the given spend reader and clock.
func NewQuotaEngine(spend SpendReader, clock func() time.Time) *QuotaEngine {
	return &QuotaEngine{
		spend:   spend,
		clock:   clock,
		rpmHits: map[string]*rpmWindow{},
	}
}

type rpmWindow struct {
	mu       sync.Mutex
	minute   string
	count    int
}

// Allow returns true if the request is within the virtual key's budget and RPM limits.
// On denial it returns false, an HTTP status code (429), and a human-readable reason.
func (e *QuotaEngine) Allow(vk *VirtualKeyInfo, model string) (ok bool, status int, reason string) {
	if err := e.checkBudget(vk); err != nil {
		return false, 429, err.Error()
	}
	if err := e.checkRPM(vk); err != nil {
		return false, 429, err.Error()
	}
	// 2-level hierarchy (bf-gov-1, D3): when the VK owns a team, the Team budget
	// and RPM must ALSO pass. Un-teamed VKs skip these steps.
	if err := e.checkTeamBudget(vk); err != nil {
		return false, 429, err.Error()
	}
	if err := e.checkTeamRPM(vk); err != nil {
		return false, 429, err.Error()
	}
	return true, 0, ""
}

func (e *QuotaEngine) checkBudget(vk *VirtualKeyInfo) error {
	if vk.BudgetLimit <= 0 || vk.BudgetPeriod == "" {
		return nil
	}
	since := e.windowStart(vk.BudgetPeriod)
	spent, err := e.spend.SumCostByAPIKey(vk.Key, since.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("spend check failed: %w", err)
	}
	if spent > vk.BudgetLimit {
		return fmt.Errorf("budget exhausted")
	}
	return nil
}

func (e *QuotaEngine) checkRPM(vk *VirtualKeyInfo) error {
	if vk.RateLimitRPM <= 0 {
		return nil
	}
	now := e.clock()
	minute := now.UTC().Format("2006-01-02T15:04")

	e.mu.Lock()
	w, ok := e.rpmHits[vk.Key]
	if !ok {
		w = &rpmWindow{}
		e.rpmHits[vk.Key] = w
	}
	e.mu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.minute != minute {
		w.minute = minute
		w.count = 0
	}
	if w.count >= vk.RateLimitRPM {
		return fmt.Errorf("rate limit exceeded")
	}
	w.count++
	return nil
}

// checkTeamBudget enforces the owning team's budget via the live SumCostByTeam
// aggregate (bf-gov-1, D8). It is a no-op for un-teamed keys or teams without a
// positive budget. The display-only teams.budget_used_usd column is NOT consulted.
func (e *QuotaEngine) checkTeamBudget(vk *VirtualKeyInfo) error {
	if vk.TeamID == "" || vk.TeamBudgetLimit <= 0 || vk.TeamBudgetPeriod == "" {
		return nil
	}
	since := e.windowStart(vk.TeamBudgetPeriod)
	spent, err := e.spend.SumCostByTeam(vk.TeamID, since.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("team spend check failed: %w", err)
	}
	if spent > vk.TeamBudgetLimit {
		return fmt.Errorf("team budget exhausted")
	}
	return nil
}

// checkTeamRPM enforces the owning team's per-minute rate limit using the shared
// in-memory rpm window keyed by a synthetic "team:<id>" key (bf-gov-1, D3/D8).
// It is a no-op for un-teamed keys or teams without a positive RPM.
func (e *QuotaEngine) checkTeamRPM(vk *VirtualKeyInfo) error {
	if vk.TeamID == "" || vk.TeamRateLimitRPM <= 0 {
		return nil
	}
	now := e.clock()
	minute := now.UTC().Format("2006-01-02T15:04")
	teamKey := "team:" + vk.TeamID

	e.mu.Lock()
	w, ok := e.rpmHits[teamKey]
	if !ok {
		w = &rpmWindow{}
		e.rpmHits[teamKey] = w
	}
	e.mu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.minute != minute {
		w.minute = minute
		w.count = 0
	}
	if w.count >= vk.TeamRateLimitRPM {
		return fmt.Errorf("team rate limit exceeded")
	}
	w.count++
	return nil
}

func (e *QuotaEngine) windowStart(period string) time.Time {
	now := e.clock().UTC()
	switch period {
	case "daily":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case "weekly":
		// Week starts on Monday.
		offset := (int(now.Weekday()) + 6) % 7
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -offset)
	case "monthly":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return now
	}
}
