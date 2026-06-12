package inference

import (
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers/catalog"
)

// AliasStore abstracts persisted model alias operations.
type AliasStore interface {
	CreateAlias(name, target string) error
	ResolveChain(name string) (string, error)
}

// ResolveModelAlias follows alias chains and returns the resolved model name.
// Unknown names pass through unchanged.
func ResolveModelAlias(st AliasStore, name string) string {
	resolved, err := st.ResolveChain(name)
	if err != nil {
		return name
	}
	return resolved
}

// CreateAlias persists a new alias after validating that it would not create a cycle.
func CreateAlias(st AliasStore, name, target string) error {
	if name == "" {
		return fmt.Errorf("alias name must not be empty")
	}
	if target == "" {
		return fmt.Errorf("alias target must not be empty")
	}
	if name == target {
		return fmt.Errorf("alias %q would create a cycle (self-loop)", name)
	}

	// Cycle detection: if following the existing chain from target ever reaches name,
	// adding name→target would close a cycle.
	resolved, err := st.ResolveChain(target)
	if err != nil {
		return fmt.Errorf("cycle check for %q: %w", name, err)
	}
	if resolved == name {
		return fmt.Errorf("alias %q -> %q would create a cycle", name, target)
	}

	if err := st.CreateAlias(name, target); err != nil {
		return fmt.Errorf("create alias %q -> %q: %w", name, target, err)
	}
	return nil
}

// ParseModelPrefix splits a model string into an optional provider or alias
// prefix and the bare model name. It mirrors parseModel from
// open-sse/services/model.js:155-167.
func ParseModelPrefix(model string) (providerPrefix, bareModel string) {
	idx := strings.Index(model, "/")
	if idx < 0 {
		return "", model
	}
	return model[:idx], model[idx+1:]
}

// InferProvider checks whether the bare model name starts with a known provider
// alias prefix. This is the PAR-ROUTE-008 name-prefix inference fallback.
func InferProvider(bareModel string) (providerID string, ok bool) {
	var found string
	catalog.ForEachProviderAlias(func(alias, id string) {
		if strings.HasPrefix(bareModel, alias+"-") {
			found = id
		}
	})
	if found != "" {
		return found, true
	}
	return "", false
}
