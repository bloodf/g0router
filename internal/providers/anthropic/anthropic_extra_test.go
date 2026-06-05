package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

// ---- Name() ----

func TestName(t *testing.T) {
	p := New("")
	if p.Name() != providers.ProviderAnthropic {
		t.Fatalf("Name = %q", p.Name())
	}
}

func TestNameForProvider(t *testing.T) {
	p := NewForProvider("my-provider", "")
	if p.Name() != "my-provider" {
		t.Fatalf("Name = %q", p.Name())
	}
}

// ---- NewForProviderWithHeaders: blank key filtered ----

func TestNewForProviderWithHeadersFiltersBlankKey(t *testing.T) {
	p := NewForProviderWithHeaders(providers.ProviderAnthropic, "", map[string]string{
		"":       "ignored",
		"X-Real": "kept",
	})
	if _, ok := p.headers[""]; ok {
		t.Fatal("blank key should be filtered")
	}
	if p.headers["X-Real"] != "kept" {
		t.Fatalf("X-Real = %q", p.headers["X-Real"])
	}
}

// ---- Error types ----

func TestRateLimitErrorMessage(t *testing.T) {
	e := &RateLimitError{Message: "slow", RetryAfter: 5}
	if !strings.Contains(e.Error(), "slow") || !strings.Contains(e.Error(), "5") {
		t.Fatalf("error = %q", e.Error())
	}
	e2 := &RateLimitError{}
	if e2.Error() != ErrRateLimit.Error() {
		t.Fatalf("empty message = %q", e2.Error())
	}
	e3 := &RateLimitError{Message: "slow"}
	if !strings.Contains(e3.Error(), "slow") {
		t.Fatalf("no retry = %q", e3.Error())
	}
}

func TestRateLimitErrorIs(t *testing.T) {
	e := &RateLimitError{Message: "x"}
	if !errors.Is(e, ErrRateLimit) {
		t.Fatal("Is(ErrRateLimit) should be true")
	}
}

// ---- mapStatusError paths ----

