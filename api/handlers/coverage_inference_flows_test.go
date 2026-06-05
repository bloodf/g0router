package handlers

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

func chatResp() *providers.ChatResponse {
	content := "hi"
	role := "assistant"
	stop := "stop"
	return &providers.ChatResponse{
		ID:      "chatcmpl-x",
		Object:  "chat.completion",
		Model:   "gpt-4o",
		Choices: []providers.Choice{{Message: providers.Message{Role: role, Content: content}, FinishReason: &stop}},
		Usage:   &providers.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
	}
}

// --- Inference handler direct branches ---

func TestInferenceNilEngineAndInvalidJSON(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) { Inference(ctx, nil) })
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil engine = %d, want 503", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) { Inference(ctx, &coverageEngine{}) })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestInferenceSyncSuccess(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`, func(ctx *fasthttp.RequestCtx) {
		Inference(ctx, &coverageEngine{resp: chatResp()})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
}

// --- Messages handler direct branches ---

func TestMessagesNilEngineAndInvalidJSON(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) { Messages(ctx, nil) })
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil engine = %d, want 503", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) { Messages(ctx, &coverageEngine{}) })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestMessagesSyncSuccess(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"model":"claude","messages":[{"role":"user","content":"hi"}]}`, func(ctx *fasthttp.RequestCtx) {
		Messages(ctx, &coverageEngine{resp: chatResp()})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), `"type":"message"`) {
		t.Fatalf("body = %s, want anthropic message shape", body)
	}
}

func TestMessagesDispatchErrorSanitized(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"model":"claude","messages":[{"role":"user","content":"hi"}]}`, func(ctx *fasthttp.RequestCtx) {
		Messages(ctx, &coverageEngine{err: errors.New("sqlite boom")})
	})
	if ctx.Response.StatusCode() < 400 {
		t.Fatalf("status = %d, want >=400; body=%s", ctx.Response.StatusCode(), body)
	}
}

// --- Responses handler direct branches ---

func TestResponsesNilEngineAndInvalidJSON(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) { Responses(ctx, nil) })
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil engine = %d, want 503", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) { Responses(ctx, &coverageEngine{}) })
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestResponsesSyncSuccess(t *testing.T) {
	body := `{"model":"gpt-4o","input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}]}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Responses(ctx, &coverageEngine{resp: chatResp()})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), respBody)
	}
	if !strings.Contains(string(respBody), `"object":"response"`) {
		t.Fatalf("body = %s, want responses shape", respBody)
	}
}

