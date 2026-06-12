package usage

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

type fakeUsageStore struct {
	entries []*store.RequestLogEntry
	err     error
}

func (f *fakeUsageStore) SaveUsage(e *store.RequestLogEntry) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, e)
	return nil
}

func TestRecorderComputesCost(t *testing.T) {
	us := &fakeUsageStore{}
	resolver := NewResolver(&fakeOverrideStore{}, func() int64 { return 0 })
	clock := func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) }
	rec := NewRecorder(resolver, us, clock, NewEvents())

	entry := &store.RequestLogEntry{
		Provider: "openai",
		Model:    "gpt-4o",
		Tokens:   map[string]int64{"prompt_tokens": 1000, "completion_tokens": 500},
	}
	if err := rec.Record(entry); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(us.entries) != 1 {
		t.Fatalf("captured entries = %d, want 1", len(us.entries))
	}
	got := us.entries[0]
	wantCost := 1000*2.5/1e6 + 500*10.0/1e6
	if got.Cost != wantCost {
		t.Errorf("Cost = %v, want %v", got.Cost, wantCost)
	}
	if got.PromptTokens != 1000 {
		t.Errorf("PromptTokens = %d, want 1000", got.PromptTokens)
	}
	if got.CompletionTokens != 500 {
		t.Errorf("CompletionTokens = %d, want 500", got.CompletionTokens)
	}
	if got.Timestamp != clock().Format(time.RFC3339) {
		t.Errorf("Timestamp = %q, want %q", got.Timestamp, clock().Format(time.RFC3339))
	}

	// Missing pricing resolves to zero cost.
	us.entries = nil
	entry2 := &store.RequestLogEntry{
		Provider: "unknown",
		Model:    "no-such-model",
		Tokens:   map[string]int64{"prompt_tokens": 100, "completion_tokens": 100},
	}
	if err := rec.Record(entry2); err != nil {
		t.Fatalf("Record missing pricing: %v", err)
	}
	if us.entries[0].Cost != 0 {
		t.Errorf("Cost for missing pricing = %v, want 0", us.entries[0].Cost)
	}
}

func TestRecorderEmitsUpdate(t *testing.T) {
	us := &fakeUsageStore{}
	resolver := NewResolver(&fakeOverrideStore{}, func() int64 { return 0 })
	clock := func() time.Time { return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC) }
	events := NewEvents()
	rec := NewRecorder(resolver, us, clock, events)

	var kinds []string
	events.OnEvent(func(kind string) {
		kinds = append(kinds, kind)
	})

	entry := &store.RequestLogEntry{Provider: "openai", Model: "gpt-4o", Tokens: map[string]int64{"prompt_tokens": 1, "completion_tokens": 1}}
	if err := rec.Record(entry); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(kinds) != 1 || kinds[0] != "update" {
		t.Errorf("emitted kinds = %v, want [update]", kinds)
	}

	// No event on save failure.
	us.err = errors.New("boom")
	kinds = nil
	if err := rec.Record(&store.RequestLogEntry{}); err == nil {
		t.Fatal("expected error from failing store")
	}
	if len(kinds) != 0 {
		t.Errorf("emitted kinds on failure = %v, want []", kinds)
	}
}
