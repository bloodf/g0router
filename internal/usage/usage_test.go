package usage_test

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/usage"
)

type fakeUsageReader struct {
	logs       []usage.UsageLog
	total      int
	getErr     error
	countErr   error
	summary    *usage.UsageSummary
	summaryErr error
}

func (f *fakeUsageReader) GetUsage(filter usage.UsageFilter) ([]usage.UsageLog, error) {
	return f.logs, f.getErr
}

func (f *fakeUsageReader) CountUsage(filter usage.UsageFilter) (int, error) {
	return f.total, f.countErr
}

func (f *fakeUsageReader) GetUsageSummary(filter usage.UsageFilter) (*usage.UsageSummary, error) {
	return f.summary, f.summaryErr
}

func TestListUsageReturnsMappedEntriesAndTotal(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	reader := &fakeUsageReader{
		logs: []usage.UsageLog{
			{
				ID:          1,
				RequestID:   "req-1",
				Timestamp:   now,
				Provider:    "openai",
				Model:       "gpt-4o",
				AuthType:    "api_key",
				TotalTokens: intPtr(10),
			},
			{
				ID:        2,
				RequestID: "req-2",
				Timestamp: now.Add(time.Minute),
				Provider:  "anthropic",
				Model:     "claude",
				AuthType:  "api_key",
			},
		},
		total: 42,
	}

	logs, total, err := usage.ListUsage(reader, usage.UsageFilter{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 42 {
		t.Fatalf("total = %d, want 42", total)
	}
	if len(logs) != 2 {
		t.Fatalf("len(logs) = %d, want 2", len(logs))
	}
	if logs[0].ID != 1 || logs[0].RequestID != "req-1" || logs[0].Provider != "openai" {
		t.Fatalf("log[0] = %+v", logs[0])
	}
	if !logs[0].Timestamp.Equal(now) {
		t.Fatalf("log[0].timestamp = %v, want %v", logs[0].Timestamp, now)
	}
	if logs[0].TotalTokens == nil || *logs[0].TotalTokens != 10 {
		t.Fatalf("log[0].total_tokens = %v, want 10", logs[0].TotalTokens)
	}
	if logs[1].Provider != "anthropic" {
		t.Fatalf("log[1].provider = %s, want anthropic", logs[1].Provider)
	}
}

func TestListUsagePropagatesGetUsageError(t *testing.T) {
	reader := &fakeUsageReader{getErr: errors.New("db error")}
	_, _, err := usage.ListUsage(reader, usage.UsageFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListUsagePropagatesCountUsageError(t *testing.T) {
	reader := &fakeUsageReader{
		logs:     []usage.UsageLog{{}},
		countErr: errors.New("count error"),
	}
	_, _, err := usage.ListUsage(reader, usage.UsageFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSummaryReturnsSummary(t *testing.T) {
	reader := &fakeUsageReader{
		summary: &usage.UsageSummary{
			RequestCount: 5,
			TotalTokens:  100,
			TotalCostUSD: 1.5,
		},
	}
	summary, err := usage.GetSummary(reader, usage.UsageFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.RequestCount != 5 || summary.TotalTokens != 100 || summary.TotalCostUSD != 1.5 {
		t.Fatalf("summary = %+v, want 5/100/1.5", summary)
	}
}

func TestGetSummaryPropagatesError(t *testing.T) {
	reader := &fakeUsageReader{summaryErr: errors.New("summary error")}
	_, err := usage.GetSummary(reader, usage.UsageFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func intPtr(v int) *int {
	return &v
}
