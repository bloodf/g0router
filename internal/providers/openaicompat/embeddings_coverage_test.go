package openaicompat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

// ---- Embeddings: newJSONRequest error via expired deadline ----

func TestCompatEmbeddingsExpiredDeadline(t *testing.T) {
	p := newEmbeddingsProvider(t, "http://127.0.0.1:1")
	deadline := time.Now().Add(-time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, err := p.Embeddings(ctx, testKey(providers.ProviderTogether), &providers.EmbeddingsRequest{Model: "m", Input: "x"})
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

// ---- Embeddings: do() network error ----

func TestCompatEmbeddingsDoError(t *testing.T) {
	p := newEmbeddingsProvider(t, "http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.Embeddings(ctx, testKey(providers.ProviderTogether), &providers.EmbeddingsRequest{Model: "m", Input: "x"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- Embeddings: bad JSON response ----

func TestCompatEmbeddingsBadJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	t.Cleanup(server.Close)

	p := newEmbeddingsProvider(t, server.URL)
	_, err := p.Embeddings(context.Background(), testKey(providers.ProviderTogether), &providers.EmbeddingsRequest{Model: "m", Input: "x"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}
