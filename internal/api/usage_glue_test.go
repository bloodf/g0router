package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// --- Test fakes for the consumer interfaces (UsageRecorder / PendingTracker / DetailCapture) ---

// fakeUsageRecorder captures Record calls for assertions.
type fakeUsageRecorder struct {
	mu      sync.Mutex
	entries []*UsageEntry
	err     error
}

func (f *fakeUsageRecorder) Record(entry *UsageEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, entry)
	return nil
}

func (f *fakeUsageRecorder) snapshot() []*UsageEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*UsageEntry, len(f.entries))
	copy(out, f.entries)
	return out
}

// fakePendingTracker captures Start/End calls.
type fakePendingTracker struct {
	mu       sync.Mutex
	starts   []trackerCall
	ends     []trackerCall
	hadError bool
}

type trackerCall struct {
	Model, Provider, ConnectionID string
	IsError                       bool
}

func (f *fakePendingTracker) Start(model, provider, connectionID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts = append(f.starts, trackerCall{Model: model, Provider: provider, ConnectionID: connectionID})
}

func (f *fakePendingTracker) End(model, provider, connectionID string, isError bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ends = append(f.ends, trackerCall{Model: model, Provider: provider, ConnectionID: connectionID, IsError: isError})
	if isError {
		f.hadError = true
	}
}

func (f *fakePendingTracker) startCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.starts)
}

// fakeDetailCapture captures Save calls.
type fakeDetailCapture struct {
	mu    sync.Mutex
	items []RequestDetailCapture
}

func (f *fakeDetailCapture) Save(capture RequestDetailCapture) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append(f.items, capture)
	return nil
}

func (f *fakeDetailCapture) Close() error { return nil }

func (f *fakeDetailCapture) snapshot() []RequestDetailCapture {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]RequestDetailCapture, len(f.items))
	copy(out, f.items)
	return out
}

// recordingProvider is a fake messages provider that also satisfies Embedding.
type recordingProvider struct {
	fakeMessagesProvider
	providerName string
	connectionID string
	response     *schemas.ChatResponse
	streamCh     chan *schemas.StreamChunk
	chatErr      *schemas.ProviderError
	streamErr    *schemas.ProviderError
	embedResp    *schemas.EmbeddingResponse
}

func (p *recordingProvider) ChatCompletion(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	return p.response, p.chatErr
}

func (p *recordingProvider) ChatCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return p.streamCh, p.streamErr
}

func (p *recordingProvider) Embedding(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.EmbeddingRequest) (*schemas.EmbeddingResponse, *schemas.ProviderError) {
	if p.embedResp == nil {
		return nil, &schemas.ProviderError{StatusCode: 404, Message: "no embed response", Type: "not_found"}
	}
	return p.embedResp, nil
}

// recordingResolver returns a recording provider keyed by model.
type recordingResolver struct {
	providers map[string]schemas.Provider
}

func (r *recordingResolver) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	p, ok := r.providers[req.Model]
	if !ok {
		return nil, schemas.Key{}, errors.New("unknown model")
	}
	return p, schemas.Key{ID: "conn-1", Provider: p.(*recordingProvider).providerName}, nil
}

func (r *recordingResolver) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	p, ok := r.providers[model]
	if !ok {
		return nil, schemas.Key{}, errors.New("unknown model")
	}
	return p, schemas.Key{ID: "conn-1", Provider: p.(*recordingProvider).providerName}, nil
}

