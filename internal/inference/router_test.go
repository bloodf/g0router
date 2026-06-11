package inference

import (
	"sync"
	"testing"

	"github.com/bloodf/g0router/internal/providers/generic"
	"github.com/bloodf/g0router/internal/providers/ollama"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

func TestResolveOpenAI(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Errorf("provider = %q, want openai", p.GetProvider())
	}
	if key.Provider != "openai" {
		t.Errorf("key provider = %q, want openai", key.Provider)
	}
}

func TestResolveAnthropic(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("anthropic/claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Errorf("provider = %q, want anthropic", p.GetProvider())
	}
	if key.Provider != "anthropic" {
		t.Errorf("key provider = %q, want anthropic", key.Provider)
	}
}

func TestResolveAnthropicByModelName(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("claude-3-opus-20240229")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Errorf("provider = %q, want anthropic", p.GetProvider())
	}
	if key.Provider != "anthropic" {
		t.Errorf("key provider = %q, want anthropic", key.Provider)
	}
}

func TestResolveGemini(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gemini/gemini-1.5-pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderGemini {
		t.Errorf("provider = %q, want gemini", p.GetProvider())
	}
	if key.Provider != "gemini" {
		t.Errorf("key provider = %q, want gemini", key.Provider)
	}
}

func TestResolveGeminiByModelName(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gemini-1.5-flash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderGemini {
		t.Errorf("provider = %q, want gemini", p.GetProvider())
	}
	if key.Provider != "gemini" {
		t.Errorf("key provider = %q, want gemini", key.Provider)
	}
}

func TestResolveDeepSeekRoutesToGeneric(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("deepseek-chat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*generic.Provider); !ok {
		t.Fatalf("provider type = %T, want *generic.Provider", p)
	}
	if p.GetProvider() != schemas.ProviderDeepSeek {
		t.Errorf("GetProvider() = %q, want deepseek", p.GetProvider())
	}
	if key.Provider != "deepseek" {
		t.Errorf("key provider = %q, want deepseek", key.Provider)
	}
}

func TestResolveOllamaRoutesToOllama(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("gpt-oss:120b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := p.(*ollama.Provider); !ok {
		t.Fatalf("provider type = %T, want *ollama.Provider", p)
	}
	if p.GetProvider() != schemas.ProviderOllama {
		t.Errorf("GetProvider() = %q, want ollama", p.GetProvider())
	}
	if key.Provider != "ollama" {
		t.Errorf("key provider = %q, want ollama", key.Provider)
	}
}

func TestResolveClaudePrefixUnchanged(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderAnthropic {
		t.Errorf("provider = %q, want anthropic", p.GetProvider())
	}
	if key.Provider != "anthropic" {
		t.Errorf("key provider = %q, want anthropic", key.Provider)
	}
}

func TestResolveUnknownDefaultsOpenAI(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	p, key, err := r.Resolve("totally-unknown-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Errorf("provider = %q, want openai", p.GetProvider())
	}
	if key.Provider != "openai" {
		t.Errorf("key provider = %q, want openai", key.Provider)
	}
}

func TestRouterUsesKeyResolver(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	r.SetKeyResolver(&fakeResolver{
		key: schemas.Key{ID: "k1", Provider: "openai", Value: "resolved-key"},
		psd: map[string]string{"baseUrl": "http://custom"},
	})
	p, key, err := r.Resolve("gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.Value != "resolved-key" {
		t.Errorf("key.Value = %q, want resolved-key", key.Value)
	}
	if key.Provider != "openai" {
		t.Errorf("key.Provider = %q, want openai", key.Provider)
	}
	if key.ProviderSpecificData["baseUrl"] != "http://custom" {
		t.Errorf("psd baseUrl = %q, want http://custom", key.ProviderSpecificData["baseUrl"])
	}
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Errorf("provider = %q, want openai", p.GetProvider())
	}
}

func TestRouterNilResolverUnchanged(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	r.SetKeyResolver(nil)
	p, key, err := r.Resolve("gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GetProvider() != schemas.ProviderOpenAI {
		t.Errorf("provider = %q, want openai", p.GetProvider())
	}
	if key.Provider != "openai" {
		t.Errorf("key.Provider = %q, want openai", key.Provider)
	}
	if key.Value != "" {
		t.Errorf("key.Value = %q, want empty", key.Value)
	}
}

type fakeResolver struct {
	key schemas.Key
	psd map[string]string
	err error
}

func (f *fakeResolver) ResolveKey(providerID string) (schemas.Key, map[string]string, error) {
	k := f.key
	k.Provider = providerID
	return k, f.psd, f.err
}

func TestSetKeyResolverConcurrent(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	models := []string{"gpt-4", "anthropic/claude-3-5-sonnet"}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				r.SetKeyResolver(&fakeResolver{
					key: schemas.Key{ID: "k", Value: "resolved"},
				})
			} else {
				r.SetKeyResolver(nil)
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			for _, m := range models {
				p, _, err := r.Resolve(m)
				if err != nil {
					t.Errorf("Resolve(%q) error: %v", m, err)
				}
				if p == nil {
					t.Errorf("Resolve(%q) returned nil provider", m)
				}
			}
		}
	}()

	wg.Wait()
}

func TestRouterConcurrentResolve(t *testing.T) {
	r := NewRouter(translation.NewRegistry())
	models := []string{"deepseek-chat", "gpt-4", "anthropic/claude-3-5-sonnet", "gemini/gemini-1.5-pro"}

	const workers = 50
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(workers * len(models))

	for i := 0; i < workers; i++ {
		for _, model := range models {
			go func(m string) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					p, _, err := r.Resolve(m)
					if err != nil {
						t.Errorf("Resolve(%q) error: %v", m, err)
						return
					}
					if p == nil {
						t.Errorf("Resolve(%q) returned nil provider", m)
						return
					}
				}
			}(model)
		}
	}

	wg.Wait()
}
