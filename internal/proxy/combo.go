package proxy

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

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
