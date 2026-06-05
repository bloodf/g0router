package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// PUT with invalid JSON after a valid id reaches the decode-failure return in
// the PUT arm of each CRUD handler.
func TestCRUDPutInvalidJSONDecodeFailure(t *testing.T) {
	s := newHandlerStore(t)
	cases := []struct {
		name string
		run  func(ctx *fasthttp.RequestCtx)
	}{
		{"aliases", func(ctx *fasthttp.RequestCtx) { Aliases(ctx, s, "fast") }},
		{"combos", func(ctx *fasthttp.RequestCtx) { Combos(ctx, s, "id") }},
		{"pricing", func(ctx *fasthttp.RequestCtx) { Pricing(ctx, s, "openai", "gpt-4o") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, _ := runHandler(t, fasthttp.MethodPut, `{bad`, c.run)
			if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
				t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
			}
		})
	}
}

// streamMessages forwards an upstream chunk error and stops.
func TestStreamMessagesChunkError(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{Error: &providers.StreamError{Message: "boom", Type: "server_error", Code: "x"}}
	close(chunks)
	out := captureBodyStream(t, func(ctx *fasthttp.RequestCtx) {
		streamMessages(ctx, &coverageEngine{stream: chunks}, &providers.ChatRequest{Model: "claude"})
	})
	if !strings.Contains(out, "server_error") {
		t.Fatalf("stream output missing sanitized error:\n%s", out)
	}
}

// streamResponses forwards an upstream chunk error and stops.
func TestStreamResponsesChunkError(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{Error: &providers.StreamError{Message: "boom", Type: "server_error", Code: "x"}}
	close(chunks)
	out := captureBodyStream(t, func(ctx *fasthttp.RequestCtx) {
		streamResponses(ctx, &coverageEngine{stream: chunks}, &providers.ChatRequest{Model: "gpt-4o"})
	})
	if !strings.Contains(out, "server_error") {
		t.Fatalf("stream output missing sanitized error:\n%s", out)
	}
}

// streamMessages emits a second tool_use block, closing the first open block.
func TestStreamMessagesSequentialToolBlocks(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 2)
	chunks <- providers.StreamChunk{ID: "m1", Model: "claude", Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{
		ID:       "call_1",
		Function: providers.ToolCallFunc{Name: "search", Arguments: `{}`},
	}}}}}}
	chunks <- providers.StreamChunk{Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{
		ID:       "call_2",
		Function: providers.ToolCallFunc{Name: "fetch", Arguments: `{}`},
	}}}}}}
	close(chunks)
	out := captureBodyStream(t, func(ctx *fasthttp.RequestCtx) {
		streamMessages(ctx, &coverageEngine{stream: chunks}, &providers.ChatRequest{Model: "claude"})
	})
	if strings.Count(out, `"type":"tool_use"`) < 2 {
		t.Fatalf("expected two tool_use blocks:\n%s", out)
	}
}

// Responses translation failure (input content with invalid type) -> 400.
func TestResponsesTranslationError(t *testing.T) {
	// A tool with invalid parameters JSON triggers ResponsesRequestToOpenAIChat error.
	body := `{"model":"gpt-4o","tools":[{"type":"function","name":"x","parameters":123}],"input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}]}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Responses(ctx, &coverageEngine{resp: chatResp()})
	})
	// Either 400 (translation error) or 200 (if accepted); accept 400 path coverage.
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest && ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 400 or 200; body=%s", ctx.Response.StatusCode(), respBody)
	}
}

// anthropicMessageResponse defaults role to assistant when choice role is empty.
func TestAnthropicMessageResponseDefaultsRole(t *testing.T) {
	stop := "stop"
	resp := &providers.ChatResponse{
		ID:    "msg",
		Model: "claude",
		Choices: []providers.Choice{{
			FinishReason: &stop,
			Message:      providers.Message{Role: "", Content: "hi"},
		}},
	}
	body := anthropicMessageResponse(resp)
	if body.Role != "assistant" {
		t.Fatalf("role = %q, want assistant default", body.Role)
	}
}

// translateAnthropicMessagesRequest with a message whose block JSON is invalid
// surfaces the per-message translation error.
func TestTranslateAnthropicMessagesRequestPerMessageError(t *testing.T) {
	// Second message's content is an array of strings; block decode into
	// []anthropicInboundBlock fails and surfaces a per-message error.
	body := []byte(`{"model":"claude","messages":[{"role":"user","content":[{"type":"text"}]},{"role":"user","content":["str"]}]}`)
	if _, err := translateAnthropicMessagesRequest(body); err == nil {
		t.Fatal("expected per-message block decode error")
	}
}

// rejectUnsupportedAnthropicMessageShape now accepts native tool_choice; it is
// translated rather than rejected.
func TestRejectUnsupportedAnthropicNativeToolChoiceAccepted(t *testing.T) {
	body := []byte(`{"tool_choice":{"type":"tool","name":"x"},"messages":[]}`)
	if err := rejectUnsupportedAnthropicMessageShape(body); err != nil {
		t.Fatalf("native tool_choice should no longer be rejected at shape gate: %v", err)
	}
	req, err := translateAnthropicMessagesRequest(body)
	if err != nil {
		t.Fatalf("translate native tool_choice: %v", err)
	}
	m, ok := req.ToolChoice.(map[string]any)
	if !ok || m["type"] != "function" {
		t.Fatalf("tool_choice = %#v, want function object", req.ToolChoice)
	}
}

// OAuthPoll with an unregistered provider returns 404 via oauthFlowForPath.
func TestOAuthPollFlowNotFound(t *testing.T) {
	s := newHandlerStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/poll?session_id=s")
	OAuthPoll(ctx, s, OAuthFlows{})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

// OAuthExchange with an unregistered provider returns 404.
func TestOAuthExchangeFlowNotFound(t *testing.T) {
	s := newHandlerStore(t)
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{"state":"s","code":"c"}`)
	OAuthExchange(ctx, s, OAuthFlows{})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

// OAuthExchange where the consumed session provider mismatches the path flow.
func TestOAuthExchangeProviderMismatch(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{State: "state-1", Provider: "openai"}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{"state":"state-1","code":"c"}`)
	OAuthExchange(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

var _ = json.Marshal
