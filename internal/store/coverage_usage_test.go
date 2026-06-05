package store

import (
	"testing"
	"time"
)

// Error paths via closedStore (DB closed → every operation errors).

func TestCountUsageClosedDBReturnsError(t *testing.T) {
	s := closedStore(t)
	if _, err := s.CountUsage(UsageFilter{}); err == nil {
		t.Fatal("CountUsage on closed DB: want error")
	}
}

func TestDeleteRequestLogsOlderThanClosedDBReturnsError(t *testing.T) {
	s := closedStore(t)
	if _, err := s.DeleteRequestLogsOlderThan(time.Now()); err == nil {
		t.Fatal("DeleteRequestLogsOlderThan on closed DB: want error")
	}
}

// clampUsageLimit: 0->default, >200->200, negative->default, in-range stays.
func TestClampUsageLimitZeroReturnsDefault(t *testing.T) {
	if got := clampUsageLimit(0); got != defaultUsageLimit {
		t.Fatalf("clampUsageLimit(0) = %d, want %d", got, defaultUsageLimit)
	}
}

func TestClampUsageLimitNegativeReturnsDefault(t *testing.T) {
	if got := clampUsageLimit(-5); got != defaultUsageLimit {
		t.Fatalf("clampUsageLimit(-5) = %d, want %d", got, defaultUsageLimit)
	}
}

func TestClampUsageLimitAboveMaxClampsToMax(t *testing.T) {
	if got := clampUsageLimit(maxUsageLimit + 1); got != maxUsageLimit {
		t.Fatalf("clampUsageLimit(%d) = %d, want %d", maxUsageLimit+1, got, maxUsageLimit)
	}
}

func TestClampUsageLimitExactMaxAllowed(t *testing.T) {
	if got := clampUsageLimit(maxUsageLimit); got != maxUsageLimit {
		t.Fatalf("clampUsageLimit(%d) = %d, want %d", maxUsageLimit, got, maxUsageLimit)
	}
}

func TestClampUsageLimitInRangePassesThrough(t *testing.T) {
	if got := clampUsageLimit(10); got != 10 {
		t.Fatalf("clampUsageLimit(10) = %d, want 10", got)
	}
}

// CountUsage: with and without filters.
func TestCountUsageWithFilter(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("req-a", "openai", "gpt-4o", base),
		minimalUsageEntry("req-b", "openai", "gpt-4o", base.Add(time.Minute)),
		minimalUsageEntry("req-c", "anthropic", "claude-sonnet-4", base.Add(2*time.Minute)),
	})

	count, err := s.CountUsage(UsageFilter{Provider: stringPtr("openai")})
	if err != nil {
		t.Fatalf("CountUsage: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestCountUsageEmpty(t *testing.T) {
	s := openTestStore(t)

	count, err := s.CountUsage(UsageFilter{})
	if err != nil {
		t.Fatalf("CountUsage empty: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestCountUsageWithStatusClassFilter(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	ok := minimalUsageEntry("ok", "openai", "gpt-4o", base)
	ok.StatusCode = intPtr(200)
	bad := minimalUsageEntry("bad", "openai", "gpt-4o", base.Add(time.Minute))
	bad.StatusCode = intPtr(500)
	logUsageEntries(t, s, []RequestLogEntry{ok, bad})

	count, err := s.CountUsage(UsageFilter{StatusClass: StatusClassSuccess})
	if err != nil {
		t.Fatalf("CountUsage success: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

// DeleteRequestLogsOlderThan: zero rows (nothing to delete), cutoff boundary.
func TestDeleteRequestLogsZeroRows(t *testing.T) {
	s := openTestStore(t)

	deleted, err := s.DeleteRequestLogsOlderThan(time.Now().UTC())
	if err != nil {
		t.Fatalf("DeleteRequestLogsOlderThan empty: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

func TestDeleteRequestLogsNoneOlderThanCutoff(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("new-1", "openai", "gpt-4o", now.Add(-1*time.Hour)),
	})

	// cutoff is before all entries — nothing should be deleted
	cutoff := now.Add(-2 * time.Hour)
	deleted, err := s.DeleteRequestLogsOlderThan(cutoff)
	if err != nil {
		t.Fatalf("DeleteRequestLogsOlderThan: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

func TestDeleteRequestLogsCutoffBoundaryExclusive(t *testing.T) {
	s := openTestStore(t)
	ts := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("exact", "openai", "gpt-4o", ts),
	})

	// cutoff == timestamp: "strictly older than" — should NOT delete it
	deleted, err := s.DeleteRequestLogsOlderThan(ts)
	if err != nil {
		t.Fatalf("DeleteRequestLogsOlderThan boundary: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0 (timestamp == cutoff is not strictly older)", deleted)
	}
}

// scanRequestLogEntry: corrupt timestamp triggers parse error.
func TestGetUsageCorruptTimestampReturnsError(t *testing.T) {
	s := openTestStore(t)
	base := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("req-ts", "openai", "gpt-4o", base),
	})
	// Corrupt the timestamp so time.Parse fails.
	if _, err := s.db.Exec("UPDATE request_log SET timestamp = ? WHERE request_id = ?", "NOT-A-DATE", "req-ts"); err != nil {
		t.Fatalf("corrupt timestamp: %v", err)
	}
	if _, err := s.GetUsage(UsageFilter{}); err == nil {
		t.Fatal("GetUsage with corrupt timestamp: want error")
	}
}

func TestDeleteRequestLogsDeletesAllOlderRows(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	logUsageEntries(t, s, []RequestLogEntry{
		minimalUsageEntry("old-a", "openai", "gpt-4o", now.Add(-20*24*time.Hour)),
		minimalUsageEntry("old-b", "openai", "gpt-4o", now.Add(-15*24*time.Hour)),
		minimalUsageEntry("old-c", "openai", "gpt-4o", now.Add(-8*24*time.Hour)),
	})

	cutoff := now.Add(-7 * 24 * time.Hour)
	deleted, err := s.DeleteRequestLogsOlderThan(cutoff)
	if err != nil {
		t.Fatalf("DeleteRequestLogsOlderThan: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted = %d, want 3", deleted)
	}
}
