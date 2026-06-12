package api

import (
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

// fakeResponsesResolver captures the request and returns a fixed response.
type fakeResponsesResolver struct {
	captured       *schemas.ChatRequest
	response       *schemas.ChatResponse
	streamCh       chan *schemas.StreamChunk
	providerErr    *schemas.ProviderError
	streamCalled   bool
}

func (f *fakeResponsesResolver) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	f.captured = req
	if f.providerErr != nil {
		return nil, schemas.Key{}, errors.New(f.providerErr.Message)
	}
	return &fakeResponsesProvider{response: f.response, streamCh: f.streamCh, resolver: f}, schemas.Key{}, nil
}

type fakeResponsesProvider struct {
	response     *schemas.ChatResponse
	streamCh     chan *schemas.StreamChunk
	chatCalled   bool
	resolver     *fakeResponsesResolver
}

func (p *fakeResponsesProvider) GetProvider() schemas.ModelProvider { return schemas.ProviderOpenAI }
func (p *fakeResponsesProvider) SetNetworkConfig(_ schemas.NetworkConfig) {}

func (p *fakeResponsesProvider) ListModels(_ *schemas.GatewayContext, _ schemas.Key) (*schemas.ListModelsResponse, *schemas.ProviderError) {
	return nil, nil
}

func (p *fakeResponsesProvider) ChatCompletion(_ *schemas.GatewayContext, _ schemas.Key, req *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	p.chatCalled = true
	return p.response, nil
}

func (p *fakeResponsesProvider) ChatCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, req *schemas.ChatRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	if p.resolver != nil {
		p.resolver.streamCalled = true
	}
	return p.streamCh, nil
}

func (p *fakeResponsesProvider) TextCompletion(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.TextCompletionRequest) (*schemas.TextCompletionResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) TextCompletionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.TextCompletionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) Responses(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ResponsesRequest) (*schemas.ResponsesResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) ResponsesStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.ResponsesRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) Embedding(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.EmbeddingRequest) (*schemas.EmbeddingResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) ImageGeneration(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ImageGenerationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) ImageGenerationStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.ImageGenerationRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) ImageEdit(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ImageEditRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) ImageVariation(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ImageVariationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) Speech(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.SpeechRequest) (*schemas.SpeechResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) SpeechStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.SpeechRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) Transcription(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.TranscriptionRequest) (*schemas.TranscriptionResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) TranscriptionStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, _ schemas.Key, _ *schemas.TranscriptionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) FileUpload(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.FileUploadRequest) (*schemas.FileObject, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) FileList(_ *schemas.GatewayContext, _ schemas.Key) (*schemas.FileListResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) FileRetrieve(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.FileObject, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) FileDelete(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.FileDeleteResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) FileContent(_ *schemas.GatewayContext, _ schemas.Key, _ string) ([]byte, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) BatchCreate(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.BatchCreateRequest) (*schemas.Batch, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) BatchList(_ *schemas.GatewayContext, _ schemas.Key) (*schemas.BatchListResponse, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) BatchRetrieve(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.Batch, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) BatchCancel(_ *schemas.GatewayContext, _ schemas.Key, _ string) (*schemas.Batch, *schemas.ProviderError) {
	return nil, nil
}
func (p *fakeResponsesProvider) CountTokens(_ *schemas.GatewayContext, _ schemas.Key, _ *schemas.ChatRequest) (*schemas.TokenCountResponse, *schemas.ProviderError) {
	return nil, nil
}

func TestResponsesEndpointRejectsInvalidBody(t *testing.T) {
	h := NewResponsesHandler(inference.NewRouter(translation.NewRegistry()))
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(`{not json`))
	h.Handle(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("status = %d, want %d", ctx.Response.StatusCode(), fasthttp.StatusBadRequest)
	}
}

func TestResponsesEndpointTranslatesRequest(t *testing.T) {
	ch := make(chan *schemas.StreamChunk)
	close(ch)
	fake := &fakeResponsesResolver{streamCh: ch}
	h := &ResponsesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{
		"model": "gpt-4",
		"input": [
			{"role": "user", "content": "hi"}
		]
	}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if fake.captured == nil {
		t.Fatal("resolver was never called")
	}
	msgs := fake.captured.Messages
	if len(msgs) < 1 {
		t.Fatalf("len(messages) = %d, want >= 1", len(msgs))
	}
	foundUser := false
	for _, m := range msgs {
		if m.Role == "user" && m.Content == "hi" {
			foundUser = true
			break
		}
	}
	if !foundUser {
		t.Errorf("expected user message with content 'hi', got %+v", msgs)
	}
}

func TestResponsesEndpointStreamsEvents(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	fake := &fakeResponsesResolver{streamCh: ch}
	h := &ResponsesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}
	got := string(ctx.Response.Body())
	if !strings.HasPrefix(got, "event: response.created") {
		t.Errorf("missing response.created event framing; output:\n%s", got)
	}
	if !strings.Contains(got, "event: response.output_text.delta") {
		t.Errorf("missing output_text.delta event framing; output:\n%s", got)
	}
	if !strings.Contains(got, "event: response.completed") {
		t.Errorf("missing response.completed event framing; output:\n%s", got)
	}
	lines := strings.Split(got, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") && !strings.HasPrefix(line, "data: [DONE]") {
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &map[string]any{}); err != nil {
				t.Errorf("non-JSON data line: %q", line)
			}
		}
	}
}

func TestResponsesVKDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-denied", &VKInfo{
		Key:           "vk-denied",
		AllowedModels: []string{"gpt-3.5-turbo"},
		IsActive:      true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 403, reason: "model not allowed for virtual key"})

	ch := make(chan *schemas.StreamChunk)
	close(ch)
	fake := &fakeResponsesResolver{streamCh: ch}
	h := &ResponsesHandler{router: fake, registry: translation.NewRegistry()}
	h.SetVKGate(NewVKGate(resolver, quota))

	body := `{"model":"gpt-4","input":[{"role":"user","content":"hi"}]} `
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403", ctx.Response.StatusCode())
	}
	if fake.streamCalled {
		t.Fatal("provider stream should not be called")
	}
}

func TestResponsesEndpointForcesStreaming(t *testing.T) {
	ch := make(chan *schemas.StreamChunk)
	close(ch)
	fake := &fakeResponsesResolver{streamCh: ch}
	h := &ResponsesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"gpt-4","input":[{"role":"user","content":"hi"}],"stream":false}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	if !fake.streamCalled {
		t.Errorf("expected ChatCompletionStream to be called despite stream:false")
	}
	ct := string(ctx.Response.Header.ContentType())
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}
}