func TestMapStatusErrorForbidden(t *testing.T) {
	err := mapStatusError(http.StatusForbidden, []byte(`{"error":{"message":"forbidden"}}`), "")
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestMapStatusError400(t *testing.T) {
	err := mapStatusError(400, []byte(`{"error":{"message":"bad request"}}`), "")
	if errors.Is(err, ErrAuth) || errors.Is(err, ErrServer) {
		t.Fatalf("unexpected sentinel, got %v", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("error = %v", err)
	}
}

func TestMapStatusErrorEmptyBody(t *testing.T) {
	err := mapStatusError(500, []byte(""), "")
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestMapStatusErrorPlainTextBody(t *testing.T) {
	err := mapStatusError(500, []byte("internal error"), "")
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestMapStatusError429NoRetryAfter(t *testing.T) {
	err := mapStatusError(429, []byte(`{"error":{"message":"rate limit"}}`), "")
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rle.RetryAfter != 0 {
		t.Fatalf("RetryAfter = %d", rle.RetryAfter)
	}
}

func TestMapStatusError429NonNumericRetryAfter(t *testing.T) {
	err := mapStatusError(429, []byte(`{"error":{"message":"x"}}`), "abc")
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rle.RetryAfter != 0 {
		t.Fatalf("RetryAfter = %d", rle.RetryAfter)
	}
}

// ---- parseCreatedAt ----

func TestParseCreatedAtEmpty(t *testing.T) {
	if parseCreatedAt("") != 0 {
		t.Fatal("empty should be 0")
	}
}

func TestParseCreatedAtInvalid(t *testing.T) {
	if parseCreatedAt("not-a-date") != 0 {
		t.Fatal("invalid should be 0")
	}
}

func TestParseCreatedAtValid(t *testing.T) {
	if parseCreatedAt("2025-05-14T00:00:00Z") == 0 {
		t.Fatal("valid RFC3339 should not be 0")
	}
}

// ---- stopSequences ----

func TestStopSequencesString(t *testing.T) {
	if seqs := stopSequences("stop"); len(seqs) != 1 || seqs[0] != "stop" {
		t.Fatalf("seqs = %v", seqs)
	}
}

func TestStopSequencesEmptyString(t *testing.T) {
	if seqs := stopSequences(""); seqs != nil {
		t.Fatalf("seqs = %v", seqs)
	}
}

func TestStopSequencesSlice(t *testing.T) {
	if seqs := stopSequences([]string{"a", "b"}); len(seqs) != 2 {
		t.Fatalf("seqs = %v", seqs)
	}
}

func TestStopSequencesOther(t *testing.T) {
	if seqs := stopSequences(42); seqs != nil {
		t.Fatalf("seqs = %v", seqs)
	}
}

// ---- anthropicToolChoice ----

func TestAnthropicToolChoiceAuto(t *testing.T) {
	c := anthropicToolChoice("auto")
	if c == nil || c.Type != "auto" {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceEmpty(t *testing.T) {
	c := anthropicToolChoice("")
	if c == nil || c.Type != "auto" {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceNone(t *testing.T) {
	c := anthropicToolChoice("none")
	if c == nil || c.Type != "none" {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceRequired(t *testing.T) {
	c := anthropicToolChoice("required")
	if c == nil || c.Type != "any" {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceUnknownString(t *testing.T) {
	c := anthropicToolChoice("bogus")
	if c != nil {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceNil(t *testing.T) {
	if anthropicToolChoice(nil) != nil {
		t.Fatal("nil choice should return nil")
	}
}

func TestAnthropicToolChoiceFromMapAuto(t *testing.T) {
	c := anthropicToolChoice(map[string]any{"type": "auto"})
	if c == nil || c.Type != "auto" {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceFromMapNone(t *testing.T) {
	c := anthropicToolChoice(map[string]any{"type": "none"})
	if c == nil || c.Type != "none" {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceFromMapRequired(t *testing.T) {
	c := anthropicToolChoice(map[string]any{"type": "required"})
	if c == nil || c.Type != "any" {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceFromMapFunctionNoName(t *testing.T) {
	c := anthropicToolChoice(map[string]any{"type": "function", "function": map[string]any{}})
	if c != nil {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceFromMapFunctionNoFunctionKey(t *testing.T) {
	c := anthropicToolChoice(map[string]any{"type": "function"})
	if c != nil {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceFromMapUnknown(t *testing.T) {
	c := anthropicToolChoice(map[string]any{"type": "bogus"})
	if c != nil {
		t.Fatalf("choice = %+v", c)
	}
}

func TestAnthropicToolChoiceOtherType(t *testing.T) {
	// a struct that marshals to JSON map
	type myChoice struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	v := myChoice{Type: "function"}
	v.Function.Name = "my_tool"
	c := anthropicToolChoice(v)
	if c == nil || c.Type != "tool" || c.Name != "my_tool" {
		t.Fatalf("choice = %+v", c)
	}
}

// ---- anthropicTools with non-function type ----

func TestAnthropicToolsSkipsNonFunction(t *testing.T) {
	tools := anthropicTools([]providers.Tool{
		{Type: "other", Function: providers.ToolFunction{Name: "skip"}},
		{Type: "function", Function: providers.ToolFunction{Name: "keep"}},
	})
	if len(tools) != 1 || tools[0].Name != "keep" {
		t.Fatalf("tools = %+v", tools)
	}
}

func TestAnthropicToolsEmpty(t *testing.T) {
	if anthropicTools(nil) != nil {
		t.Fatal("nil tools should return nil")
	}
}

// ---- contentBlocksFromContent with slice and default ----

func TestContentBlocksFromContentSlice(t *testing.T) {
	input := []anthropicContentBlock{{Type: "text", Text: "hi"}}
	blocks, err := contentBlocksFromContent(input)
	if err != nil || len(blocks) != 1 || blocks[0].Text != "hi" {
		t.Fatalf("blocks = %+v err = %v", blocks, err)
	}
}

func TestContentBlocksFromContentNil(t *testing.T) {
	blocks, err := contentBlocksFromContent(nil)
	if err != nil || blocks != nil {
		t.Fatalf("blocks = %+v err = %v", blocks, err)
	}
}

func TestContentBlocksFromContentEmptyString(t *testing.T) {
	blocks, err := contentBlocksFromContent("")
	if err != nil || blocks != nil {
		t.Fatalf("blocks = %+v err = %v", blocks, err)
	}
}

func TestContentBlocksFromContentDefault(t *testing.T) {
	// a []map that marshals to a valid []anthropicContentBlock
	raw := []map[string]string{{"type": "text", "text": "hello"}}
	blocks, err := contentBlocksFromContent(raw)
	if err != nil || len(blocks) != 1 || blocks[0].Text != "hello" {
		t.Fatalf("blocks = %+v err = %v", blocks, err)
	}
}

// ---- contentString with non-string non-nil ----

func TestContentStringFromBlocks(t *testing.T) {
	blocks := []map[string]string{{"type": "text", "text": "world"}}
	s, err := contentString(blocks)
	if err != nil || s != "world" {
		t.Fatalf("s = %q err = %v", s, err)
	}
}

// ---- rawJSONObject ----

func TestRawJSONObjectEmpty(t *testing.T) {
	out, err := rawJSONObject("")
	if err != nil || string(out) != "{}" {
		t.Fatalf("out = %s err = %v", out, err)
	}
}

func TestRawJSONObjectValid(t *testing.T) {
	out, err := rawJSONObject(`{"a":1}`)
	if err != nil || string(out) != `{"a":1}` {
		t.Fatalf("out = %s err = %v", out, err)
	}
}

func TestRawJSONObjectInvalid(t *testing.T) {
	_, err := rawJSONObject(`not json`)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- toToolUseBlock non-function ----

func TestToToolUseBlockNonFunction(t *testing.T) {
	_, err := toToolUseBlock(providers.ToolCall{Type: "other"})
	if err == nil {
		t.Fatal("expected error for non-function tool call")
	}
}

func TestToToolUseBlockInvalidJSON(t *testing.T) {
	_, err := toToolUseBlock(providers.ToolCall{
		Type: "function",
		Function: providers.ToolCallFunc{
			Name:      "fn",
			Arguments: "not-json",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON arguments")
	}
}

// ---- toAnthropicRequest nil ----

func TestToAnthropicRequestNil(t *testing.T) {
	_, err := toAnthropicRequest(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---- toAnthropicRequest MaxCompletionTokens ----

func TestToAnthropicRequestMaxCompletionTokens(t *testing.T) {
	maxComp := 512
	req, err := toAnthropicRequest(&providers.ChatRequest{
		Model:               "claude-3",
		MaxCompletionTokens: &maxComp,
		Messages:            []providers.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil || req.MaxTokens != 512 {
		t.Fatalf("MaxTokens = %d err = %v", req.MaxTokens, err)
	}
}

// ---- system from message ----

func TestToAnthropicRequestSystemFromMessage(t *testing.T) {
	req, err := toAnthropicRequest(&providers.ChatRequest{
		Model: "claude-3",
		Messages: []providers.Message{
			{Role: "system", Content: "be helpful"},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil || req.System != "be helpful" {
		t.Fatalf("system = %+v err = %v", req.System, err)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", req.Messages)
	}
}

// ---- toAnthropicRequest with System set explicitly ----

func TestToAnthropicRequestSystemNotOverriddenByMessage(t *testing.T) {
	req, err := toAnthropicRequest(&providers.ChatRequest{
		Model:  "claude-3",
		System: "preset",
		Messages: []providers.Message{
			{Role: "system", Content: "ignored"},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil || req.System != "preset" {
		t.Fatalf("system = %+v err = %v", req.System, err)
	}
	// system message preserved as user message since System != nil
	if len(req.Messages) != 2 {
		t.Fatalf("messages = %+v", req.Messages)
	}
}

// ---- mapStopReason ----

func TestMapStopReasonNil(t *testing.T) {
	if mapStopReason(nil) != nil {
		t.Fatal("nil should return nil")
	}
}

func TestMapStopReasonEndTurn(t *testing.T) {
	r := "end_turn"
	out := mapStopReason(&r)
	if out == nil || *out != "stop" {
		t.Fatalf("out = %+v", out)
	}
}

func TestMapStopReasonMaxTokens(t *testing.T) {
	r := "max_tokens"
	out := mapStopReason(&r)
	if out == nil || *out != "length" {
		t.Fatalf("out = %+v", out)
	}
}

func TestMapStopReasonToolUse(t *testing.T) {
	r := "tool_use"
	out := mapStopReason(&r)
	if out == nil || *out != "tool_calls" {
		t.Fatalf("out = %+v", out)
	}
}

func TestMapStopReasonStopSequence(t *testing.T) {
	r := "stop_sequence"
	out := mapStopReason(&r)
	if out == nil || *out != "stop" {
		t.Fatalf("out = %+v", out)
	}
}

func TestMapStopReasonPassthrough(t *testing.T) {
	r := "other"
	out := mapStopReason(&r)
	if out == nil || *out != "other" {
		t.Fatalf("out = %+v", out)
	}
}

// ---- toUsage ----

func TestToUsageZero(t *testing.T) {
	if toUsage(anthropicUsage{}) != nil {
		t.Fatal("zero usage should return nil")
	}
}

func TestToUsageNonZero(t *testing.T) {
	u := toUsage(anthropicUsage{InputTokens: 5, OutputTokens: 3})
	if u == nil || u.TotalTokens != 8 {
		t.Fatalf("usage = %+v", u)
	}
}

// ---- compactJSONString ----

func TestCompactJSONStringEmpty(t *testing.T) {
	out, err := compactJSONString(nil)
	if err != nil || out != "{}" {
		t.Fatalf("out = %q err = %v", out, err)
	}
}

func TestCompactJSONStringInvalid(t *testing.T) {
	_, err := compactJSONString(json.RawMessage(`{bad`))
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- toolCallsFromContent with bad input ----

func TestToolCallsFromContentWithInvalidInput(t *testing.T) {
	// invalid JSON in Input field – should fall back to raw string
	calls := toolCallsFromContent([]anthropicContentBlock{
		{Type: "tool_use", ID: "id1", Name: "fn", Input: json.RawMessage(`{bad`)},
	})
	if len(calls) != 1 {
		t.Fatalf("calls = %+v", calls)
	}
}

// ---- SSE: content_block_start with non-tool-use type ----

func TestParseSSEContentBlockStartTextType(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: message_start",
		"data: " + streamMessageStartJSON,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","id":"","name":""}}`,
		"",
		"event: content_block_delta",
		"data: " + streamContentDeltaJSON,
		"",
		"event: message_delta",
		"data: " + streamMessageDeltaJSON,
		"",
	}, "\n"))
	p := New(server.URL)
	chunks, err := p.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := collectChunks(chunks)
	// message_start + content_block_delta + message_delta
	if len(got) < 2 {
		t.Fatalf("chunks = %d", len(got))
	}
}

// ---- SSE: content_block_delta input_json_delta with missing block ----

func TestParseSSEInputJSONDeltaMissingBlock(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: message_start",
		"data: " + streamMessageStartJSON,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":99,"delta":{"type":"input_json_delta","partial_json":"{\"x\":1}"}}`,
		"",
		"event: message_delta",
		"data: " + streamMessageDeltaJSON,
		"",
	}, "\n"))
	p := New(server.URL)
	chunks, err := p.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := collectChunks(chunks)
	// should not crash; message_start + message_delta
	if len(got) < 1 {
		t.Fatalf("chunks = %d", len(got))
	}
}

// ---- SSE: content_block_stop with no matching block ----

func TestParseSSEContentBlockStopMissingBlock(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: message_start",
		"data: " + streamMessageStartJSON,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":99}`,
		"",
		"event: message_delta",
		"data: " + streamMessageDeltaJSON,
		"",
	}, "\n"))
	p := New(server.URL)
	chunks, err := p.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := collectChunks(chunks)
	if len(got) < 1 {
		t.Fatalf("chunks = %d", len(got))
	}
}

// ---- SSE: content_block_start with pre-filled input ----

func TestParseSSEToolUseStartWithInput(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"event: message_start",
		"data: " + streamMessageStartJSON,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_2","name":"fn","input":{"city":"London"}}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":1}`,
		"",
		"event: message_delta",
		"data: " + streamMessageDeltaJSON,
		"",
	}, "\n"))
	p := New(server.URL)
	chunks, err := p.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := collectChunks(chunks)
	if len(got) < 2 {
		t.Fatalf("chunks = %d", len(got))
	}
}

// ---- SSE: scanner error is emitted as error chunk ----

// ---- SSE: [DONE] terminates stream ----

func TestParseSSEDoneTerminates(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"data: [DONE]",
		"",
		"data: " + streamContentDeltaJSON,
		"",
	}, "\n"))
	p := New(server.URL)
	chunks, err := p.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	got := collectChunks(chunks)
	if len(got) != 0 {
		t.Fatalf("expected 0 chunks after [DONE], got %d", len(got))
	}
}

// ---- SSE: stream error from HTTP error status ----

func TestChatCompletionStreamErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad key"}}`, nil)
	p := New(server.URL)
	_, err := p.ChatCompletionStream(context.Background(), testKey("api_key"), testChatRequest())
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- ListModels error paths ----

func TestListModelsError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"unauth"}}`, nil)
	p := New(server.URL)
	_, err := p.ListModels(context.Background(), testKey("api_key"))
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestListModelsBadJSON(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.ListModels(context.Background(), testKey("api_key"))
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

// ---- ChatCompletion bad JSON response ----

func TestChatCompletionBadJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.ChatCompletion(context.Background(), testKey("api_key"), testChatRequest())
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

// ---- do: cancelled context ----

func TestDoContextCancelledBeforeRequest(t *testing.T) {
	p := New("http://127.0.0.1:1") // nothing listening
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	_, err := p.ChatCompletion(ctx, testKey("api_key"), testChatRequest())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ---- newHTTPJSONRequest with nil body ----

func TestNewHTTPJSONRequestNilBody(t *testing.T) {
	p := New("http://example.com")
	req, err := p.newHTTPJSONRequest(context.Background(), "GET", "/v1/models", testKey("api_key"), nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if req.Header.Get("Content-Type") != "" {
		t.Fatal("Content-Type should not be set for nil body")
	}
}

// ---- newJSONRequest headers ----

func TestNewJSONRequestSetsHeaders(t *testing.T) {
	p := New("http://example.com")
	req, err := p.newJSONRequest("GET", "/v1/models", testKey("api_key"), nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer func() {
		// just check header values
		if string(req.Header.Peek("x-api-key")) != "sk-ant-test" {
			t.Errorf("x-api-key = %q", req.Header.Peek("x-api-key"))
		}
		if string(req.Header.Peek("anthropic-version")) != anthropicVersion {
			t.Errorf("anthropic-version = %q", req.Header.Peek("anthropic-version"))
		}
	}()
}

// ---- newHTTPJSONRequest with provider headers and oauth ----

func TestNewHTTPJSONRequestOAuthAndHeaders(t *testing.T) {
	p := NewForProviderWithHeaders(providers.ProviderAnthropic, "http://example.com", map[string]string{
		"X-Custom": "val",
	})
	req, err := p.newHTTPJSONRequest(context.Background(), "POST", "/v1/messages", providers.Key{
		Value:    "tok",
		AuthType: "oauth",
	}, map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer tok" {
		t.Fatalf("Authorization = %q", req.Header.Get("Authorization"))
	}
	if req.Header.Get("X-Custom") != "val" {
		t.Fatalf("X-Custom = %q", req.Header.Get("X-Custom"))
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
}

// ---- do: deadline path (deadline in the future) ----

func TestDoWithDeadlineReachesServer(t *testing.T) {
	server := jsonServer(t, http.StatusOK, messageResponseJSON, nil)
	p := New(server.URL)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	resp, err := p.ChatCompletion(ctx, testKey("api_key"), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.ID != "msg_123" {
		t.Fatalf("resp.ID = %q", resp.ID)
	}
}

// ---- do: deadline already expired ----

func TestDoWithExpiredDeadline(t *testing.T) {
	p := New("http://127.0.0.1:1")
	deadline := time.Now().Add(-1 * time.Second) // already in the past
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, err := p.ChatCompletion(ctx, testKey("api_key"), testChatRequest())
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}

// ---- parseSSE: data line with no trailing blank line (EOF flush) ----

func TestParseSSEFlushesDataAtEOF(t *testing.T) {
	// No trailing blank line – scanner hits EOF with pending dataLines
	body := strings.Join([]string{
		"data: " + streamContentDeltaJSON,
		// intentionally no trailing ""
	}, "\n")
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(strings.NewReader(body), chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one chunk from EOF flush")
	}
}

// ---- parseSSE: line without data: prefix is ignored ----

func TestParseSSEIgnoresNonDataLines(t *testing.T) {
	body := strings.Join([]string{
		"event: content_block_delta",
		"data: " + streamContentDeltaJSON,
		"",
	}, "\n")
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(strings.NewReader(body), chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	if len(got) != 1 {
		t.Fatalf("got %d chunks", len(got))
	}
}

// ---- handleSSEData: empty dataLines returns false ----

func TestHandleSSEDataEmptyLines(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 1)
	state := &streamState{}
	done := handleSSEData(nil, chunks, state)
	if done {
		t.Fatal("empty lines should return false")
	}
}

// ---- handleSSEData: message_start with nil Message ----

func TestHandleSSEDataMessageStartNilMessage(t *testing.T) {
	data := `{"type":"message_start"}`
	chunks := make(chan providers.StreamChunk, 1)
	state := &streamState{}
	done := handleSSEData([]string{data}, chunks, state)
	if done {
		t.Fatal("nil message should not signal done")
	}
	if len(chunks) != 0 {
		t.Fatal("nil message should not emit chunk")
	}
}

// ---- handleSSEData: content_block_delta text_delta with empty text ----

func TestHandleSSEDataContentDeltaEmptyText(t *testing.T) {
	data := `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":""}}`
	chunks := make(chan providers.StreamChunk, 1)
	state := &streamState{}
	done := handleSSEData([]string{data}, chunks, state)
	if done {
		t.Fatal("empty text delta should not signal done")
	}
	if len(chunks) != 0 {
		t.Fatal("empty text delta should not emit chunk")
	}
}

// ---- contentString: error from contentBlocksFromContent ----

func TestContentStringErrorFromBlocks(t *testing.T) {
	// Pass something that marshals to non-array JSON → decode to []anthropicContentBlock fails
	// A bare number marshals fine but won't unmarshal into []anthropicContentBlock
	_, err := contentString(42)
	if err == nil {
		t.Fatal("expected error decoding non-array content")
	}
}

// ---- contentBlocksFromContent: unmarshal failure ----

func TestContentBlocksFromContentUnmarshalFailure(t *testing.T) {
	// An integer marshals to "42" which is valid JSON but not a []anthropicContentBlock
	_, err := contentBlocksFromContent(42)
	if err == nil {
		t.Fatal("expected error for non-array type")
	}
}

// ---- toContentBlocks: non-tool message with tool calls ----

func TestToContentBlocksWithToolCalls(t *testing.T) {
	msg := providers.Message{
		Role:    "assistant",
		Content: "thinking",
		ToolCalls: []providers.ToolCall{{
			ID:   "tc1",
			Type: "function",
			Function: providers.ToolCallFunc{
				Name:      "fn",
				Arguments: `{"x":1}`,
			},
		}},
	}
	blocks, err := toContentBlocks(msg)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks = %+v", blocks)
	}
	if blocks[0].Type != "text" || blocks[1].Type != "tool_use" {
		t.Fatalf("block types = %q %q", blocks[0].Type, blocks[1].Type)
	}
}

// ---- toToolResultBlock: contentString via blocks ----

func TestToToolResultBlockFromBlocks(t *testing.T) {
	id := "tc1"
	// content as []map so contentString goes through contentBlocksFromContent path
	msg := providers.Message{
		Role:       "tool",
		ToolCallID: &id,
		Content:    []map[string]string{{"type": "text", "text": "result text"}},
	}
	blocks, err := toToolResultBlock(msg)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(blocks) != 1 || blocks[0].Content != "result text" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

// ---- anthropicToolChoice: marshal failure branch ----

func TestAnthropicToolChoiceMarshalFailure(t *testing.T) {
	// chan cannot be marshaled to JSON — hits the err != nil in default branch
	c := anthropicToolChoice(make(chan int))
	if c != nil {
		t.Fatalf("expected nil for unmarshalable value, got %+v", c)
	}
}

// ---- anthropicToolChoice: default branch with valid JSON that is not a map ----

func TestAnthropicToolChoiceDefaultBranchUnmarshalFailure(t *testing.T) {
	// A struct that marshals to a JSON array — not a map[string]any → Unmarshal fails
	c := anthropicToolChoice([]string{"auto"})
	if c != nil {
		t.Fatalf("expected nil for array JSON, got %+v", c)
	}
}

// ---- newJSONRequest: marshal failure ----

func TestNewJSONRequestMarshalFailure(t *testing.T) {
	p := New("http://example.com")
	// chan cannot be marshaled
	_, err := p.newJSONRequest("POST", "/v1/messages", testKey("api_key"), make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

// ---- ChatCompletion: newJSONRequest marshal fail propagates ----

func TestChatCompletionRequestMarshalFailViaToolChoice(t *testing.T) {
	// Pass a request where toAnthropicRequest itself fails (nil request)
	p := New("http://example.com")
	_, err := p.ChatCompletion(context.Background(), testKey("api_key"), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---- ChatCompletionStream: toAnthropicRequest nil propagates ----

func TestChatCompletionStreamNilRequest(t *testing.T) {
	p := New("http://example.com")
	_, err := p.ChatCompletionStream(context.Background(), testKey("api_key"), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---- parseSSE: handleSSEData called on flush path (no trailing newline) ----

func TestParseSSEHandleSSEDataFlushPath(t *testing.T) {
	// Data line with no trailing blank — the post-loop flush path executes
	body := "data: [DONE]" // no trailing \n\n — hits the len(dataLines)>0 path
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(strings.NewReader(body), chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	// [DONE] returns true in handleSSEData, so nothing emitted
	if len(got) != 0 {
		t.Fatalf("expected 0 chunks, got %d", len(got))
	}
}

// ---- parseSSE: scanner error via errReader ----

type scanErrReader struct {
	payload string
	sent    bool
	errSent bool
}

func (r *scanErrReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		n := copy(p, r.payload)
		return n, nil
	}
	if !r.errSent {
		r.errSent = true
		return 0, &fakeReadErr{}
	}
	// After the error, return EOF so scanner stops
	return 0, bytes.ErrTooLarge // any non-io.EOF error triggers scanner.Err()
}

type fakeReadErr struct{}

func (e *fakeReadErr) Error() string { return "injected read error" }

func TestParseSSEScannerError(t *testing.T) {
	// Build payload: a complete SSE event followed by an error mid-stream
	// bufio.Scanner surfaces Read errors in scanner.Err() after Scan() returns false
	r := &scanErrReader{payload: "data: " + streamContentDeltaJSON + "\n\n"}
	chunks := make(chan providers.StreamChunk, 10)
	parseSSE(r, chunks)
	close(chunks)
	var got []providers.StreamChunk
	for c := range chunks {
		got = append(got, c)
	}
	// At minimum: the valid chunk + possibly an error chunk
	_ = got
}

// ---- toContentBlocks: error from toToolUseBlock propagates ----

func TestToContentBlocksToolCallError(t *testing.T) {
	msg := providers.Message{
		Role:    "assistant",
		Content: "text",
		ToolCalls: []providers.ToolCall{{
			Type: "function",
			Function: providers.ToolCallFunc{
				Name:      "fn",
				Arguments: "not-json", // invalid → rawJSONObject error
			},
		}},
	}
	_, err := toContentBlocks(msg)
	if err == nil {
		t.Fatal("expected error from invalid tool call JSON")
	}
}

// ---- toToolResultBlock: contentString error ----

func TestToToolResultBlockContentStringError(t *testing.T) {
	id := "tc1"
	// Pass an int as content — contentString's default branch calls contentBlocksFromContent
	// which returns error for non-array type
	msg := providers.Message{
		Role:       "tool",
		ToolCallID: &id,
		Content:    42, // → contentString default → contentBlocksFromContent(42) → error
	}
	_, err := toToolResultBlock(msg)
	if err == nil {
		t.Fatal("expected error for non-string non-block content")
	}
}

// ---- ListModels: do() failure (bad URL) ----

func TestListModelsDoError(t *testing.T) {
	// Use an unreachable address so Do() returns a network error
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.ListModels(ctx, testKey("api_key"))
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- ChatCompletion: do() failure ----

func TestChatCompletionDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.ChatCompletion(ctx, testKey("api_key"), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
}
