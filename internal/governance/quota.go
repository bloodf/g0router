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

// Decision is the typed governance outcome of an Evaluate call (bf-gov-3, D8).
// It maps to g0router's {data,error} envelope via a snake_case error.code. The
// Model/Provider/MCP values are declared as the shared enum the bf-gov-2 list
// tier and the bf-mcp tier emit; bf-gov-3 emits only the budget/RPM/token/
// request subset.
type Decision int

const (
	DecisionAllow Decision = iota
	DecisionVirtualKeyNotFound
	DecisionVirtualKeyBlocked
	DecisionRateLimited
	DecisionBudgetExceeded
	DecisionTokenLimited
	DecisionRequestLimited
	DecisionModelBlocked    // emitted by bf-gov-2 list tier (declared, not emitted here)
	DecisionProviderBlocked // declared, not emitted here
	DecisionMCPToolBlocked  // declared, not emitted here
)

// Code returns the snake_case error.code for the decision, surfaced live in the
// gate's {data,error} response (bf-gov-3, D8). DecisionAllow has no code.
func (d Decision) Code() string {
	switch d {
	case DecisionVirtualKeyNotFound:
		return "virtual_key_not_found"
	case DecisionVirtualKeyBlocked:
		return "virtual_key_blocked"
	case DecisionRateLimited:
		return "rate_limited"
	case DecisionBudgetExceeded:
		return "budget_exceeded"
	case DecisionTokenLimited:
		return "token_limited"
	case DecisionRequestLimited:
		return "request_limited"
	case DecisionModelBlocked:
		return "model_blocked"
	case DecisionProviderBlocked:
		return "provider_blocked"
	case DecisionMCPToolBlocked:
		return "mcp_tool_blocked"
	default:
		return ""
	}
}

// EvaluationResult is the typed outcome returned by Evaluate (bf-gov-3, D8).
// Reason is the human-readable denial message (preserved from the shipped Allow
// tuple); Status is the HTTP status (429 for gov-3 denials, 0 on allow). The
// richer Bifrost sub-structs (RateLimitInfo/BudgetInfo/UsageInfo) are
// ESC-REF-ABSENT and not carried.
type EvaluationResult struct {
	Decision Decision
	Reason   string
	Status   int
}

