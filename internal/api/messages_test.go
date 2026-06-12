package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// fakeMessagesResolver captures the minimal request passed to ResolveForModel and
// stores a ref to the provider so tests can inspect ChatCompletion arguments.
type fakeMessagesResolver struct {
	captured    *schemas.ChatRequest
	lastProv    *fakeMessagesProvider
	response    *schemas.ChatResponse
	streamCh    chan *schemas.StreamChunk
	providerErr *schemas.ProviderError
}

func (f *fakeMessagesResolver) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	f.captured = req
	if f.providerErr != nil {
		return nil, schemas.Key{}, errors.New(f.providerErr.Message)
	}
	p := &fakeMessagesProvider{response: f.response, streamCh: f.streamCh}
	f.lastProv = p
	return p, schemas.Key{}, nil
}

type fakeMessagesProvider struct {
	response    *schemas.ChatResponse
	streamCh    chan *schemas.StreamChunk
	chatCalled  bool
	capturedReq *schemas.ChatRequest
}

func (p *fakeMessagesProvider) GetProvider() schemas.ModelProvider { return schemas.ProviderOpenAI }
func (p *fakeMessagesProvider) SetNetworkConfig(_ schemas.NetworkConfig) {}

func (p *fakeMessagesProvider) ListModels(_ *schemas.GatewayContext, _ schemas.Key) (*schemas.ListModelsResponse, *schemas.ProviderError) {
	return nil, nil
}

func (p *fakeMessagesProvider) ChatCompletion(_ *schemas.GatewayContext, _ schemas.Key, req *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	p.chatCalled = true
	p.capturedReq = req
	return p.response, nil
}

func (p *fakeMessagesProvider) ChatCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, req *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return p.streamCh, nil
}

func (p *fakeMessagesProvider) TextCompletion(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.TextCompletionRequest) (*schemas.TextCompletionResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) TextCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.TextCompletionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) Responses(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ResponsesRequest) (*schemas.ResponsesResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) ResponsesStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.ResponsesRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) Embedding(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.EmbeddingRequest) (*schemas.EmbeddingResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) ImageGeneration(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ImageGenerationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) ImageGenerationStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.ImageGenerationRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) ImageEdit(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ImageEditRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) ImageVariation(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ImageVariationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) Speech(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.SpeechRequest) (*schemas.SpeechResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) SpeechStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.SpeechRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) Transcription(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.TranscriptionRequest) (*schemas.TranscriptionResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) TranscriptionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.TranscriptionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) FileUpload(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.FileUploadRequest) (*schemas.FileObject, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) FileList(_ *schemas.GatewayContext, _ schemas.Key) (*schemas.FileListResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) FileRetrieve(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.FileObject, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) FileDelete(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.FileDeleteResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) FileContent(_ *schemas.GatewayContext, _ schemas.Key, _ string) ([]byte, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) BatchCreate(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.BatchCreateRequest) (*schemas.Batch, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) BatchList(_ *schemas.GatewayContext, _ schemas.Key) (*schemas.BatchListResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) BatchRetrieve(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.Batch, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) BatchCancel(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.Batch, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeMessagesProvider) CountTokens(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ChatRequest) (*schemas.TokenCountResponse, *schemas.ProviderError) {
	return nil, nil
}

func TestMessagesHandlerBadJSON(t *testing.T) {
	h := NewMessagesHandler(inference.NewRouter(translation.NewRegistry()))
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.SetBody([]byte(`{not json`))
	h.Handle(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("status = %d, want %d", ctx.Response.StatusCode(), fasthttp.StatusBadRequest)
	}
}

