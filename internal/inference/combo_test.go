package inference

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// fakeComboStore implements ComboStore for combo engine tests.
type fakeComboStore struct {
	combos map[string]*store.Combo
}

func (f *fakeComboStore) GetCombo(name string) (*store.Combo, error) {
	c, ok := f.combos[name]
	if !ok {
		return nil, store.ErrNotFound
	}
	return c, nil
}

// fakeModelRunner implements ModelRunner for combo engine tests.
// failMap[model][n] is the error returned on the n-th call to RunModel for that model.
// A nil entry means success (calls fn with a fake connection).
type fakeModelRunner struct {
	calls       []string
	failMap     map[string][]error
	callCounts  map[string]int
	retryAfters map[string]time.Time
}

func newFakeModelRunner() *fakeModelRunner {
	return &fakeModelRunner{
		failMap:     make(map[string][]error),
		callCounts:  make(map[string]int),
		retryAfters: make(map[string]time.Time),
	}
}

func (f *fakeModelRunner) RunModel(model string, fn func(*store.Connection) (Verdict, error)) error {
	f.calls = append(f.calls, model)
	n := f.callCounts[model]
	f.callCounts[model]++
	if results, ok := f.failMap[model]; ok && n < len(results) && results[n] != nil {
		return results[n]
	}
	fn(&store.Connection{ID: "conn-" + model})
	return nil
}

func (f *fakeModelRunner) ModelRetryAfter(model string, now time.Time) (time.Time, bool, error) {
	t := f.retryAfters[model]
	if t.IsZero() || !t.After(now) {
		return time.Time{}, false, nil
	}
	return t, true, nil
}

func TestComboFallbackOrder(t *testing.T) {
	cs := &fakeComboStore{combos: map[string]*store.Combo{
		"c1": {Name: "c1", Models: []string{"modelA", "modelB"}},
	}}
	mr := newFakeModelRunner()
	mr.failMap["modelA"] = []error{errors.New("unavailable")}
	engine := NewComboEngine(cs, &fakeSettingStore{}, mr, time.Now, func(d time.Duration) {})

	var calledWith []string
	err := engine.ExecuteCombo("c1", func(model string, conn *store.Connection) (Verdict, error) {
		calledWith = append(calledWith, model)
		return VerdictUnknown, nil
	})
	if err != nil {
		t.Fatalf("ExecuteCombo: %v", err)
	}
	// modelA tried first (failed), then modelB tried and succeeded.
	if len(mr.calls) != 2 || mr.calls[0] != "modelA" || mr.calls[1] != "modelB" {
		t.Errorf("runner calls = %v, want [modelA, modelB]", mr.calls)
	}
	// fn was only called for modelB (modelA returned error before fn was called).
	if len(calledWith) != 1 || calledWith[0] != "modelB" {
		t.Errorf("fn called with = %v, want [modelB]", calledWith)
	}
}

func TestComboRoundRobinSticky(t *testing.T) {
	cs := &fakeComboStore{combos: map[string]*store.Combo{
		"c1": {Name: "c1", Models: []string{"modelA", "modelB"}},
	}}
	ss := &fakeSettingStore{settings: map[string]string{
		"comboStrategy":              "round-robin",
		"comboStickyRoundRobinLimit": "2",
	}}
	mr := newFakeModelRunner()
	engine := NewComboEngine(cs, ss, mr, time.Now, func(d time.Duration) {})
	fn := func(model string, conn *store.Connection) (Verdict, error) { return VerdictUnknown, nil }

	// Calls 1 and 2: sticky on modelA (stickyLimit=2).
	for i := 0; i < 2; i++ {
		mr.calls = nil
		if err := engine.ExecuteCombo("c1", fn); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
		if len(mr.calls) != 1 || mr.calls[0] != "modelA" {
			t.Errorf("call %d: calls = %v, want [modelA]", i+1, mr.calls)
		}
	}

	// Call 3: stickyLimit reached — rotates to modelB.
	mr.calls = nil
	if err := engine.ExecuteCombo("c1", fn); err != nil {
		t.Fatalf("call 3: %v", err)
	}
	if len(mr.calls) != 1 || mr.calls[0] != "modelB" {
		t.Errorf("call 3: calls = %v, want [modelB]", mr.calls)
	}
}

func TestComboRecursionGuard(t *testing.T) {
	cs := &fakeComboStore{combos: map[string]*store.Combo{
		"a": {Name: "a", Models: []string{"b"}},
		"b": {Name: "b", Models: []string{"a"}},
	}}
	mr := newFakeModelRunner()
	engine := NewComboEngine(cs, &fakeSettingStore{}, mr, time.Now, func(d time.Duration) {})

	err := engine.ExecuteCombo("a", func(model string, conn *store.Connection) (Verdict, error) {
		return VerdictUnknown, nil
	})
	if !errors.Is(err, ErrComboRecursion) {
		t.Fatalf("expected ErrComboRecursion, got: %v", err)
	}
}

