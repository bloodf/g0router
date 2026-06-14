package voyageai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestEmbeddingRoundTrip verifies the voyage-ai adapter POSTs the OpenAI-shaped
// embedding body with bearer auth and decodes the canned response. HERMETIC:
// the upstream is an httptest server returning a golden embeddings payload.
func TestEmbeddingRoundTrip(t *testing.T) {
	var gotMethod, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","embedding":[0.1,0.2,0.3],"index":0}],"model":"voyage-3-large","usage":{"prompt_tokens":4,"total_tokens":4}}`))
	}))
	defer srv.Close()

	p, err := New("voyage-ai")
	if err != nil {
		t.Fatalf("New(\"voyage-ai\") error: %v", err)
	}
	p.urlOverride = srv.URL

	resp, perr := p.Embedding(&schemas.GatewayContext{}, schemas.Key{Value: "voyage-secret-key"}, &schemas.EmbeddingRequest{
		Model: "voyage-3-large",
		Input: "hello world",
	})
	if perr != nil {
		t.Fatalf("Embedding error: %v", perr.Message)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("request method = %q, want POST", gotMethod)
	}
	if gotAuth != "Bearer voyage-secret-key" {
		t.Errorf("Authorization = %q, want Bearer voyage-secret-key", gotAuth)
	}
	var sent schemas.EmbeddingRequest
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("request body not valid EmbeddingRequest JSON: %v (body=%q)", err, gotBody)
	}
	if sent.Model != "voyage-3-large" {
		t.Errorf("request body model = %q, want voyage-3-large", sent.Model)
	}
	if resp == nil {
		t.Fatal("Embedding returned nil response")
	}
	if got, want := resp.Model, "voyage-3-large"; got != want {
		t.Errorf("response Model = %q, want %q", got, want)
	}
	if len(resp.Data) != 1 || len(resp.Data[0].Embedding) != 3 {
		t.Fatalf("response Data = %+v, want 1 entry with 3-dim vector", resp.Data)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 4 {
		t.Errorf("response Usage = %+v, want TotalTokens 4", resp.Usage)
	}
}

// TestEmbeddingNon200 verifies a non-200 upstream maps to a ProviderError.
func TestEmbeddingNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key","type":"authentication_error"}}`))
	}))
	defer srv.Close()

	p, _ := New("voyage-ai")
	p.urlOverride = srv.URL

	resp, perr := p.Embedding(&schemas.GatewayContext{}, schemas.Key{Value: "bad-key"}, &schemas.EmbeddingRequest{
		Model: "voyage-3-large",
		Input: "x",
	})
	if perr == nil {
		t.Fatal("Embedding error = nil, want ProviderError on 401")
	}
	if resp != nil {
		t.Errorf("Embedding response = %+v, want nil on error", resp)
	}
	if perr.StatusCode != http.StatusUnauthorized {
		t.Errorf("ProviderError StatusCode = %d, want 401", perr.StatusCode)
	}
}

// TestEmbeddingNoKeyLeak verifies the secret key value never appears in the
// ProviderError surfaced on failure (no echo/leak of credentials).
func TestEmbeddingNoKeyLeak(t *testing.T) {
	const secret = "super-secret-voyage-key-123"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer srv.Close()

	p, _ := New("voyage-ai")
	p.urlOverride = srv.URL

	_, perr := p.Embedding(&schemas.GatewayContext{}, schemas.Key{Value: secret}, &schemas.EmbeddingRequest{
		Model: "voyage-3-large",
		Input: "x",
	})
	if perr == nil {
		t.Fatal("expected ProviderError")
	}
	if strings.Contains(perr.Message, secret) {
		t.Errorf("ProviderError.Message leaks key value: %q", perr.Message)
	}
	if strings.Contains(string(perr.Meta.RawBody), secret) {
		t.Errorf("ProviderError.Meta.RawBody leaks key value: %q", perr.Meta.RawBody)
	}
}
