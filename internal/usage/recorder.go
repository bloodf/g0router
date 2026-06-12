package usage

import (
	"fmt"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// UsageStore persists a request log entry.
type UsageStore interface {
	SaveUsage(*store.RequestLogEntry) error
}

// Recorder computes cost and persists usage entries.
type Recorder struct {
	resolver *Resolver
	store    UsageStore
	clock    func() time.Time
	events   *Events
}

// NewRecorder creates a Recorder with the given dependencies.
func NewRecorder(resolver *Resolver, store UsageStore, clock func() time.Time, events *Events) *Recorder {
	return &Recorder{
		resolver: resolver,
		store:    store,
		clock:    clock,
		events:   events,
	}
}

// Record normalizes tokens, resolves cost, persists the entry, and emits an "update" event.
func (r *Recorder) Record(entry *store.RequestLogEntry) error {
	if entry.Timestamp == "" {
		entry.Timestamp = r.clock().UTC().Format(time.RFC3339)
	}

	tokens := NormalizeTokens(entry.Tokens)
	entry.PromptTokens = tokens.Prompt
	entry.CompletionTokens = tokens.Completion

	entry.Cost = r.resolver.CostFor(entry.Provider, entry.Model, tokens)

	if err := r.store.SaveUsage(entry); err != nil {
		return fmt.Errorf("save usage: %w", err)
	}

	r.events.Emit("update")
	return nil
}
