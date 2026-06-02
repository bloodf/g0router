package logging

import (
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

type RequestLog struct {
	RequestID      string
	Timestamp      time.Time
	Provider       string
	Model          string
	ConnectionID   *string
	AuthType       string
	Usage          *usage.Usage
	CostUSD        *float64
	Latency        time.Duration
	StatusCode     int
	Error          *string
	SourceFormat   *string
	TargetFormat   *string
	RTKEnabled     *bool
	RTKBytesSaved  *int
	CavemanEnabled *bool
	ComboName      *string
	APIKeyID       *string
	ClientTool     *string
}

func (l RequestLog) Entry() store.RequestLogEntry {
	entry := store.RequestLogEntry{
		RequestID:      l.RequestID,
		Timestamp:      l.Timestamp,
		Provider:       l.Provider,
		Model:          l.Model,
		ConnectionID:   l.ConnectionID,
		AuthType:       l.AuthType,
		CostUSD:        l.CostUSD,
		Error:          l.Error,
		SourceFormat:   l.SourceFormat,
		TargetFormat:   l.TargetFormat,
		RTKEnabled:     l.RTKEnabled,
		RTKBytesSaved:  l.RTKBytesSaved,
		CavemanEnabled: l.CavemanEnabled,
		ComboName:      l.ComboName,
		APIKeyID:       l.APIKeyID,
		ClientTool:     l.ClientTool,
	}

	if l.Usage != nil {
		entry.InputTokens = intValue(l.Usage.InputTokens)
		entry.OutputTokens = intValue(l.Usage.OutputTokens)
		entry.CacheReadTokens = intValue(l.Usage.CacheReadTokens)
		entry.TotalTokens = intValue(l.Usage.TotalTokens)
	}
	if l.Latency > 0 {
		entry.LatencyMS = intValue(int(l.Latency / time.Millisecond))
	}
	if l.StatusCode != 0 {
		entry.StatusCode = intValue(l.StatusCode)
	}

	return entry
}

func intValue(value int) *int {
	return &value
}
