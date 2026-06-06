package governance

import (
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func (g *Governance) lazyResetKey(key *store.VirtualKey, now time.Time) error {
	if key.BudgetUSD == nil || *key.BudgetUSD <= 0 {
		return nil
	}
	if key.BudgetResetAt == nil || key.BudgetResetAt.IsZero() {
		next := nextReset(now, key.BudgetPeriod)
		if err := g.repo.ResetVirtualKeyBudget(key.ID, next); err != nil {
			return err
		}
		key.BudgetUsedUSD = 0
		key.BudgetResetAt = &next
		return nil
	}
	if now.After(*key.BudgetResetAt) {
		next := nextReset(now, key.BudgetPeriod)
		if err := g.repo.ResetVirtualKeyBudget(key.ID, next); err != nil {
			return err
		}
		key.BudgetUsedUSD = 0
		key.BudgetResetAt = &next
		return nil
	}
	return nil
}

func (g *Governance) lazyResetTeam(team *store.Team, now time.Time) error {
	if team.BudgetUSD == nil || *team.BudgetUSD <= 0 {
		return nil
	}
	if team.BudgetResetAt == nil || team.BudgetResetAt.IsZero() {
		next := nextReset(now, team.BudgetPeriod)
		if err := g.repo.ResetTeamBudget(team.ID, next); err != nil {
			return err
		}
		team.BudgetUsedUSD = 0
		team.BudgetResetAt = &next
		return nil
	}
	if now.After(*team.BudgetResetAt) {
		next := nextReset(now, team.BudgetPeriod)
		if err := g.repo.ResetTeamBudget(team.ID, next); err != nil {
			return err
		}
		team.BudgetUsedUSD = 0
		team.BudgetResetAt = &next
		return nil
	}
	return nil
}

func nextReset(now time.Time, period string) time.Time {
	now = now.UTC()
	switch period {
	case "daily":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
	case "weekly":
		weekday := int(now.Weekday())
		daysUntilSunday := (7 - weekday) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, daysUntilSunday)
	case "monthly":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	default:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	}
}
