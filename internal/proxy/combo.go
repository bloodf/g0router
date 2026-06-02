package proxy

import (
	"context"
	"errors"
	"fmt"

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
			Provider: providers.ModelProvider(step.Provider),
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
		lastErr = err
	}

	return nil, lastErr
}

func dispatchComboStep(ctx context.Context, engine *Engine, step ComboStep, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	provider, ok := engine.pool.get(step.Provider)
	if !ok {
		return nil, fmt.Errorf("combo step provider %s: %w", step.Provider, ErrProviderNotFound)
	}

	key, err := engine.keyFor(step.Provider)
	if err != nil {
		return nil, fmt.Errorf("combo step key %s: %w", step.Provider, err)
	}

	stepReq := *req
	stepReq.Model = step.Model
	resp, err := provider.ChatCompletion(ctx, key, &stepReq)
	if err != nil {
		return nil, fmt.Errorf("combo step %s/%s: %w", step.Provider, step.Model, err)
	}

	return resp, nil
}