// --- TestChatRecordsUsageNonStream: fake Recorder/Tracker injected via setters;
// POST chat → tracker Start+End called, Recorder.Record got provider/model/
// connection/endpoint="/v1/chat/completions" + tokens from the provider response.
func TestChatRecordsUsageNonStream(t *testing.T) {
	rec := &recordingProvider{
		providerName: "openai",
		connectionID: "conn-1",
		response: &schemas.ChatResponse{
			ID:    "r1",
			Model: "gpt-4",
			Usage: &schemas.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		},
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"gpt-4": rec}}

	recorder := &fakeUsageRecorder{}
	tracker := &fakePendingTracker{}
	detail := &fakeDetailCapture{}

	h := &ChatHandler{router: resolver}
	h.SetUsageRecorder(recorder)
	h.SetPendingTracker(tracker)
	h.SetDetailCapture(detail)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	// Tracker start+end.
	if got := tracker.startCount(); got != 1 {
		t.Errorf("tracker starts = %d, want 1", got)
	}
	if len(tracker.ends) != 1 {
		t.Errorf("tracker ends = %d, want 1", len(tracker.ends))
	}
	if tracker.ends[0].IsError {
		t.Error("tracker end should not be error")
	}

	// Recorder got one entry with correct attribution.
	entries := recorder.snapshot()
	if len(entries) != 1 {
		t.Fatalf("recorder entries = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.Provider != "openai" {
		t.Errorf("entry.Provider = %q, want openai", e.Provider)
	}
	if e.Model != "gpt-4" {
		t.Errorf("entry.Model = %q, want gpt-4", e.Model)
	}
	if e.ConnectionID != "conn-1" {
		t.Errorf("entry.ConnectionID = %q, want conn-1", e.ConnectionID)
	}
	if e.Endpoint != "/v1/chat/completions" {
		t.Errorf("entry.Endpoint = %q, want /v1/chat/completions", e.Endpoint)
	}
	if e.PromptTokens != 100 || e.CompletionTokens != 50 {
		t.Errorf("entry tokens = (%d, %d), want (100, 50)", e.PromptTokens, e.CompletionTokens)
	}
	if e.Status != "ok" {
		t.Errorf("entry.Status = %q, want ok", e.Status)
	}

	// Detail capture called on success.
	details := detail.snapshot()
	if len(details) != 1 {
		t.Fatalf("detail captures = %d, want 1", len(details))
	}
	if details[0].Status != "success" {
		t.Errorf("detail.Status = %q, want success", details[0].Status)
	}
}

// TestChatRecordsUsageStream: stream finishes → Record with accumulated/estimated usage.
func TestChatRecordsUsageStream(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}, Usage: &schemas.Usage{PromptTokens: 10, CompletionTokens: 5}}
	close(ch)

	rec := &recordingProvider{
		providerName: "openai",
		connectionID: "conn-1",
		streamCh:     ch,
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"gpt-4": rec}}

	recorder := &fakeUsageRecorder{}
	tracker := &fakePendingTracker{}
	detail := &fakeDetailCapture{}

	h := &ChatHandler{router: resolver}
	h.SetUsageRecorder(recorder)
	h.SetPendingTracker(tracker)
	h.SetDetailCapture(detail)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	if got := tracker.startCount(); got != 1 {
		t.Errorf("tracker starts = %d, want 1", got)
	}
	entries := recorder.snapshot()
	if len(entries) != 1 {
		t.Fatalf("recorder entries = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.Endpoint != "/v1/chat/completions" {
		t.Errorf("entry.Endpoint = %q, want /v1/chat/completions", e.Endpoint)
	}
	if e.PromptTokens != 10 || e.CompletionTokens != 5 {
		t.Errorf("entry tokens = (%d, %d), want (10, 5)", e.PromptTokens, e.CompletionTokens)
	}

	if got := len(detail.snapshot()); got != 1 {
		t.Errorf("detail captures = %d, want 1", got)
	}
}

