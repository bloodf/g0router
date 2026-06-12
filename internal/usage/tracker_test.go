package usage

import (
	"sync"
	"testing"
	"time"
)

type testTimer struct {
	fn      func()
	stopped bool
}

func newTestTracker() (*Tracker, *time.Time, *[]testTimer, func()) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	var timers []testTimer
	var mu sync.Mutex
	clock := func() time.Time { return now }
	timerFactory := func(d time.Duration, fn func()) func() {
		mu.Lock()
		defer mu.Unlock()
		entry := testTimer{fn: fn}
		timers = append(timers, entry)
		idx := len(timers) - 1
		return func() {
			mu.Lock()
			defer mu.Unlock()
			timers[idx].stopped = true
		}
	}
	tr := NewTracker(clock, timerFactory, NewEvents())
	fireLast := func() {
		mu.Lock()
		fn := timers[len(timers)-1].fn
		mu.Unlock()
		fn()
	}
	return tr, &now, &timers, fireLast
}

func TestTrackerStartEnd(t *testing.T) {
	tr, _, _, _ := newTestTracker()

	tr.Start("gpt-4o", "openai", "conn-1")
	if got := tr.ByModelCount("gpt-4o (openai)"); got != 1 {
		t.Errorf("byModel count = %d, want 1", got)
	}
	if got := tr.ByAccountCount("conn-1", "gpt-4o (openai)"); got != 1 {
		t.Errorf("byAccount count = %d, want 1", got)
	}

	tr.End("gpt-4o", "openai", "conn-1", false)
	if got := tr.ByModelCount("gpt-4o (openai)"); got != 0 {
		t.Errorf("byModel count after end = %d, want 0", got)
	}
	if got := tr.ByAccountCount("conn-1", "gpt-4o (openai)"); got != 0 {
		t.Errorf("byAccount count after end = %d, want 0", got)
	}

	// Cleanup: maps should not retain empty entries.
	if len(tr.byModel) != 0 {
		t.Errorf("byModel not cleaned: %v", tr.byModel)
	}
	if len(tr.byAccount) != 0 {
		t.Errorf("byAccount not cleaned: %v", tr.byAccount)
	}

	// Clamping below zero.
	tr.End("gpt-4o", "openai", "conn-1", false)
	if got := tr.ByModelCount("gpt-4o (openai)"); got != 0 {
		t.Errorf("byModel count after over-end = %d, want 0", got)
	}
}

func TestTrackerTimeout(t *testing.T) {
	tr, _, timers, fireLast := newTestTracker()
	events := NewEvents()
	tr.events = events

	var kinds []string
	events.OnEvent(func(kind string) { kinds = append(kinds, kind) })

	tr.Start("gpt-4o", "openai", "conn-1")
	if len(*timers) != 1 {
		t.Fatalf("timers = %d, want 1", len(*timers))
	}
	if (*timers)[0].stopped {
		t.Error("timer stopped before end")
	}

	// Fire the timer manually.
	fireLast()
	if got := tr.ByModelCount("gpt-4o (openai)"); got != 0 {
		t.Errorf("byModel count after timeout = %d, want 0", got)
	}
	if got := tr.ByAccountCount("conn-1", "gpt-4o (openai)"); got != 0 {
		t.Errorf("byAccount count after timeout = %d, want 0", got)
	}
	if len(kinds) != 2 || kinds[1] != "pending" {
		t.Errorf("emitted kinds = %v, want [pending pending]", kinds)
	}
}

func TestTrackerErrorProvider(t *testing.T) {
	tr, now, _, _ := newTestTracker()

	tr.Start("gpt-4o", "openai", "conn-1")
	tr.End("gpt-4o", "openai", "conn-1", true)
	if got := tr.LastErrorProvider(); got != "openai" {
		t.Errorf("lastErrorProvider = %q, want openai", got)
	}

	*now = now.Add(11 * time.Second)
	if got := tr.LastErrorProvider(); got != "" {
		t.Errorf("lastErrorProvider after window = %q, want empty", got)
	}
}

func TestTrackerEmitsPending(t *testing.T) {
	tr, _, _, _ := newTestTracker()
	events := NewEvents()
	tr.events = events

	var kinds []string
	var mu sync.Mutex
	events.OnEvent(func(kind string) {
		mu.Lock()
		defer mu.Unlock()
		kinds = append(kinds, kind)
	})

	tr.Start("gpt-4o", "openai", "conn-1")
	tr.End("gpt-4o", "openai", "conn-1", false)

	mu.Lock()
	defer mu.Unlock()
	if len(kinds) != 2 {
		t.Errorf("emitted count = %d, want 2", len(kinds))
	}
	for _, k := range kinds {
		if k != "pending" {
			t.Errorf("kind = %q, want pending", k)
		}
	}
}

func TestTrackerConcurrent(t *testing.T) {
	tr, _, _, _ := newTestTracker()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr.Start("gpt-4o", "openai", "conn-1")
			tr.End("gpt-4o", "openai", "conn-1", false)
		}()
	}
	wg.Wait()
	if got := tr.ByModelCount("gpt-4o (openai)"); got != 0 {
		t.Errorf("byModel count = %d, want 0", got)
	}
}
