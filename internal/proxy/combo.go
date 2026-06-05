package proxy

import (
	"context"
	"errors"
	"fmt"
	"sync"

	providerids "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

var ErrNoComboSteps = errors.New("combo has no steps")

type ComboStep struct {
	Provider providers.ModelProvider
	Model    string
}

type ComboResolver struct {
	store      *store.Store
	selectorMu sync.Mutex
	selectors  map[string]*comboSelector
}

func NewComboResolver(s *store.Store) *ComboResolver {
	return &ComboResolver{store: s, selectors: make(map[string]*comboSelector)}
}

// selectorFor returns the shared selection state for a combo, creating it on
// first use. Thread-safe.
func (r *ComboResolver) selectorFor(name string) *comboSelector {
	r.selectorMu.Lock()
	defer r.selectorMu.Unlock()
	sel, ok := r.selectors[name]
	if !ok {
		sel = &comboSelector{}
		r.selectors[name] = sel
	}
	return sel
}

func (r *ComboResolver) Resolve(name string) ([]ComboStep, error) {
	steps, _, err := r.resolveWithStrategy(name)
	return steps, err
}

func (r *ComboResolver) resolveWithStrategy(name string) ([]ComboStep, string, error) {
	combo, err := r.store.GetActiveCombo(name)
	if err != nil {
		return nil, "", fmt.Errorf("get active combo: %w", err)
	}

	steps := make([]ComboStep, len(combo.Steps))
	for i, step := range combo.Steps {
		steps[i] = ComboStep{
			Provider: providers.ModelProvider(providerids.CanonicalProviderID(step.Provider)),
			Model:    step.Model,
		}
	}
	strategy := combo.Strategy
	if strategy == "" {
		strategy = store.ComboStrategyFallback
	}
	return steps, strategy, nil
}

// orderComboSteps applies the combo strategy to produce the ordered list of
// steps to try. The original order is always fully represented so remaining
// steps act as fallbacks.
func (r *ComboResolver) orderComboSteps(name, strategy string, steps []ComboStep, req *providers.ChatRequest) []ComboStep {
	return r.orderComboStepsWithStats(name, strategy, steps, req, nil)
}

// orderComboStepsWithStats is the same as orderComboSteps but accepts
// pre-fetched telemetry stats so fastest/cheapest avoid a redundant query.
func (r *ComboResolver) orderComboStepsWithStats(name, strategy string, steps []ComboStep, req *providers.ChatRequest, stats map[string]store.ModelStat) []ComboStep {
	if strategy == store.ComboStrategyFallback {
		return steps
	}
	ordered, _ := r.selectorFor(name).orderedStepsWithStats(strategy, steps, req, stats)
	return ordered
}

func (r *ComboResolver) Dispatch(ctx context.Context, engine *Engine, name string, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	steps, strategy, err := r.resolveWithStrategy(name)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, ErrNoComboSteps
	}
	var stats map[string]store.ModelStat
	if strategy == store.ComboStrategyFastest || strategy == store.ComboStrategyCheapest {
		stats = fetchTelemetryStats(r.store)
	}
	steps = r.orderComboStepsWithStats(name, strategy, steps, req, stats)

	var lastErr error
	for _, step := range steps {
		resp, err := dispatchComboStep(ctx, engine, step, req)
		if err == nil {
			return resp, nil
		}
		if errors.Is(err, ErrQuotaExhausted) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func (r *ComboResolver) DispatchStream(ctx context.Context, engine *Engine, name string, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	steps, strategy, err := r.resolveWithStrategy(name)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, ErrNoComboSteps
	}
	var stats map[string]store.ModelStat
	if strategy == store.ComboStrategyFastest || strategy == store.ComboStrategyCheapest {
		stats = fetchTelemetryStats(r.store)
	}
	steps = r.orderComboStepsWithStats(name, strategy, steps, req, stats)

	var lastErr error
	for _, step := range steps {
		stream, err := dispatchComboStreamStep(ctx, engine, step, req)
		if err == nil {
			return stream, nil
		}
		if errors.Is(err, ErrQuotaExhausted) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func dispatchComboStep(ctx context.Context, engine *Engine, step ComboStep, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	route, err := engine.resolveComboStepRoute(step)
	if err != nil {
		return nil, err
	}
	resp, err := engine.dispatchRoute(ctx, route, req)
	if err != nil {
		return nil, fmt.Errorf("combo step %s/%s: %w", route.Provider, route.Model, err)
	}
	return resp, nil
}

func dispatchComboStreamStep(ctx context.Context, engine *Engine, step ComboStep, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	route, err := engine.resolveComboStepRoute(step)
	if err != nil {
		return nil, err
	}
	stream, err := engine.dispatchStreamRoute(ctx, route, req)
	if err != nil {
		return nil, fmt.Errorf("combo step %s/%s stream: %w", route.Provider, route.Model, err)
	}
	return stream, nil
}