// TestChatRecordsErrorStatus: provider error → tracker End(error=true), entry status != "ok".
func TestChatRecordsErrorStatus(t *testing.T) {
	rec := &recordingProvider{
		providerName: "openai",
		connectionID: "conn-1",
		chatErr:      &schemas.ProviderError{StatusCode: 502, Message: "bad gateway", Type: "upstream_error"},
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"gpt-4": rec}}

	recorder := &fakeUsageRecorder{}
	tracker := &fakePendingTracker{}
	detail := &fakeDetailCapture{}

	h := &ChatHandler{router: resolver}
	h.SetUsageRecorder(recorder)
	h.SetPendingTracker(tracker)
	h.SetDetailCapture(detail)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502", ctx.Response.StatusCode())
	}

	if len(tracker.ends) != 1 || !tracker.ends[0].IsError {
		t.Errorf("tracker end not flagged as error: %+v", tracker.ends)
	}
	entries := recorder.snapshot()
	if len(entries) != 1 {
		t.Fatalf("recorder entries = %d, want 1", len(entries))
	}
	if entries[0].Status != "error" {
		t.Errorf("entry.Status = %q, want error", entries[0].Status)
	}
	details := detail.snapshot()
	if len(details) != 1 {
		t.Fatalf("detail captures = %d, want 1", len(details))
	}
	if details[0].Status != "error" {
		t.Errorf("detail.Status = %q, want error", details[0].Status)
	}
}

// TestChatCapturesRequestDetail: fake DetailWriter receives sanitized capture
// on success AND error paths.
func TestChatCapturesRequestDetail(t *testing.T) {
	rec := &recordingProvider{
		providerName: "openai",
		connectionID: "conn-1",
		response: &schemas.ChatResponse{
			ID:    "r1",
			Model: "gpt-4",
			Usage: &schemas.Usage{PromptTokens: 1, CompletionTokens: 1},
		},
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"gpt-4": rec}}
	detail := &fakeDetailCapture{}
	h := &ChatHandler{router: resolver}
	h.SetDetailCapture(detail)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	captures := detail.snapshot()
	if len(captures) != 1 {
		t.Fatalf("detail captures = %d, want 1", len(captures))
	}
	c := captures[0]
	if c.Provider != "openai" || c.Model != "gpt-4" {
		t.Errorf("capture attribution = (%q, %q), want (openai, gpt-4)", c.Provider, c.Model)
	}
	if c.Status != "success" {
		t.Errorf("capture.Status = %q, want success", c.Status)
	}
}

// TestMessagesRecordsUsage: endpoint attribution is /v1/messages.
func TestMessagesRecordsUsage(t *testing.T) {
	rec := &recordingProvider{
		providerName: "anthropic",
		connectionID: "conn-1",
		response: &schemas.ChatResponse{
			ID:    "m1",
			Model: "claude-3",
			Usage: &schemas.Usage{PromptTokens: 20, CompletionTokens: 10},
		},
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"claude-3": rec}}

	recorder := &fakeUsageRecorder{}
	tracker := &fakePendingTracker{}
	detail := &fakeDetailCapture{}

	h := &MessagesHandler{router: resolver, registry: translation.NewRegistry()}
	h.SetUsageRecorder(recorder)
	h.SetPendingTracker(tracker)
	h.SetDetailCapture(detail)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.SetBody([]byte(`{"model":"claude-3","messages":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), string(ctx.Response.Body()))
	}

	if got := tracker.startCount(); got != 1 {
		t.Errorf("tracker starts = %d, want 1", got)
	}
	entries := recorder.snapshot()
	if len(entries) != 1 {
		t.Fatalf("recorder entries = %d, want 1", len(entries))
	}
	if entries[0].Endpoint != "/v1/messages" {
		t.Errorf("entry.Endpoint = %q, want /v1/messages", entries[0].Endpoint)
	}
	if entries[0].Provider != "anthropic" {
		t.Errorf("entry.Provider = %q, want anthropic", entries[0].Provider)
	}
	if len(detail.snapshot()) != 1 {
		t.Error("detail capture not called")
	}
}

// TestEmbeddingsRecordsUsage: endpoint attribution is /v1/embeddings.
func TestEmbeddingsRecordsUsage(t *testing.T) {
	rec := &recordingProvider{
		providerName: "openai",
		connectionID: "conn-1",
		embedResp: &schemas.EmbeddingResponse{
			Object: "list",
			Data:   []schemas.Embedding{{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2}}},
			Model:  "text-embedding-3-small",
			Usage:  &schemas.Usage{PromptTokens: 5, CompletionTokens: 0, TotalTokens: 5},
		},
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"text-embedding-3-small": rec}}

	recorder := &fakeUsageRecorder{}
	tracker := &fakePendingTracker{}
	detail := &fakeDetailCapture{}

	h := &EmbeddingsHandler{router: resolver}
	h.SetUsageRecorder(recorder)
	h.SetPendingTracker(tracker)
	h.SetDetailCapture(detail)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/embeddings")
	ctx.Request.SetBody([]byte(`{"model":"text-embedding-3-small","input":"hi"}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), string(ctx.Response.Body()))
	}

	if got := tracker.startCount(); got != 1 {
		t.Errorf("tracker starts = %d, want 1", got)
	}
	entries := recorder.snapshot()
	if len(entries) != 1 {
		t.Fatalf("recorder entries = %d, want 1", len(entries))
	}
	if entries[0].Endpoint != "/v1/embeddings" {
		t.Errorf("entry.Endpoint = %q, want /v1/embeddings", entries[0].Endpoint)
	}
	if len(detail.snapshot()) != 1 {
		t.Error("detail capture not called")
	}
}