func TestComboTransientCooldownCap5s(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("wait_then_next", func(t *testing.T) {
		mr := newFakeModelRunner()
		mr.failMap["modelA"] = []error{errors.New("transient")}
		mr.retryAfters["modelA"] = now.Add(3 * time.Second) // ≤5s → sleep then fall to next model

		var slept []time.Duration
		cs := &fakeComboStore{combos: map[string]*store.Combo{
			"c1": {Name: "c1", Models: []string{"modelA", "modelB"}},
		}}
		engine := NewComboEngine(cs, &fakeSettingStore{}, mr, func() time.Time { return now }, func(d time.Duration) {
			slept = append(slept, d)
		})

		err := engine.ExecuteCombo("c1", func(model string, conn *store.Connection) (Verdict, error) {
			return VerdictUnknown, nil
		})
		if err != nil {
			t.Fatalf("expected success via modelB after wait, got: %v", err)
		}
		if len(slept) != 1 {
			t.Fatalf("expected 1 sleep call, got %d", len(slept))
		}
		if slept[0] > 5*time.Second {
			t.Errorf("sleep duration %v > 5s cap", slept[0])
		}
		// modelA tried once (failed + slept), then modelB tried once (succeeded).
		if len(mr.calls) != 2 || mr.calls[0] != "modelA" || mr.calls[1] != "modelB" {
			t.Errorf("calls = %v, want [modelA, modelB]", mr.calls)
		}
		if mr.callCounts["modelA"] != 1 {
			t.Errorf("modelA called %d times, want 1 (no retry of same model)", mr.callCounts["modelA"])
		}
	})

	t.Run("skip_on_long_cooldown", func(t *testing.T) {
		mr := newFakeModelRunner()
		mr.failMap["modelA"] = []error{errors.New("transient")}
		mr.retryAfters["modelA"] = now.Add(10 * time.Second) // >5s → skip to next model

		var slept []time.Duration
		cs := &fakeComboStore{combos: map[string]*store.Combo{
			"c1": {Name: "c1", Models: []string{"modelA", "modelB"}},
		}}
		engine := NewComboEngine(cs, &fakeSettingStore{}, mr, func() time.Time { return now }, func(d time.Duration) {
			slept = append(slept, d)
		})

		err := engine.ExecuteCombo("c1", func(model string, conn *store.Connection) (Verdict, error) {
			return VerdictUnknown, nil
		})
		if err != nil {
			t.Fatalf("expected success via modelB, got: %v", err)
		}
		if len(slept) != 0 {
			t.Errorf("expected no sleep, got %d sleep calls", len(slept))
		}
		if len(mr.calls) != 2 || mr.calls[0] != "modelA" || mr.calls[1] != "modelB" {
			t.Errorf("calls = %v, want [modelA, modelB]", mr.calls)
		}
	})
}

func TestComboEarliestRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cs := &fakeComboStore{combos: map[string]*store.Combo{
		"c1": {Name: "c1", Models: []string{"modelA", "modelB"}},
	}}
	mr := newFakeModelRunner()
	mr.retryAfters["modelA"] = now.Add(100 * time.Second)
	mr.retryAfters["modelB"] = now.Add(50 * time.Second)
	engine := NewComboEngine(cs, &fakeSettingStore{}, mr, func() time.Time { return now }, func(d time.Duration) {})

	retryAt, ok, err := engine.EarliestRetryAfter("c1", now)
	if err != nil {
		t.Fatalf("EarliestRetryAfter: %v", err)
	}
	if !ok {
		t.Fatal("ok=false, want true")
	}
	want := now.Add(50 * time.Second)
	if !retryAt.Equal(want) {
		t.Errorf("retryAt = %v, want %v", retryAt, want)
	}
}

func TestComboStateResetOnChange(t *testing.T) {
	cs := &fakeComboStore{combos: map[string]*store.Combo{
		"c1": {Name: "c1", Models: []string{"modelA", "modelB"}},
	}}
	ss := &fakeSettingStore{settings: map[string]string{
		"comboStrategy":              "round-robin",
		"comboStickyRoundRobinLimit": "1",
	}}
	mr := newFakeModelRunner()
	engine := NewComboEngine(cs, ss, mr, time.Now, func(d time.Duration) {})
	fn := func(model string, conn *store.Connection) (Verdict, error) { return VerdictUnknown, nil }

	// First call → modelA (round-robin, stickyLimit=1, starts at idx=0).
	mr.calls = nil
	if err := engine.ExecuteCombo("c1", fn); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if len(mr.calls) != 1 || mr.calls[0] != "modelA" {
		t.Fatalf("first call = %v, want [modelA]", mr.calls)
	}

	// Update combo definition — hash changes.
	cs.combos["c1"] = &store.Combo{Name: "c1", Models: []string{"modelX", "modelY"}}

	// Next call must start from idx=0 (modelX), not carry over idx=1 (modelB).
	mr.calls = nil
	if err := engine.ExecuteCombo("c1", fn); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(mr.calls) != 1 || mr.calls[0] != "modelX" {
		t.Fatalf("second call = %v, want [modelX] (state reset)", mr.calls)
	}
}