// VirtualKeyInfo is the subset of virtual key state needed by the quota engine.
// The Team* fields carry the optional owning team's budget/RPM for the 2-level
// hierarchy check (bf-gov-1, D3); they are zero/empty for un-teamed keys.
type VirtualKeyInfo struct {
	Key          string
	BudgetLimit  float64
	BudgetPeriod string
	RateLimitRPM int

	// Dual-dimension rate limit (bf-gov-3, D1/D3), each SQL-live over
	// request_log within windowStart(*ResetPeriod). Zero max disables the
	// dimension. *ResetPeriod accepts the calendar words daily/weekly/monthly
	// or a rolling-duration token (1h/1d/1M).
	TokenMax           int64
	TokenResetPeriod   string
	RequestMax         int64
	RequestResetPeriod string

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
// It is a thin, signature-preserving wrapper over Evaluate (bf-gov-3, D8): every
// existing caller keeps the exact (bool,int,string) contract.
func (e *QuotaEngine) Allow(vk *VirtualKeyInfo, model string) (ok bool, status int, reason string) {
	r := e.Evaluate(vk, model)
	return r.Decision == DecisionAllow, r.Status, r.Reason
}

// Evaluate runs the fail-closed governance chain and returns a typed
// EvaluationResult (bf-gov-3, D8). Precedence (D3): VK budget -> VK RPM ->
// VK request-limit -> VK token-limit -> Team budget -> Team RPM. The first
// failing level wins; all denials carry HTTP 429.
func (e *QuotaEngine) Evaluate(vk *VirtualKeyInfo, model string) EvaluationResult {
	if err := e.checkBudget(vk); err != nil {
		return EvaluationResult{Decision: DecisionBudgetExceeded, Reason: err.Error(), Status: 429}
	}
	if err := e.checkRPM(vk); err != nil {
		return EvaluationResult{Decision: DecisionRateLimited, Reason: err.Error(), Status: 429}
	}
	// Dual-dimension rate limit (bf-gov-3, D1/D3), both SQL-live over request_log
	// within the calendar/rolling window. Request runs before token (precedence).
	if err := e.checkRequestLimit(vk); err != nil {
		return EvaluationResult{Decision: DecisionRequestLimited, Reason: err.Error(), Status: 429}
	}
	if err := e.checkTokenLimit(vk); err != nil {
		return EvaluationResult{Decision: DecisionTokenLimited, Reason: err.Error(), Status: 429}
	}
	// 2-level hierarchy (bf-gov-1, D3): when the VK owns a team, the Team budget
	// and RPM must ALSO pass. Un-teamed VKs skip these steps.
	if err := e.checkTeamBudget(vk); err != nil {
		return EvaluationResult{Decision: DecisionBudgetExceeded, Reason: err.Error(), Status: 429}
	}
	if err := e.checkTeamRPM(vk); err != nil {
		return EvaluationResult{Decision: DecisionRateLimited, Reason: err.Error(), Status: 429}
	}
	return EvaluationResult{Decision: DecisionAllow}
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

// checkRequestLimit enforces the VK's request-count dimension via the live
// SumRequestsByAPIKey COUNT over request_log within windowStart(RequestResetPeriod)
// (bf-gov-3, D3). It denies when the count REACHES RequestMax. A non-positive
// RequestMax disables the dimension. Lazy reset is inherent in the windowStart
// lower bound — no in-memory counter, no worker.
func (e *QuotaEngine) checkRequestLimit(vk *VirtualKeyInfo) error {
	if vk.RequestMax <= 0 || vk.RequestResetPeriod == "" {
		return nil
	}
	since := e.windowStart(vk.RequestResetPeriod)
	count, err := e.spend.SumRequestsByAPIKey(vk.Key, since.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("request limit check failed: %w", err)
	}
	if count >= vk.RequestMax {
		return fmt.Errorf("request limit exceeded")
	}
	return nil
}

// checkTokenLimit enforces the VK's token dimension via the live
// SumTokensByAPIKey SUM over request_log within windowStart(TokenResetPeriod)
// (bf-gov-3, D1). It denies when the summed tokens EXCEED TokenMax. A
// non-positive TokenMax disables the dimension. Lazy reset is inherent.
func (e *QuotaEngine) checkTokenLimit(vk *VirtualKeyInfo) error {
	if vk.TokenMax <= 0 || vk.TokenResetPeriod == "" {
		return nil
	}
	since := e.windowStart(vk.TokenResetPeriod)
	used, err := e.spend.SumTokensByAPIKey(vk.Key, since.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("token limit check failed: %w", err)
	}
	if used > vk.TokenMax {
		return fmt.Errorf("token limit exceeded")
	}
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
		// Rolling-duration tokens (bf-gov-3, D2): 1h/1d/1M etc. yield a
		// now.Add(-d) lower bound. An unparseable token falls back to now
		// (effectively an empty window); ValidateRateLimit rejects such tokens
		// at config time so they never reach here in production.
		if start, ok := rollingWindowStart(now, period); ok {
			return start
		}
		return now
	}
}

// rollingWindowStart parses a rolling-duration period token of the form
// "<n><unit>" where unit is h (hours), d (days), or M (calendar months) and
// returns the lower bound now-duration. It returns ok=false for unparseable
// tokens (bf-gov-3, D2). It does NOT accept the calendar words handled by the
// windowStart switch cases.
func rollingWindowStart(now time.Time, period string) (time.Time, bool) {
	if len(period) < 2 {
		return time.Time{}, false
	}
	unit := period[len(period)-1]
	numPart := period[:len(period)-1]
	var n int
	if _, err := fmt.Sscanf(numPart, "%d", &n); err != nil || n <= 0 {
		return time.Time{}, false
	}
	// Reject any non-digit residue (e.g. "1.5h", "x1h") that Sscanf would skip.
	if fmt.Sprintf("%d", n) != numPart {
		return time.Time{}, false
	}
	switch unit {
	case 'h':
		return now.Add(-time.Duration(n) * time.Hour), true
	case 'd':
		return now.AddDate(0, 0, -n), true
	case 'M':
		return now.AddDate(0, -n, 0), true
	default:
		return time.Time{}, false
	}
}