// TestResponsesRecordsUsage: endpoint attribution is /v1/responses.
func TestResponsesRecordsUsage(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 1)
	close(ch)
	rec := &recordingProvider{
		providerName: "openai",
		connectionID: "conn-1",
		streamCh:     ch,
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"gpt-4": rec}}

	recorder := &fakeUsageRecorder{}
	tracker := &fakePendingTracker{}
	detail := &fakeDetailCapture{}

	h := &ResponsesHandler{router: resolver, registry: translation.NewRegistry()}
	h.SetUsageRecorder(recorder)
	h.SetPendingTracker(tracker)
	h.SetDetailCapture(detail)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	entries := recorder.snapshot()
	if len(entries) != 1 {
		t.Fatalf("recorder entries = %d, want 1", len(entries))
	}
	if entries[0].Endpoint != "/v1/responses" {
		t.Errorf("entry.Endpoint = %q, want /v1/responses", entries[0].Endpoint)
	}
}

// TestNoSettersNoGlue verifies that handlers without the setters wired still
// work (no panic, no usage recording) — preserves backward compat.
func TestNoSettersNoGlue(t *testing.T) {
	rec := &recordingProvider{
		providerName: "openai",
		connectionID: "conn-1",
		response: &schemas.ChatResponse{ID: "r1", Model: "gpt-4", Usage: &schemas.Usage{PromptTokens: 1, CompletionTokens: 1}},
	}
	resolver := &recordingResolver{providers: map[string]schemas.Provider{"gpt-4": rec}}
	h := &ChatHandler{router: resolver}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

// TestUsageEntryJSONRoundtrip ensures UsageEntry marshals consistently. The
// actual snake_case serialization to the request_log table is the server-side
// adapter's responsibility; this test pins the Go-side field set so any drift
// is visible here.
func TestUsageEntryJSONRoundtrip(t *testing.T) {
	e := &UsageEntry{
		Provider:         "openai",
		Model:            "gpt-4",
		ConnectionID:     "conn-1",
		Endpoint:         "/v1/chat/completions",
		PromptTokens:     100,
		CompletionTokens: 50,
		Status:           "ok",
		Tokens:           map[string]int64{"prompt_tokens": 100, "completion_tokens": 50},
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{`"Provider":"openai"`, `"Model":"gpt-4"`, `"Endpoint":"/v1/chat/completions"`, `"PromptTokens":100`, `"Status":"ok"`, `"prompt_tokens":100`} {
		if !contains(s, want) {
			t.Errorf("UsageEntry JSON missing %q: %s", want, s)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
