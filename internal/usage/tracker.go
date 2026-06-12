package usage

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const pendingTimeout = 60 * time.Second
const errorProviderWindow = 10 * time.Second

// Tracker counts in-flight requests by model and account.
// It emits "pending" events on every change and supports a 60s timeout per
// (connectionId, modelKey) pair.
type Tracker struct {
	clock         func() time.Time
	timerFactory  func(time.Duration, func()) func()
	events        *Events

	mu            sync.Mutex
	byModel       map[string]int64
	byAccount     map[string]map[string]int64
	timers        map[string]func()
	lastError     string
	lastErrorAt   time.Time
}

// NewTracker creates a Tracker with injected clock, timer factory, and event emitter.
func NewTracker(clock func() time.Time, timerFactory func(time.Duration, func()) func(), events *Events) *Tracker {
	return &Tracker{
		clock:        clock,
		timerFactory: timerFactory,
		events:       events,
		byModel:      make(map[string]int64),
		byAccount:    make(map[string]map[string]int64),
		timers:       make(map[string]func()),
	}
}

// Start increments the pending count for the given model/provider/connection.
func (t *Tracker) Start(model, provider, connectionID string) {
	t.mu.Lock()

	modelKey := t.modelKey(model, provider)
	t.byModel[modelKey] = t.byModel[modelKey] + 1

	if connectionID != "" {
		if t.byAccount[connectionID] == nil {
			t.byAccount[connectionID] = make(map[string]int64)
		}
		t.byAccount[connectionID][modelKey] = t.byAccount[connectionID][modelKey] + 1
	}

	timerKey := t.timerKey(connectionID, modelKey)
	if stop, ok := t.timers[timerKey]; ok {
		stop()
	}
	t.timers[timerKey] = t.timerFactory(pendingTimeout, func() {
		t.zeroOnTimeout(modelKey, connectionID)
	})

	t.mu.Unlock()
	t.events.Emit("pending")
}

// End decrements the pending count. If error is true and provider is present,
// the lastErrorProvider is recorded (lowercased) for a 10s window.
func (t *Tracker) End(model, provider, connectionID string, error bool) {
	t.mu.Lock()

	modelKey := t.modelKey(model, provider)
	t.byModel[modelKey] = max(0, t.byModel[modelKey]-1)
	if t.byModel[modelKey] == 0 {
		delete(t.byModel, modelKey)
	}

	if connectionID != "" {
		if acct, ok := t.byAccount[connectionID]; ok {
			acct[modelKey] = max(0, acct[modelKey]-1)
			if acct[modelKey] == 0 {
				delete(acct, modelKey)
			}
			if len(acct) == 0 {
				delete(t.byAccount, connectionID)
			}
		}
	}

	timerKey := t.timerKey(connectionID, modelKey)
	if stop, ok := t.timers[timerKey]; ok {
		stop()
		delete(t.timers, timerKey)
	}

	if error && provider != "" {
		t.lastError = strings.ToLower(provider)
		t.lastErrorAt = t.clock()
	}

	t.mu.Unlock()
	t.events.Emit("pending")
}

func (t *Tracker) zeroOnTimeout(modelKey, connectionID string) {
	t.mu.Lock()

	delete(t.timers, t.timerKey(connectionID, modelKey))

	if t.byModel[modelKey] > 0 {
		t.byModel[modelKey] = 0
	}
	delete(t.byModel, modelKey)

	if connectionID != "" {
		if acct, ok := t.byAccount[connectionID]; ok {
			if acct[modelKey] > 0 {
				acct[modelKey] = 0
			}
			delete(acct, modelKey)
			if len(acct) == 0 {
				delete(t.byAccount, connectionID)
			}
		}
	}

	t.mu.Unlock()
	t.events.Emit("pending")
}

// ByModelCount returns the current count for a model key (test seam).
func (t *Tracker) ByModelCount(modelKey string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.byModel[modelKey]
}

// ByAccountCount returns the current count for an account/model key (test seam).
func (t *Tracker) ByAccountCount(connectionID, modelKey string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if acct, ok := t.byAccount[connectionID]; ok {
		return acct[modelKey]
	}
	return 0
}

// LastErrorProvider returns the lowercased provider recorded within the last 10s.
func (t *Tracker) LastErrorProvider() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.clock().Sub(t.lastErrorAt) <= errorProviderWindow {
		return t.lastError
	}
	return ""
}

func (t *Tracker) modelKey(model, provider string) string {
	if provider != "" {
		return fmt.Sprintf("%s (%s)", model, provider)
	}
	return model
}

func (t *Tracker) timerKey(connectionID, modelKey string) string {
	return fmt.Sprintf("%s|%s", connectionID, modelKey)
}
