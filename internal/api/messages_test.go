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

// fakeMessagesResolver captures the request and returns a fixed response.
type fakeMessagesResolver struct {
	captured    *schemas.ChatRequest
	response    *schemas.ChatResponse
	streamCh    chan *schemas.StreamChunk
	providerErr *schemas.ProviderError
}

func (f *fakeMessagesResolver) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	f.captured = req
	if f.providerErr != nil {
		return nil, schemas.Key{}, errors.New(f.providerErr.Message)
	}
	return &fakeMessagesProvider{response: f.response, streamCh: f.streamCh}, schemas.Key{}, nil
}

type fakeMessagesProvider struct {
	response   *schemas.ChatResponse
	streamCh   chan *schemas.StreamChunk
	chatCalled bool
}

func (p *fakeMessagesProvider) GetProvider() schemas.ModelProvider { return schemas.ProviderOpenAI }
func (p *fakeMessagesProvider) SetNetworkConfig(_ schemas.NetworkConfig) {}

func (p *fakeMessagesProvider) ListModels(_ *schemas.GatewayContext, _ schemas.Key) (*schemas.ListModelsResponse, *schemas.ProviderError) {
	return nil, nil
}

func (p *fakeMessagesProvider) ChatCompletion(_ *schemas.GatewayContext, _ schemas.Key, req *schemas.ChatRequest) (*schemas.ChatResponse, *schemas.ProviderError) {
	p.chatCalled = true
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
	msgs := fake.captured.Messages
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