func TestMessagesHandlerTranslatesClaudeBody(t *testing.T) {
	fake := &fakeMessagesResolver{}
	h := &MessagesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{
		"model": "claude-opus-4",
		"system": "you are helpful",
		"messages": [
			{"role": "user", "content": "hi"},
			{"role": "assistant", "content": [{"type": "tool_use", "id": "tu_1", "name": "Read", "input": {"file_path": "/tmp/a"}}]}
		]
	}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if fake.captured == nil {
		t.Fatal("resolver was never called")
	}
	// After restructuring, ResolveForModel receives a minimal req (just model).
	// Translation output is passed to ChatCompletion; check via lastProv.capturedReq.
	if fake.lastProv == nil || fake.lastProv.capturedReq == nil {
		t.Fatal("ChatCompletion not called")
	}
	msgs := fake.lastProv.capturedReq.Messages
	if len(msgs) < 3 {
		t.Fatalf("len(messages) = %d, want >= 3 (system + user + assistant + tool placeholder)", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "you are helpful" {
		t.Errorf("system message = %+v", msgs[0])
	}
	if msgs[2].Role != "assistant" || len(msgs[2].ToolCalls) != 1 || msgs[2].ToolCalls[0].Function.Name != "Read" {
		t.Errorf("assistant message = %+v", msgs[2])
	}
}

func TestMessagesHandlerNonStreamingPassthrough(t *testing.T) {
	want := &schemas.ChatResponse{
		ID:      "chatcmpl-1",
		Object:  "chat.completion",
		Created: 123,
		Model:   "claude-opus-4",
		Choices: []schemas.Choice{
			{Index: 0, Message: &schemas.Message{Role: "assistant", Content: "hello"}, FinishReason: "stop"},
		},
	}
	fake := &fakeMessagesResolver{response: want}
	h := &MessagesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"claude-opus-4","messages":[{"role":"user","content":"hi"}]}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}
	got := ctx.Response.Body()
	wantBytes, _ := json.Marshal(want)
	if !bytes.Equal(got, wantBytes) {
		t.Errorf("body = %s, want %s", got, wantBytes)
	}
}

func TestMessagesHandlerStreamingFraming(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	fake := &fakeMessagesResolver{streamCh: ch}
	h := &MessagesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"claude-opus-4","messages":[{"role":"user","content":"hi"}],"stream":true}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}
	got := string(ctx.Response.Body())
	lines := strings.Split(got, "\n")
	if !strings.HasPrefix(got, "event: message_start") {
		t.Errorf("missing message_start event framing; output:\n%s", got)
	}
	if !strings.Contains(got, "event: content_block_delta") {
		t.Errorf("missing content_block_delta event framing; output:\n%s", got)
	}
	if !strings.Contains(got, "event: message_stop") {
		t.Errorf("missing message_stop event framing; output:\n%s", got)
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") && !strings.HasPrefix(line, "data: [DONE]") {
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &map[string]any{}); err != nil {
				t.Errorf("non-JSON data line: %q", line)
			}
		}
	}
}

func TestMessagesHandlerStreamingAbortsOnErrorChunk(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{Error: &schemas.ProviderError{Message: "boom", Type: "stream_error"}}
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "ignored"}}}}
	close(ch)

	fake := &fakeMessagesResolver{streamCh: ch}
	h := &MessagesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"claude-opus-4","messages":[{"role":"user","content":"hi"}],"stream":true}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	got := string(ctx.Response.Body())
	if strings.Contains(got, "ignored") {
		t.Errorf("error chunk did not abort stream; output:\n%s", got)
	}
	if strings.Contains(got, "boom") {
		t.Errorf("error leaked to client; output:\n%s", got)
	}
}

func strPtr(s string) *string { return &s }

// captureMessagesProvider captures ChatCompletion requests for assertions.
// It optionally implements NativeFormat() and ThinkingMode().
type captureMessagesProvider struct {
	fakeMessagesProvider
	capturedReq  *schemas.ChatRequest
	nativeFormat string
	thinkingMode string
}

func (p *captureMessagesProvider) NativeFormat() string { return p.nativeFormat }
func (p *captureMessagesProvider) ThinkingMode() string { return p.thinkingMode }

func (p *captureMessagesProvider) ChatCompletion(_ *schemas.GatewayContext, _ schemas.Key, req *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	p.capturedReq = req
	return &schemas.ChatResponse{}, nil
}

// captureMessagesResolver returns a captureMessagesProvider.
type captureMessagesResolver struct {
	prov *captureMessagesProvider
}

