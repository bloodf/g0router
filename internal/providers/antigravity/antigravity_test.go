package antigravity

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// TestNewRejectsWrongFormat verifies the constructor enforces the catalog
// Format.
func TestNewRejectsWrongFormat(t *testing.T) {
	reg := translation.NewRegistry()
	if _, err := New("openai", reg); err == nil {
		t.Fatal("New(openai) error = nil, want error (format mismatch)")
	}
}

// TestBackendForModel verifies the per-model backend selection: claude models →
// claude backend, gpt-oss models → gpt-oss backend, everything else → gemini
// (executors/antigravity.js transformRequest dispatches by isClaudeModel).
func TestBackendForModel(t *testing.T) {
	cases := map[string]string{
		"claude-sonnet-4-6":        "claude",
		"claude-opus-4-6-thinking": "claude",
		"gpt-oss-120b-medium":      "gpt-oss",
		"gemini-3-flash-agent":     "gemini",
		"gemini-pro-agent":         "gemini",
	}
	for model, want := range cases {
		if got := backendForModel(model); got != want {
			t.Errorf("backendForModel(%q) = %q, want %q", model, got, want)
		}
	}
}

// TestBuildURLFallbackOrdering verifies the fallback URL ordering: index 0 is the
// primary daily-cloudcode-pa host, index 1 is the sandbox host, and the action
// suffix differs by stream mode (providers.js:106-108, antigravity.js:26-31).
func TestBuildURLFallbackOrdering(t *testing.T) {
	reg := translation.NewRegistry()
	p, err := New("antigravity", reg)
	if err != nil {
		t.Fatalf("New(antigravity) error: %v", err)
	}

	if got := p.buildURL(0, true); got != "https://daily-cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse" {
		t.Errorf("buildURL(0,stream) = %q", got)
	}
	if got := p.buildURL(1, true); got != "https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:streamGenerateContent?alt=sse" {
		t.Errorf("buildURL(1,stream) = %q", got)
	}
	if got := p.buildURL(0, false); got != "https://daily-cloudcode-pa.googleapis.com/v1internal:generateContent" {
		t.Errorf("buildURL(0,non-stream) = %q", got)
	}
	if got := p.fallbackCount(); got != 2 {
		t.Errorf("fallbackCount() = %d, want 2", got)
	}
}

// TestCloakToolsUnavailableFilter verifies the PAR-MCP-060 unavailable-tool
// ride-along: client tools are renamed with the _ide suffix, the AG decoy tools
// (marked "This tool is currently unavailable.") are injected, and native AG tool
// names are preserved without a suffix (antigravity.js cloakTools + AG_DECOY_TOOLS).
func TestCloakToolsUnavailableFilter(t *testing.T) {
	clientTools := []map[string]any{
		{"name": "my_search", "description": "search the web"},
		{"name": "run_command", "description": "run a shell command"}, // native AG name → preserved
	}
	cloaked, nameMap := cloakTools(clientTools)

	// my_search is renamed with the _ide suffix.
	if nameMap["my_search_ide"] != "my_search" {
		t.Errorf("name map missing my_search_ide -> my_search; got %#v", nameMap)
	}
	// run_command is a native AG name → preserved (no suffix, not in map).
	if _, ok := nameMap["run_command_ide"]; ok {
		t.Error("native AG name run_command should not be suffixed")
	}

	names := map[string]string{}
	for _, tl := range cloaked {
		name, _ := tl["name"].(string)
		desc, _ := tl["description"].(string)
		names[name] = desc
	}
	if _, ok := names["my_search_ide"]; !ok {
		t.Error("cloaked tools missing my_search_ide")
	}
	// At least one decoy tool present and marked unavailable.
	if names["search_web"] != "This tool is currently unavailable." {
		t.Errorf("decoy search_web description = %q, want unavailable marker", names["search_web"])
	}
	// Decoy count: all 21 AG decoy tools are injected (deduped by name).
	unavailable := 0
	for _, d := range names {
		if d == "This tool is currently unavailable." {
			unavailable++
		}
	}
	if unavailable < 20 {
		t.Errorf("decoy unavailable tool count = %d, want >= 20", unavailable)
	}
}

// TestCloakToolsNoTools verifies an empty tool list yields no cloaking.
func TestCloakToolsNoTools(t *testing.T) {
	cloaked, nameMap := cloakTools(nil)
	if cloaked != nil {
		t.Errorf("cloakTools(nil) tools = %#v, want nil", cloaked)
	}
	if nameMap != nil {
		t.Errorf("cloakTools(nil) nameMap = %#v, want nil", nameMap)
	}
}

// geminiSSEBody returns a canned Gemini-format SSE body (the antigravity wire).
func geminiSSEBody() string {
	return "data: {\"response\":{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"Hi there\"}]}}],\"modelVersion\":\"gemini-3-flash\",\"responseId\":\"r1\"}}\n\n" +
		"data: {\"response\":{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[]},\"finishReason\":\"STOP\"}],\"modelVersion\":\"gemini-3-flash\",\"responseId\":\"r1\"}}\n\n"
}

// TestChatCompletionStreamGemini verifies the executor POSTs to the v1internal
// stream URL, sends a bearer token, and translates the Gemini SSE response to
// OpenAI chunks via the registry.
func TestChatCompletionStreamGemini(t *testing.T) {
	var gotPath, gotAuth, gotUA, gotSource string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotSource = r.Header.Get("x-request-source")
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, geminiSSEBody())
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, _ := New("antigravity", reg)
	p.urlOverride = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "ag-token"},
		&schemas.ChatRequest{Model: "gemini-3-flash-agent", Messages: []schemas.Message{{Role: "user", Content: "hi"}}})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}
	var content string
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("error chunk: %v", chunk.Error.Message)
		}
		for _, c := range chunk.Choices {
			content += c.Delta.Content
		}
	}
	if content != "Hi there" {
		t.Errorf("content = %q, want %q", content, "Hi there")
	}
	if gotAuth != "Bearer ag-token" {
		t.Errorf("Authorization = %q, want Bearer ag-token", gotAuth)
	}
	if !strings.Contains(gotPath, "streamGenerateContent") {
		t.Errorf("path = %q, want streamGenerateContent", gotPath)
	}
	if !strings.Contains(gotUA, "antigravity/1.107") {
		t.Errorf("User-Agent = %q, want antigravity/1.107", gotUA)
	}
	if gotSource != "local" {
		t.Errorf("x-request-source = %q, want local", gotSource)
	}
}

// TestChatCompletionStreamFallbackURL verifies that when the primary URL returns
// a 5xx, the executor retries on the sandbox fallback URL.
func TestChatCompletionStreamFallbackURL(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"unavailable"}`))
			return
		}
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, geminiSSEBody())
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, _ := New("antigravity", reg)
	// Both fallback slots point at the same test server; the first attempt 503s,
	// the second (fallback) succeeds.
	p.urlOverride = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "t"},
		&schemas.ChatRequest{Model: "gemini-3-flash-agent", Messages: []schemas.Message{{Role: "user", Content: "hi"}}})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("error chunk: %v", chunk.Error.Message)
		}
	}
	if hits < 2 {
		t.Errorf("upstream hits = %d, want >= 2 (fallback retried)", hits)
	}
}
