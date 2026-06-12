package usage

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bloodf/g0router/internal/store"
)

// StatsMap returns the Stats result for the given period as a JSON-shaped map.
// It is used by the SSE usage stream to send full stats frames.
func (s *StatsService) StatsMap(period string) (map[string]any, error) {
	stats, err := s.Stats(period)
	if err != nil {
		return nil, fmt.Errorf("stats %s: %w", period, err)
	}
	b, err := json.Marshal(stats)
	if err != nil {
		return nil, fmt.Errorf("marshal stats: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal stats: %w", err)
	}
	return m, nil
}

// StreamSnapshot returns the lightweight fields that are overlaid onto cached
// stats for quick and pending SSE frames.
func (s *StatsService) StreamSnapshot() (map[string]any, error) {
	if s.tracker == nil || s.ring == nil {
		return nil, fmt.Errorf("tracker or ring not available")
	}
	entries := s.ring.Snapshot()
	recent := make([]RecentRequest, 0, len(entries))
	for _, e := range entries {
		recent = append(recent, RecentRequest{
			Timestamp:        e.Timestamp,
			Model:            e.Model,
			Provider:         e.Provider,
			PromptTokens:     e.PromptTokens,
			CompletionTokens: e.CompletionTokens,
			Status:           e.Status,
		})
	}
	return map[string]any{
		"active_requests": []any{},
		"recent_requests": DedupeRecent(recent),
		"error_provider":  s.tracker.LastErrorProvider(),
	}, nil
}

// StreamEvents exposes the tracker event emitter so the usage stream can
// subscribe to update/pending events.
func (s *StatsService) StreamEvents() *Events {
	return s.tracker.events
}

// OffEvent removes a previously registered callback. It is the counterpart to
// OnEvent and is used by the usage stream to avoid leaking subscriptions.
func (e *Events) OffEvent(fn func(kind string)) {
	target := reflect.ValueOf(fn).Pointer()
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, cb := range e.cbs {
		if reflect.ValueOf(cb).Pointer() == target {
			e.cbs = append(e.cbs[:i], e.cbs[i+1:]...)
			return
		}
	}
}

// FetchProviderUsage returns usage/quota data for a single provider connection.
// Stage-1 supports anthropic (Claude) and gemini; all other providers return a
// fallback message.
func FetchProviderUsage(providerType string, conn *store.Connection, client interface{}) (map[string]any, error) {
	_ = conn
	_ = client
	return map[string]any{
		"message": fmt.Sprintf("Usage API not implemented for %s", providerType),
	}, nil
}
