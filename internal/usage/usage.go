package usage

import (
	"fmt"
	"time"
)

type UsageFilter struct {
	Provider     *string
	Model        *string
	AuthType     *string
	APIKeyID     *string
	SourceFormat *string
	StatusClass  string
	Search       string
	From         *time.Time
	To           *time.Time
	Start        *time.Time
	End          *time.Time
	Limit        int
	Offset       int
}

type UsageLog struct {
	ID                 int64
	RequestID          string
	Timestamp          time.Time
	Provider           string
	Model              string
	ConnectionID       *string
	AuthType           string
	InputTokens        *int
	OutputTokens       *int
	CacheReadTokens    *int
	CacheWriteTokens   *int
	TotalTokens        *int
	CostUSD            *float64
	LatencyMS          *int
	StatusCode         *int
	Error              *string
	SourceFormat       *string
	TargetFormat       *string
	RTKEnabled         *bool
	RTKBytesSaved      *int
	CavemanEnabled     *bool
	ComboName          *string
	APIKeyID           *string
	APIKeyName         *string
	ClientTool         *string
	ConnectionName     *string
	ConnectionProvider *string
	AccountEmail       *string
}

type UsageSummary struct {
	RequestCount int64
	TotalTokens  int64
	TotalCostUSD float64
}

type UsageReader interface {
	GetUsage(filter UsageFilter) ([]UsageLog, error)
	CountUsage(filter UsageFilter) (int, error)
	GetUsageSummary(filter UsageFilter) (*UsageSummary, error)
}

func ListUsage(reader UsageReader, filter UsageFilter) ([]UsageLog, int, error) {
	logs, err := reader.GetUsage(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("get usage: %w", err)
	}
	total, err := reader.CountUsage(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count usage: %w", err)
	}
	return logs, total, nil
}

func GetSummary(reader UsageReader, filter UsageFilter) (UsageSummary, error) {
	summary, err := reader.GetUsageSummary(filter)
	if err != nil {
		return UsageSummary{}, fmt.Errorf("get usage summary: %w", err)
	}
	if summary == nil {
		return UsageSummary{}, nil
	}
	return *summary, nil
}