func (r *captureMessagesResolver) ResolveForModel(_ *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	return r.prov, schemas.Key{}, nil
}

// TestNativePassthroughSkipsTranslation verifies PAR-ROUTE-041: when the provider's
// NativeFormat matches the detected source format, translation is skipped and the
// request is rebuilt directly from the original body.
func TestNativePassthroughSkipsTranslation(t *testing.T) {
	prov := &captureMessagesProvider{nativeFormat: "claude"}
	h := &MessagesHandler{
		router:   &captureMessagesResolver{prov: prov},
		registry: translation.NewRegistry(),
	}

	// Claude body: translation would extract "system" as messages[0].
	// Native passthrough: system key is NOT in ChatRequest schema → ignored;
	// messages[0] stays the user message.
	body := `{"model":"claude-3","system":"you are helpful","messages":[{"role":"user","content":"hello"}],"stream":false}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if prov.capturedReq == nil {
		t.Fatal("ChatCompletion not called")
	}
	msgs := prov.capturedReq.Messages
	if len(msgs) == 0 {
		t.Fatal("captured req has no messages")
	}
	if msgs[0].Role == "system" {
		t.Error("native passthrough should skip translation; got system injected as messages[0]")
	}
	if msgs[0].Role != "user" {
		t.Errorf("messages[0].role = %q, want user (native passthrough preserves original body)", msgs[0].Role)
	}
}

// TestThinkingOverrideInjected verifies PAR-ROUTE-042: when a provider implements
// ThinkingMode() returning "on", the handler injects thinking config into the request
// even when the client body omits it.
func TestThinkingOverrideInjected(t *testing.T) {
	prov := &captureMessagesProvider{thinkingMode: "on"}
	h := &MessagesHandler{
		router:   &captureMessagesResolver{prov: prov},
		registry: translation.NewRegistry(),
	}

	body := `{"model":"claude-3","messages":[{"role":"user","content":"hello"}],"stream":false}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if prov.capturedReq == nil {
		t.Fatal("ChatCompletion not called")
	}
	if prov.capturedReq.Thinking == nil {
		t.Fatal("thinking config not injected (want non-nil Thinking)")
	}
	if prov.capturedReq.Thinking.Type != "enabled" {
		t.Errorf("thinking.type = %q, want enabled", prov.capturedReq.Thinking.Type)
	}
	if prov.capturedReq.Thinking.BudgetTokens != 10000 {
		t.Errorf("thinking.budget_tokens = %d, want 10000", prov.capturedReq.Thinking.BudgetTokens)
	}
}

// TestBypassWarmupShortCircuits verifies that a "Warmup" message with claude-cli User-Agent
// returns a fake bypass response without calling the provider (PAR-ROUTE-034).
func TestBypassWarmupShortCircuits(t *testing.T) {
	fake := &fakeMessagesResolver{}
	h := &MessagesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"claude-opus-4","messages":[{"role":"user","content":[{"type":"text","text":"Warmup"}]}],"stream":false}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.Header.Set("User-Agent", "claude-cli/1.0")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if fake.captured != nil {
		t.Fatal("provider was called for bypass request — should have short-circuited")
	}
	if len(ctx.Response.Body()) == 0 {
		t.Fatal("bypass response body is empty")
	}
}

// TestBypassTitleSkip verifies that an assistant message with content "{" (title extraction pattern)
// short-circuits the provider with a bypass response (PAR-ROUTE-034).
func TestBypassTitleSkip(t *testing.T) {
	fake := &fakeMessagesResolver{}
	h := &MessagesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"claude-opus-4","messages":[{"role":"user","content":"hi"},{"role":"assistant","content":[{"type":"text","text":"{"}]}],"stream":false}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/messages")
	ctx.Request.Header.Set("User-Agent", "claude-cli/1.0")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if fake.captured != nil {
		t.Fatal("provider was called for bypass request — should have short-circuited")
	}
	if len(ctx.Response.Body()) == 0 {
		t.Fatal("bypass response body is empty")
	}
}
