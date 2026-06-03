package proxy

import (
	"context"
	"errors"
	"fmt"

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
	store *store.Store
}

func NewComboResolver(s *store.Store) *ComboResolver {
	return &ComboResolver{store: s}
}

func (r *ComboResolver) Resolve(name string) ([]ComboStep, error) {
	combo, err := r.store.GetActiveCombo(name)
	if err != nil {
		return nil, fmt.Errorf("get active combo: %w", err)
	}

	steps := make([]ComboStep, len(combo.Steps))
	for i, step := range combo.Steps {
		steps[i] = ComboStep{
			Provider: providers.ModelProvider(providerids.CanonicalProviderID(step.Provider)),
			Model:    step.Model,
		}
	}
	return steps, nil
}

func (r *ComboResolver) Dispatch(ctx context.Context, engine *Engine, name string, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	steps, err := r.Resolve(name)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, ErrNoComboSteps
	}

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
	steps, err := r.Resolve(name)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, ErrNoComboSteps
	}

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
