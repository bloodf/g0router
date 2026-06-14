package urltemplate

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestBuildURLVertexPartner verifies the vertex partner-openai URL build
// (vertex.js:49-53): the global OpenAI-compatible endpoint scoped by project_id
// from providerSpecificData. The native gemini-on-vertex format is deferred
// (ESC-A1).
func TestBuildURLVertexPartner(t *testing.T) {
	p, err := New("vertex")
	if err != nil {
		t.Fatalf("New(vertex) error: %v", err)
	}
	got := p.buildURL(schemas.Key{ProviderSpecificData: map[string]string{"projectId": "my-proj"}}, "zai-org/glm-5-maas")
	want := "https://aiplatform.googleapis.com/v1/projects/my-proj/locations/global/endpoints/openapi/chat/completions"
	if got != want {
		t.Errorf("vertex partner buildURL = %q, want %q", got, want)
	}
}

// TestBuildURLVertexMissingProject verifies a missing project_id is a hard error
// (vertex.js:51).
func TestBuildURLVertexMissingProject(t *testing.T) {
	p, _ := New("vertex")
	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.ChatRequest{Model: "x"})
	if perr == nil {
		t.Fatal("expected error for missing projectId, got nil")
	}
}

// TestVertexBearerAuthAndSecretSafety verifies the partner path authenticates
// with the Bearer token (the SA-JSON->token mint is deferred ESC-A1) and never
// logs/echoes the secret in the error path. The OpenAI request/response passes
// through unchanged.
func TestVertexBearerAuthAndSecretSafety(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p, _ := New("vertex")
	p.urlOverride = srv.URL

	secret := "ya29.super-secret-token"
	resp, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{
		Value:                secret,
		ProviderSpecificData: map[string]string{"projectId": "my-proj"},
	}, &schemas.ChatRequest{Model: "zai-org/glm-5-maas"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if gotAuth != "Bearer "+secret {
		t.Errorf("Authorization = %q, want Bearer <secret>", gotAuth)
	}
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	// Secret-safety: an error response must not echo the secret token.
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"unauthorized","type":"auth_error"}}`))
	}))
	defer errSrv.Close()
	p2, _ := New("vertex")
	p2.urlOverride = errSrv.URL
	_, perr2 := p2.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{
		Value:                secret,
		ProviderSpecificData: map[string]string{"projectId": "my-proj"},
	}, &schemas.ChatRequest{Model: "zai-org/glm-5-maas"})
	if perr2 == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if strings.Contains(perr2.Message, secret) || strings.Contains(string(perr2.Meta.RawBody), secret) {
		t.Errorf("error path leaked the secret token: msg=%q raw=%q", perr2.Message, string(perr2.Meta.RawBody))
	}
}
