package inference

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// ErrComboRecursion is returned when a circular combo reference is detected.
var ErrComboRecursion = errors.New("combo recursion detected")

// ErrComboAllExhausted is returned when all models in a combo are exhausted.
var ErrComboAllExhausted = errors.New("all combo models exhausted")

// ErrModelTransient is wrapped by ModelRunner.RunModel for HTTP 502/503/504 failures.
// Only transient-classified errors qualify for the ≤5s cooldown-sleep before the next model.
var ErrModelTransient = errors.New("model transient failure")

// ComboStore provides combo lookup for the engine.
type ComboStore interface {
	GetCombo(name string) (*store.Combo, error)
}

// ModelRunner executes a model request with account-level fallback.
type ModelRunner interface {
	RunModel(model string, fn func(*store.Connection) (Verdict, error)) error
	ModelRetryAfter(model string, now time.Time) (time.Time, bool, error)
}

type comboRRState struct {
	hash  string
	idx   int
	count int
}

// ComboEngine resolves and executes model combo chains.
// Strategy and sticky limit are read from settings, NOT stored on the combo row.
// Round-robin state is in-memory with no TTL — resets on restart.
// Mirrors combo.js strategy dispatch and ordered-model fallback loop.
type ComboEngine struct {
	cs       ComboStore
	ss       SettingStore
	mr       ModelRunner
	clock    func() time.Time
	sleep    func(time.Duration)
	mu       sync.Mutex
	rrStates map[string]*comboRRState // keyed by combo name
}

// NewComboEngine creates a ComboEngine with injected dependencies.
func NewComboEngine(cs ComboStore, ss SettingStore, mr ModelRunner, clock func() time.Time, sleep func(time.Duration)) *ComboEngine {
	return &ComboEngine{
		cs:       cs,
		ss:       ss,
		mr:       mr,
		clock:    clock,
		sleep:    sleep,
		rrStates: make(map[string]*comboRRState),
	}
}

type comboStrategyCfg struct {
	Strategy    string `json:"fallbackStrategy"`
	StickyLimit int    `json:"stickyRoundRobinLimit"`
}

// normalizeStickyLimit returns n if positive, otherwise 1.
// Mirrors normalizeStickyLimit in combo.js:14-17.
func normalizeStickyLimit(n int) int {
	if n <= 0 {
		return 1
	}
	return n
}

// resolveComboStrategy returns effective strategy and sticky limit for a combo name.
func (e *ComboEngine) resolveComboStrategy(name string) (strategy string, stickyLimit int) {
	raw, _ := e.ss.GetSetting("comboStrategies")
	if raw != "" {
		var pmap map[string]comboStrategyCfg
		if json.Unmarshal([]byte(raw), &pmap) == nil {
			if cfg, ok := pmap[name]; ok {
				strategy = cfg.Strategy
				if cfg.StickyLimit > 0 {
					stickyLimit = cfg.StickyLimit
				}
			}
		}
	}
	if strategy == "" {
		s, _ := e.ss.GetSetting("comboStrategy")
		strategy = s
	}
	if strategy == "" {
		strategy = "fallback"
	}
	if stickyLimit == 0 && strategy == "round-robin" {
		s, _ := e.ss.GetSetting("comboStickyRoundRobinLimit")
		if n, err := strconv.Atoi(s); err == nil {
			stickyLimit = n
		}
	}
	stickyLimit = normalizeStickyLimit(stickyLimit)
	return
}

func comboHash(models []string) string {
	return strings.Join(models, "\x00")
}

// startIdx returns the model index to start with for this call and advances round-robin state.
// For fill-first, always returns 0. Mirrors combo.js:36-65.
func (e *ComboEngine) startIdx(name string, models []string, strategy string, stickyLimit int) int {
	if strategy != "round-robin" {
		return 0
	}
	hash := comboHash(models)
	e.mu.Lock()
	defer e.mu.Unlock()
	state, ok := e.rrStates[name]
	if !ok || state.hash != hash {
		// New combo or definition changed (PAR-PR-648) — reset to idx=0.
		e.rrStates[name] = &comboRRState{hash: hash, idx: 0, count: 1}
		return 0
	}
	if state.count < stickyLimit {
		state.count++
		return state.idx
	}
	state.idx = (state.idx + 1) % len(models)
	state.count = 1
	return state.idx
}

// ExecuteCombo resolves the named combo and executes fn against models in strategy order,
// falling back through models on per-model failure. Detects and rejects circular references.
func (e *ComboEngine) ExecuteCombo(name string, fn func(model string, conn *store.Connection) (Verdict, error)) error {
	return e.executeCombo(name, fn, map[string]bool{name: true})
}

func (e *ComboEngine) executeCombo(name string, fn func(model string, conn *store.Connection) (Verdict, error), visited map[string]bool) error {
	combo, err := e.cs.GetCombo(name)
	if err != nil {
		return fmt.Errorf("get combo %s: %w", name, err)
	}
	if len(combo.Models) == 0 {
		return ErrComboAllExhausted
	}

	strategy, stickyLimit := e.resolveComboStrategy(name)
	start := e.startIdx(name, combo.Models, strategy, stickyLimit)
	n := len(combo.Models)

	for i := 0; i < n; i++ {
		idx := (start + i) % n
		model := combo.Models[idx]

		// Sub-combo reference: check for recursion, then delegate.
		if _, comboErr := e.cs.GetCombo(model); comboErr == nil {
			if visited[model] {
				return ErrComboRecursion
			}
			visited[model] = true
			subErr := e.executeCombo(model, fn, visited)
			delete(visited, model)
			if subErr == nil {
				return nil
			}
			if errors.Is(subErr, ErrComboRecursion) {
				return subErr
			}
			continue
		}

		// Plain model: delegate to account-level runner.
		runErr := e.mr.RunModel(model, func(conn *store.Connection) (Verdict, error) {
			return fn(model, conn)
		})
		if runErr == nil {
			return nil
		}

		// Transient cooldown ≤5s: sleep before falling back to the next model. (combo.js:161-165)
		// Only fires for ErrModelTransient (502/503/504); rate-limit and other errors skip directly.
		if errors.Is(runErr, ErrModelTransient) {
			now := e.clock()
			if retryAt, ok, _ := e.mr.ModelRetryAfter(model, now); ok {
				if wait := retryAt.Sub(now); wait <= 5*time.Second {
					e.sleep(wait)
				}
			}
		}
	}
	return ErrComboAllExhausted
}

// EarliestRetryAfter returns the earliest retry-after time across all models in the named combo.
// Mirrors the retryAfter aggregation in combo.js (PAR-ROUTE-046).
func (e *ComboEngine) EarliestRetryAfter(name string, now time.Time) (time.Time, bool, error) {
	combo, err := e.cs.GetCombo(name)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("get combo %s: %w", name, err)
	}
	var earliest time.Time
	found := false
	for _, model := range combo.Models {
		t, ok, err := e.mr.ModelRetryAfter(model, now)
		if err != nil {
			return time.Time{}, false, err
		}
		if ok && (!found || t.Before(earliest)) {
			earliest = t
			found = true
		}
	}
	return earliest, found, nil
}