func TestResponsesDispatchErrorSanitized(t *testing.T) {
	body := `{"model":"gpt-4o","input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}]}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Responses(ctx, &coverageEngine{err: errors.New("sqlite boom")})
	})
	if ctx.Response.StatusCode() < 400 {
		t.Fatalf("status = %d, want >=400; body=%s", ctx.Response.StatusCode(), respBody)
	}
}

func TestResponsesStreamingSuccess(t *testing.T) {
	content := "hi"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 2)
	chunks <- providers.StreamChunk{ID: "resp-1", Model: "gpt-4o", Created: 1, Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{Content: &content}}}}
	chunks <- providers.StreamChunk{Choices: []providers.StreamChoice{{FinishReason: &finish}}, Usage: &providers.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2}}
	close(chunks)

	out := captureBodyStream(t, func(ctx *fasthttp.RequestCtx) {
		streamResponses(ctx, &coverageEngine{stream: chunks}, &providers.ChatRequest{Model: "gpt-4o"})
	})
	for _, want := range []string{"response.output_text.delta", "response.output_text.done", "response.completed", "[DONE]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stream missing %q:\n%s", want, out)
		}
	}
}

func TestStreamResponsesDispatchError(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		streamResponses(ctx, &coverageEngine{streamErr: errors.New("sqlite boom")}, &providers.ChatRequest{Model: "gpt-4o"})
	})
	if ctx.Response.StatusCode() < 400 {
		t.Fatalf("status = %d, want >=400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestResponsesStreamingEmitsToolCalls(t *testing.T) {
	id := "call_abc"
	name := "get_weather"
	argsA := `{"city":`
	argsB := `"sf"}`
	finish := "tool_calls"
	chunks := make(chan providers.StreamChunk, 3)
	chunks <- providers.StreamChunk{ID: "resp-1", Model: "gpt-4o", Created: 1, Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{ID: id, Type: "function", Function: providers.ToolCallFunc{Name: name, Arguments: argsA}}}}}}}
	chunks <- providers.StreamChunk{Choices: []providers.StreamChoice{{Delta: providers.StreamDelta{ToolCalls: []providers.ToolCall{{Function: providers.ToolCallFunc{Arguments: argsB}}}}}}}
	chunks <- providers.StreamChunk{Choices: []providers.StreamChoice{{FinishReason: &finish}}, Usage: &providers.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2}}
	close(chunks)

	out := captureBodyStream(t, func(ctx *fasthttp.RequestCtx) {
		streamResponses(ctx, &coverageEngine{stream: chunks}, &providers.ChatRequest{Model: "gpt-4o"})
	})
	for _, want := range []string{
		"event: response.function_call_arguments.delta",
		`"item_id":"call_abc"`,
		`"delta":"{\"city\":"`,
		`"delta":"\"sf\"}"`,
		"response.completed",
		`"type":"function_call"`,
		`"call_id":"call_abc"`,
		`"name":"get_weather"`,
		`"arguments":"{\"city\":\"sf\"}"`,
		"[DONE]",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stream missing %q:\n%s", want, out)
		}
	}
}

func TestWriteStreamMarshalErrorEmitsTerminalEvent(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	writeStreamMarshalError(w)
	_ = w.Flush()
	got := buf.String()
	if !strings.Contains(got, "stream_encoding_error") || !strings.Contains(got, "server_error") {
		t.Fatalf("terminal error event missing: %s", got)
	}
}

func TestWriteResponsesFunctionCallDeltaShape(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := writeResponsesFunctionCallDelta(w, "call_1", `{"a":1}`, 3); err != nil {
		t.Fatalf("writeResponsesFunctionCallDelta: %v", err)
	}
	_ = w.Flush()
	got := buf.String()
	for _, want := range []string{"event: response.function_call_arguments.delta", `"item_id":"call_1"`, `"sequence_number":3`} {
		if !strings.Contains(got, want) {
			t.Fatalf("delta event missing %q: %s", want, got)
		}
	}
}

// responsesToolCallAccumulator stitches fragmented deltas: id/name on the first
// fragment, arguments continuing on later fragments without ids.
func TestResponsesToolCallAccumulatorStitchesFragments(t *testing.T) {
	acc := newResponsesToolCallAccumulator()
	acc.add(providers.ToolCall{ID: "c1", Function: providers.ToolCallFunc{Name: "f", Arguments: "{"}})
	acc.add(providers.ToolCall{Function: providers.ToolCallFunc{Arguments: "}"}})
	outputs := acc.outputs()
	if len(outputs) != 1 {
		t.Fatalf("outputs = %d, want 1", len(outputs))
	}
	if outputs[0].CallID != "c1" || outputs[0].Name != "f" || outputs[0].Arguments != "{}" {
		t.Fatalf("output = %+v, want c1/f/{}", outputs[0])
	}
}

// --- write* marshal failures ---

func TestWriteErrorAndOpenAIErrorAreWellFormed(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	writeError(ctx, fasthttp.StatusBadRequest, "bad")
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest || !strings.Contains(string(ctx.Response.Body()), "bad") {
		t.Fatalf("writeError body = %s", ctx.Response.Body())
	}
	ctx = &fasthttp.RequestCtx{}
	writeOpenAIError(ctx, fasthttp.StatusTooManyRequests, "limit", "rate_limit_error", "rate_limited")
	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests || !strings.Contains(string(ctx.Response.Body()), "rate_limit_error") {
		t.Fatalf("writeOpenAIError body = %s", ctx.Response.Body())
	}
}

// --- Providers handler with empty models ---

func TestProvidersListModelsEmptyForProvider(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, handlerModelSource{models: nil}, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProvidersListModelsSourceError(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, handlerModelSource{err: errors.New("sqlite boom")}, "openai")
	})
	if ctx.Response.StatusCode() < 400 {
		t.Fatalf("status = %d, want >=400; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoInternalDetail(t, body)
}

// --- UsageQuota fallback / missing provider ---

func TestUsageQuotaMissingProviderInPath(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/")
	UsageQuota(ctx, nil, nil, providers.Key{})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing provider = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestUsageQuotaFetcherErrorIsBadGateway(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage/quota/openai")
	UsageQuota(ctx, nil, map[providers.ModelProvider]usage.QuotaFetcher{
		providers.ProviderOpenAI: &fakeQuotaFetcher{err: errors.New("upstream down")},
	}, providers.Key{})
	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("fetcher error = %d, want 502; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	assertNoInternalDetail(t, ctx.Response.Body())
}

// --- OAuth full flows using fakeOAuthFlow ---

func TestOAuthStartFullFlow(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	ctx.Request.SetBodyString(`{"account_label":"work"}`)
	OAuthStart(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("start status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if !flow.started {
		t.Fatal("flow.Start was not invoked")
	}
}

func TestOAuthStartFlowStartError(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic"), startErr: errors.New("boom")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	OAuthStart(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("start error = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestOAuthPollFullFlowPersistsConnection(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	// Seed a session so the poll consume+persist branch runs.
	start := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	start.Request.SetBodyString(`{"account_label":"work"}`)
	OAuthStart(start, s, flows)

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/poll?session_id=session-123")
	OAuthPoll(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("poll status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if !strings.Contains(string(ctx.Response.Body()), "connection") {
		t.Fatalf("poll body = %s, want connection", ctx.Response.Body())
	}
}

func TestOAuthExchangeFullFlow(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	start := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	start.Request.SetBodyString(`{}`)
	OAuthStart(start, s, flows)

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{"state":"session-123","code":"auth-code"}`)
	OAuthExchange(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("exchange status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.exCode != "auth-code" {
		t.Fatalf("exchange code = %q, want auth-code", flow.exCode)
	}
}

func TestOAuthCallbackFullFlow(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	start := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	start.Request.SetBodyString(`{}`)
	OAuthStart(start, s, flows)

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/callback?code=auth-code&state=session-123")
	OAuthCallback(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("callback status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestExchangeOAuthExchangeFailureIsBadGateway(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic"), exErr: errors.New("upstream")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	start := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	start.Request.SetBodyString(`{}`)
	OAuthStart(start, s, flows)

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{"state":"session-123","code":"auth-code"}`)
	OAuthExchange(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("exchange failure = %d, want 502", ctx.Response.StatusCode())
	}
}

// --- parseTimeArg valid path via Usage handler ---

func TestUsageParsesValidDateRange(t *testing.T) {
	s := newHandlerStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?from=2026-06-01T00:00:00Z&to=2026-06-02T00:00:00Z")
	Usage(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// --- MCPOAuthComplete completer exchange failure -> bad gateway ---

func TestMCPOAuthCompleteExchangeFailure(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{err: errors.New("exchange failed")}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"callback_url":"https://x?code=c&state=s"}`, func(ctx *fasthttp.RequestCtx) {
		MCPOAuthComplete(ctx, completer, nil, nil, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), body)
	}
}

var _ = mcp.ErrOAuthFlowNotFound
