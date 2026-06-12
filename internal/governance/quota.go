package governance

import (
	"fmt"
	"sync"
	"time"
)

// SpendReader returns the total cost attributed to a key since a timestamp.
type SpendReader interface {
	SumCostByAPIKey(key, sinceISO string) (float64, error)
}

// VirtualKeyInfo is the subset of virtual key state needed by the quota engine.
type VirtualKeyInfo struct {
	Key          string
	BudgetLimit  float64
	BudgetPeriod string
	RateLimitRPM int
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
