package cli

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/providers"
)

func TestRegisterProviderFactoryError(t *testing.T) {
	engine := proxy.NewEngine(nil)
	registerProvider(engine, func() (providers.Provider, error) {
		return nil, errors.New("boom")
	})
	if len(engine.RegisteredProviders()) != 0 {
		t.Fatal("expected no providers registered")
	}
}
