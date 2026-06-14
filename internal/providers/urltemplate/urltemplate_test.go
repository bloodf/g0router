package urltemplate

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestBuildURLCloudflare verifies the {accountId} template substitution
// (default.js:64-68) yields the exact Cloudflare Workers AI endpoint.
func TestBuildURLCloudflare(t *testing.T) {
	p, err := New("cloudflare-ai")
	if err != nil {
		t.Fatalf("New(cloudflare-ai) error: %v", err)
	}
	got := p.buildURL(schemas.Key{ProviderSpecificData: map[string]string{"accountId": "abc123"}}, "@cf/meta/llama-3.2-1b-instruct")
	want := "https://api.cloudflare.com/client/v4/accounts/abc123/ai/v1/chat/completions"
	if got != want {
		t.Errorf("cloudflare buildURL = %q, want %q", got, want)
	}
}

// TestBuildURLCloudflareMissingAccount verifies a missing accountId is a hard
// error (default.js:66) surfaced at request time.
func TestBuildURLCloudflareMissingAccount(t *testing.T) {
	p, _ := New("cloudflare-ai")
	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.ChatRequest{Model: "x"})
	if perr == nil {
		t.Fatal("expected error for missing accountId, got nil")
	}
}

// TestBuildURLAzure verifies the Azure resource-URL build (azure.js:8-23) from
// providerSpecificData (azureEndpoint/deployment/apiVersion), with the model as
// the default deployment.
func TestBuildURLAzure(t *testing.T) {
	p, err := New("azure")
	if err != nil {
		t.Fatalf("New(azure) error: %v", err)
	}
	// Explicit deployment + apiVersion.
	got := p.buildURL(schemas.Key{ProviderSpecificData: map[string]string{
		"azureEndpoint": "https://my-res.openai.azure.com/",
		"deployment":    "gpt-4o",
		"apiVersion":    "2024-10-01-preview",
	}}, "gpt-4")
	want := "https://my-res.openai.azure.com/openai/deployments/gpt-4o/chat/completions?api-version=2024-10-01-preview"
	if got != want {
		t.Errorf("azure buildURL (explicit) = %q, want %q", got, want)
	}
	// Deployment falls back to the model; apiVersion default.
	got2 := p.buildURL(schemas.Key{ProviderSpecificData: map[string]string{
		"azureEndpoint": "https://r2.openai.azure.com",
	}}, "my-model")
	want2 := "https://r2.openai.azure.com/openai/deployments/my-model/chat/completions?api-version=2024-10-01-preview"
	if got2 != want2 {
		t.Errorf("azure buildURL (model deployment) = %q, want %q", got2, want2)
	}
}

// TestBuildURLXiaomiTokenplan verifies the region->baseURL resolution
// (providers.js:447-457) and the default region (sgp).
func TestBuildURLXiaomiTokenplan(t *testing.T) {
	p, err := New("xiaomi-tokenplan")
	if err != nil {
		t.Fatalf("New(xiaomi-tokenplan) error: %v", err)
	}
	cases := map[string]string{
		"sgp": "https://token-plan-sgp.xiaomimimo.com/v1/chat/completions",
		"cn":  "https://token-plan-cn.xiaomimimo.com/v1/chat/completions",
		"ams": "https://token-plan-ams.xiaomimimo.com/v1/chat/completions",
	}
	for region, want := range cases {
		got := p.buildURL(schemas.Key{ProviderSpecificData: map[string]string{"region": region}}, "mimo-v2.5-pro")
		if got != want {
			t.Errorf("xiaomi buildURL(region=%s) = %q, want %q", region, got, want)
		}
	}
	// Unknown/empty region defaults to sgp.
	if got := p.buildURL(schemas.Key{}, "mimo-v2.5-pro"); got != cases["sgp"] {
		t.Errorf("xiaomi buildURL(default) = %q, want %q", got, cases["sgp"])
	}
}

// TestChatCompletionRoundTrip verifies an OpenAI request/response passes through
// the cloudflare-ai adapter unchanged at the built URL.
func TestChatCompletionRoundTrip(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	p, _ := New("cloudflare-ai")
	// Override the resolved base for the test so the path is deterministic.
	p.urlOverride = srv.URL

	resp, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{
		Value:                "cf-key",
		ProviderSpecificData: map[string]string{"accountId": "abc"},
	}, &schemas.ChatRequest{Model: "@cf/meta/llama-3.2-1b-instruct"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if gotPath != "/" {
		t.Errorf("request path = %q, want / (urlOverride)", gotPath)
	}
	if gotAuth != "Bearer cf-key" {
		t.Errorf("Authorization = %q, want Bearer cf-key", gotAuth)
	}
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// TestAzureAuthHeader verifies azure uses the api-key header (azure.js:37), not
// Bearer.
func TestAzureAuthHeader(t *testing.T) {
	var gotAPIKey, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("api-key")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","object":"chat.completion","created":1,"model":"m","choices":[]}`))
	}))
	defer srv.Close()

	p, _ := New("azure")
	p.urlOverride = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "az-key"}, &schemas.ChatRequest{Model: "gpt-4"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if gotAPIKey != "az-key" {
		t.Errorf("api-key header = %q, want az-key", gotAPIKey)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty (azure uses api-key)", gotAuth)
	}
}

// TestStreamRoundTrip verifies SSE passes through the xiaomi-tokenplan adapter.
func TestStreamRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"c1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"a\"}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p, _ := New("xiaomi-tokenplan")
	p.urlOverride = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "k"}, &schemas.ChatRequest{Model: "mimo-v2.5-pro"})
	if perr != nil {
		t.Fatalf("stream error: %v", perr.Message)
	}
	var n int
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("error chunk: %v", chunk.Error.Message)
		}
		n++
	}
	if n != 1 {
		t.Errorf("chunks = %d, want 1", n)
	}
}
