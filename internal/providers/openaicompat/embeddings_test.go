package openaicompat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func newEmbeddingsProvider(t *testing.T, baseURL string) *Provider {
	t.Helper()
	p, err := New(Config{Provider: providers.ProviderTogether, BaseURL: baseURL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

func TestCompatEmbeddings(t *testing.T) {
	var gotPath string
	var gotBody providers.EmbeddingsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.5,0.6]}],"model":"m","usage":{"prompt_tokens":2,"total_tokens":2}}`))
	}))
	t.Cleanup(server.Close)

	p := newEmbeddingsProvider(t, server.URL)
	resp, err := p.Embeddings(context.Background(), testKey(providers.ProviderTogether), &providers.EmbeddingsRequest{
		Model: "m",
		Input: []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("Embeddings: %v", err)
	}
	if gotPath != "/v1/embeddings" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(resp.Data) != 1 || len(resp.Data[0].Embedding) != 2 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestCompatEmbeddingsNilRequest(t *testing.T) {
	p := newEmbeddingsProvider(t, "http://example.com")
	if _, err := p.Embeddings(context.Background(), testKey(providers.ProviderTogether), nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestCompatEmbeddingsErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad"}}`))
	}))
	t.Cleanup(server.Close)

	p := newEmbeddingsProvider(t, server.URL)
	if _, err := p.Embeddings(context.Background(), testKey(providers.ProviderTogether), &providers.EmbeddingsRequest{Model: "m"}); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestCompatImplementsEmbeddings(t *testing.T) {
	p := newEmbeddingsProvider(t, "http://example.com")
	var _ providers.EmbeddingsProvider = p
}
