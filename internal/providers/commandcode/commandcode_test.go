package commandcode

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

type fakePostHookRunner struct{ err error }

func (f *fakePostHookRunner) Run(ctx *schemas.GatewayContext, response any) error { return f.err }

// TestNewCommandCodeProvider verifies the adapter is catalog-bound and reports
// the commandcode provider id.
func TestNewCommandCodeProvider(t *testing.T) {
	reg := translation.NewRegistry()
	p, err := New("commandcode", reg)
	if err != nil {
		t.Fatalf("New(commandcode) error: %v", err)
	}
	if p.GetProvider() != schemas.ModelProvider("commandcode") {
		t.Errorf("GetProvider() = %q, want commandcode", p.GetProvider())
	}
}

// TestCommandCodeChatCompletion verifies the round-trip: the OpenAI request is
// translated openai->commandcode (custom JSON with threadId/params via the
// existing registry converter) before POST, the configured headers/auth are
// sent, and the canned commandcode SSE response is translated back
// commandcode->openai into a ChatResponse.
func TestCommandCodeChatCompletion(t *testing.T) {
	var gotBody map[string]any
	var gotAuth, gotVersion, gotCLIEnv string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		gotAuth = r.Header.Get("Authorization")
		gotVersion = r.Header.Get("x-command-code-version")
		gotCLIEnv = r.Header.Get("x-cli-environment")
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"type\":\"text-delta\",\"text\":\"hello\"}\n\n")
		io.WriteString(w, "data: {\"type\":\"text-delta\",\"text\":\" world\"}\n\n")
		io.WriteString(w, "data: {\"type\":\"finish-step\",\"finishReason\":\"stop\",\"usage\":{\"inputTokens\":3,\"outputTokens\":2}}\n\n")
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, err := New("commandcode", reg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	p.config.BaseURL = srv.URL

	resp, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "cc-key"}, &schemas.ChatRequest{
		Model:    "zai-org/GLM-5.1",
		Messages: []schemas.Message{{Role: "user", Content: "hi"}},
	})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}

	// Request was translated to commandcode custom JSON (top-level threadId +
	// params.model), proving the existing converter was used.
	if _, ok := gotBody["threadId"]; !ok {
		t.Errorf("request body missing threadId (openai->commandcode translation not applied): %v", gotBody)
	}
	params, _ := gotBody["params"].(map[string]any)
	if params == nil || params["model"] != "zai-org/GLM-5.1" {
		t.Errorf("request params.model = %v, want zai-org/GLM-5.1", params)
	}
	// Headers/auth.
	if gotAuth != "Bearer cc-key" {
		t.Errorf("Authorization = %q, want Bearer cc-key", gotAuth)
	}
	if gotVersion != "0.25.7" {
		t.Errorf("x-command-code-version = %q, want 0.25.7", gotVersion)
	}
	if gotCLIEnv != "cli" {
		t.Errorf("x-cli-environment = %q, want cli", gotCLIEnv)
	}

	// Response translated back to OpenAI.
	if resp == nil || len(resp.Choices) == 0 {
		t.Fatalf("ChatCompletion response empty: %+v", resp)
	}
	if got := resp.Choices[0].Message.Content; got != "hello world" {
		t.Errorf("aggregated content = %q, want %q", got, "hello world")
	}
	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason = %q, want stop", resp.Choices[0].FinishReason)
	}
}

// TestCommandCodeChatCompletionStream verifies the streaming path emits OpenAI
// chunks translated from commandcode events.
func TestCommandCodeChatCompletionStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"type\":\"text-delta\",\"text\":\"a\"}\n\n")
		io.WriteString(w, "data: {\"type\":\"text-delta\",\"text\":\"b\"}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, _ := New("commandcode", reg)
	p.config.BaseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "cc-key"}, &schemas.ChatRequest{
		Model:    "zai-org/GLM-5.1",
		Messages: []schemas.Message{{Role: "user", Content: "hi"}},
	})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}
	var content strings.Builder
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("unexpected error chunk: %v", chunk.Error.Message)
		}
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
	}
	if content.String() != "ab" {
		t.Errorf("streamed content = %q, want ab", content.String())
	}
}
