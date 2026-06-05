package translate

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// --- systemContent ---

func TestSystemContentString(t *testing.T) {
	got := systemContent("hello world")
	if got != "hello world" {
		t.Fatalf("systemContent string = %q, want hello world", got)
	}
}

func TestSystemContentNonString(t *testing.T) {
	got := systemContent(42)
	if got == "" {
		t.Fatal("systemContent non-string returned empty")
	}
}

// --- isAnthropicSystem ---

func TestIsAnthropicSystemWithArray(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"system"}]`)
	if !isAnthropicSystem(raw) {
		t.Fatal("isAnthropicSystem array = false, want true")
	}
}

func TestIsAnthropicSystemWithNumber(t *testing.T) {
	raw := json.RawMessage(`123`)
	if isAnthropicSystem(raw) {
		t.Fatal("isAnthropicSystem number = true, want false")
	}
}

// --- NormalizeOpenAI unsupported format ---

func TestNormalizeOpenAIRejectsGeminiFormat(t *testing.T) {
	_, _, err := NormalizeOpenAI([]byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`))
	if err == nil {
		t.Fatal("NormalizeOpenAI gemini format should fail")
	}
}

func TestNormalizeOpenAIRejectsUnknownFormat(t *testing.T) {
	_, _, err := NormalizeOpenAI([]byte(`{"model":"x"}`))
	if err == nil {
		t.Fatal("NormalizeOpenAI unknown format should fail")
	}
}

func TestNormalizeOpenAIRejectsInvalidJSON(t *testing.T) {
	_, _, err := NormalizeOpenAI([]byte(`{bad json`))
	if err == nil {
		t.Fatal("NormalizeOpenAI bad JSON should fail")
	}
}

// --- parseChatRequest error ---

func TestParseChatRequestInvalidJSON(t *testing.T) {
	_, err := parseChatRequest([]byte(`{bad`))
	if err == nil {
		t.Fatal("parseChatRequest bad JSON should fail")
	}
}

// --- parseFunctionArgs ---

func TestParseFunctionArgsEmpty(t *testing.T) {
	got, err := parseFunctionArgs("")
	if err != nil {
		t.Fatalf("parseFunctionArgs empty: %v", err)
	}
	if got != nil {
		t.Fatalf("parseFunctionArgs empty = %v, want nil", got)
	}
}

func TestParseFunctionArgsInvalidButString(t *testing.T) {
	// raw string that is not JSON map/value – falls back to {"arguments": raw}
	got, err := parseFunctionArgs("not-json-at-all!!!")
	if err != nil {
		t.Fatalf("parseFunctionArgs non-json: %v", err)
	}
	if got["arguments"] != "not-json-at-all!!!" {
		t.Fatalf("parseFunctionArgs fallback = %v", got)
	}
}

func TestParseFunctionArgsScalarJSON(t *testing.T) {
	got, err := parseFunctionArgs(`"hello"`)
	if err != nil {
		t.Fatalf("parseFunctionArgs scalar: %v", err)
	}
	if got["arguments"] != "hello" {
		t.Fatalf("parseFunctionArgs scalar = %v", got)
	}
}

// --- geminiParts edge cases ---

func TestGeminiPartsNil(t *testing.T) {
	got, err := geminiParts(nil)
	if err != nil {
		t.Fatalf("geminiParts nil: %v", err)
	}
	if got != nil {
		t.Fatalf("geminiParts nil = %v, want nil", got)
	}
}

func TestGeminiPartsUnsupportedType(t *testing.T) {
	_, err := geminiParts(struct{}{})
	if err == nil {
		t.Fatal("geminiParts unsupported type should fail")
	}
}

func TestGeminiPartsAnyBlocks(t *testing.T) {
	blocks := []any{
		map[string]any{"type": "text", "text": "hello"},
	}
	got, err := geminiParts(blocks)
	if err != nil {
		t.Fatalf("geminiParts []any: %v", err)
	}
	if len(got) != 1 || got[0].Text != "hello" {
		t.Fatalf("geminiParts []any = %v", got)
	}
}

func TestGeminiPartsAnyBlocksNonMap(t *testing.T) {
	blocks := []any{"not a map"}
	_, err := geminiParts(blocks)
	if err == nil {
		t.Fatal("geminiParts []any non-map should fail")
	}
}

// --- textFromContent with non-text blocks ---

func TestTextFromContentWithFunctionCallPart(t *testing.T) {
	// A block that has no text part should fail in text-only fields
	blocks := []map[string]any{
		{"type": "text", "text": ""},
	}
	// empty text part: goes through geminiPartsFromBlocks (omits empty), returns empty string
	_, err := textFromContent(blocks)
	// no error; returns empty string
	if err != nil {
		t.Fatalf("textFromContent empty text block: %v", err)
	}
}

// --- geminiFunctionCallPart error ---

func TestGeminiFunctionCallPartUnsupportedType(t *testing.T) {
	toolCall := providers.ToolCall{
		ID:   "call-1",
		Type: "unsupported",
		Function: providers.ToolCallFunc{
			Name:      "lookup",
			Arguments: `{"key":"val"}`,
		},
	}
	_, err := geminiFunctionCallPart(toolCall)
	if err == nil {
		t.Fatal("geminiFunctionCallPart unsupported type should fail")
	}
}

// --- geminiContent empty content ---

func TestGeminiContentEmptyParts(t *testing.T) {
	msg := providers.Message{Role: "user", Content: nil}
	_, err := geminiContent(msg, nil)
	if err == nil {
		t.Fatal("geminiContent with nil content and no tool calls should fail")
	}
}

// --- geminiToolContent no tool call ID ---

func TestGeminiToolContentNoID(t *testing.T) {
	msg := providers.Message{
		Role:    "tool",
		Content: "result text",
	}
	got, err := geminiToolContent(msg, nil)
	if err != nil {
		t.Fatalf("geminiToolContent no ID: %v", err)
	}
	if got.Parts[0].FunctionResponse.Name != "tool_result" {
		t.Fatalf("name = %q, want tool_result", got.Parts[0].FunctionResponse.Name)
	}
}

// --- geminiTools unsupported tool type ---

func TestGeminiToolsUnsupportedType(t *testing.T) {
	tools := []providers.Tool{{Type: "retrieval", Function: providers.ToolFunction{Name: "x"}}}
	_, err := geminiTools(tools)
	if err == nil {
		t.Fatal("geminiTools unsupported type should fail")
	}
}

// --- OpenAIToGemini with system as content blocks ---

func TestOpenAIToGeminiWithBlockSystemContent(t *testing.T) {
	blocks := []map[string]any{{"type": "text", "text": "block system"}}
	req := &providers.ChatRequest{
		Model:  "gemini-2.5-flash",
		System: blocks,
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	got, err := OpenAIToGemini(req)
	if err != nil {
		t.Fatalf("OpenAIToGemini block system: %v", err)
	}
	if got.SystemInstruction == nil || got.SystemInstruction.Parts[0].Text != "block system" {
		t.Fatalf("system instruction = %+v, want block system", got.SystemInstruction)
	}
}

// --- maxGeminiOutputTokens prefers MaxCompletionTokens ---

func TestMaxGeminiOutputTokensPrefersMaxCompletionTokens(t *testing.T) {
	completionTokens := 64
	maxTokens := 32
	req := &providers.ChatRequest{
		MaxCompletionTokens: &completionTokens,
		MaxTokens:           &maxTokens,
	}
	got := maxGeminiOutputTokens(req)
	if got == nil || *got != 64 {
		t.Fatalf("maxGeminiOutputTokens = %v, want 64", got)
	}
}

func TestMaxGeminiOutputTokensFallsBackToMaxTokens(t *testing.T) {
	maxTokens := 32
	req := &providers.ChatRequest{MaxTokens: &maxTokens}
	got := maxGeminiOutputTokens(req)
	if got == nil || *got != 32 {
		t.Fatalf("maxGeminiOutputTokens = %v, want 32", got)
	}
}

// --- responses: maxResponsesOutputTokens ---

func TestMaxResponsesOutputTokensPrefersMaxCompletionTokens(t *testing.T) {
	completionTokens := 64
	maxTokens := 32
	req := &providers.ChatRequest{
		MaxCompletionTokens: &completionTokens,
		MaxTokens:           &maxTokens,
	}
	got := maxResponsesOutputTokens(req)
	if got == nil || *got != 64 {
		t.Fatalf("maxResponsesOutputTokens = %v, want 64", got)
	}
}

func TestMaxResponsesOutputTokensFallsBackToMaxTokens(t *testing.T) {
	maxTokens := 32
	req := &providers.ChatRequest{MaxTokens: &maxTokens}
	got := maxResponsesOutputTokens(req)
	if got == nil || *got != 32 {
		t.Fatalf("maxResponsesOutputTokens = %v, want 32", got)
	}
}

// --- responseText unsupported type ---

func TestResponseTextUnsupportedType(t *testing.T) {
	_, err := responseText(42)
	if err == nil {
		t.Fatal("responseText int should fail")
	}
}

// --- responseContent nil ---

func TestResponseContentNil(t *testing.T) {
	got, err := responseContent(nil)
	if err != nil {
		t.Fatalf("responseContent nil: %v", err)
	}
	if got != nil {
		t.Fatalf("responseContent nil = %v, want nil", got)
	}
}

// --- responseTools nil ---

func TestResponseToolsNil(t *testing.T) {
	got := responseTools(nil)
	if got != nil {
		t.Fatalf("responseTools nil = %v, want nil", got)
	}
}

// --- ResponsesToOpenAIChat nil ---

func TestResponsesToOpenAIChatNil(t *testing.T) {
	got := ResponsesToOpenAIChat(nil)
	if got.Object != "chat.completion" {
		t.Fatalf("nil resp object = %q, want chat.completion", got.Object)
	}
}

// --- ResponsesToOpenAIChat with non-message output and OutputText fallback ---

func TestResponsesToOpenAIChatSkipsNonMessageOutput(t *testing.T) {
	resp := &ResponsesResponse{
		ID:     "resp-1",
		Output: []ResponsesOutput{{Type: "function_call", CallID: "x", Name: "lookup", Arguments: "{}"}},
	}
	got := ResponsesToOpenAIChat(resp)
	if len(got.Choices) != 0 {
		t.Fatalf("choices = %d, want 0 for non-message output", len(got.Choices))
	}
}

func TestResponsesToOpenAIChatFallsBackToOutputText(t *testing.T) {
	resp := &ResponsesResponse{
		ID:         "resp-2",
		OutputText: "fallback text",
		Output:     []ResponsesOutput{},
	}
	got := ResponsesToOpenAIChat(resp)
	if len(got.Choices) != 1 || got.Choices[0].Message.Content != "fallback text" {
		t.Fatalf("choices = %+v, want fallback text", got.Choices)
	}
}

// --- OpenAIChatToResponsesResponse nil ---

func TestOpenAIChatToResponsesResponseNil(t *testing.T) {
	got := OpenAIChatToResponsesResponse(nil)
	if got.Object != "response" || got.Status != "completed" {
		t.Fatalf("nil resp = %+v", got)
	}
}

// --- responseUsage nil ---

func TestResponseUsageNil(t *testing.T) {
	got := responseUsage(nil)
	if got != nil {
		t.Fatalf("responseUsage nil = %v, want nil", got)
	}
}

// --- providerUsageToResponses nil ---

func TestProviderUsageToResponsesNil(t *testing.T) {
	got := providerUsageToResponses(nil)
	if got != nil {
		t.Fatalf("providerUsageToResponses nil = %v, want nil", got)
	}
}

// --- openAIMessageContentText nil/non-string ---

func TestOpenAIMessageContentTextNil(t *testing.T) {
	got := openAIMessageContentText(nil)
	if got != "" {
		t.Fatalf("openAIMessageContentText nil = %q, want empty", got)
	}
}

func TestOpenAIMessageContentTextNonString(t *testing.T) {
	got := openAIMessageContentText(42)
	if got == "" {
		t.Fatal("openAIMessageContentText int returned empty")
	}
}

// --- OpenAIChatToResponses with system field as string ---

func TestOpenAIChatToResponsesWithSystemField(t *testing.T) {
	req := &providers.ChatRequest{
		Model:  "gpt-4o",
		System: "top system",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	got, err := OpenAIChatToResponses(req)
	if err != nil {
		t.Fatalf("OpenAIChatToResponses system string: %v", err)
	}
	if got.Instructions == nil || *got.Instructions != "top system" {
		t.Fatalf("instructions = %+v, want top system", got.Instructions)
	}
}

func TestOpenAIChatToResponsesNilRequest(t *testing.T) {
	_, err := OpenAIChatToResponses(nil)
	if err == nil {
		t.Fatal("OpenAIChatToResponses nil should fail")
	}
}

// --- ResponsesRequestToOpenAIChat empty role ---

func TestResponsesRequestToOpenAIChatEmptyRoleSkipped(t *testing.T) {
	req := &ResponsesRequest{
		Model: "gpt-4o",
		Input: []ResponsesInput{
			{
				Role:    "",
				Content: []ResponsesContent{{Type: "input_text", Text: ""}},
			},
		},
	}
	got, err := ResponsesRequestToOpenAIChat(req)
	if err != nil {
		t.Fatalf("ResponsesRequestToOpenAIChat empty role content: %v", err)
	}
	// empty text content -> skipped entirely
	if len(got.Messages) != 0 {
		t.Fatalf("messages = %d, want 0 for empty content", len(got.Messages))
	}
}

func TestResponsesRequestToOpenAIChatNilRequest(t *testing.T) {
	_, err := ResponsesRequestToOpenAIChat(nil)
	if err == nil {
		t.Fatal("ResponsesRequestToOpenAIChat nil should fail")
	}
}

// --- OpenAIChatToResponsesResponse with multiple choices ---

func TestOpenAIChatToResponsesResponseMultipleChoices(t *testing.T) {
	resp := &providers.ChatResponse{
		ID:    "c1",
		Model: "gpt-4o",
		Choices: []providers.Choice{
			{Message: providers.Message{Role: "assistant", Content: "first"}},
			{Message: providers.Message{Role: "assistant", Content: "second"}},
		},
	}
	got := OpenAIChatToResponsesResponse(resp)
	if got.OutputText != "first\nsecond" {
		t.Fatalf("output text = %q, want first\\nsecond", got.OutputText)
	}
	if len(got.Output) != 2 {
		t.Fatalf("output len = %d, want 2", len(got.Output))
	}
}

// --- OpenAIToGemini: error propagation ---

func TestOpenAIToGeminiSystemContentError(t *testing.T) {
	// system field with unsupported type triggers textFromContent error
	req := &providers.ChatRequest{
		Model:  "gemini-2.5-flash",
		System: 999, // not string or []map
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	_, err := OpenAIToGemini(req)
	if err == nil {
		t.Fatal("OpenAIToGemini unsupported system type should fail")
	}
}

func TestOpenAIToGeminiSystemMessageContentError(t *testing.T) {
	req := &providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "system", Content: 999}, // unsupported
			{Role: "user", Content: "hello"},
		},
	}
	_, err := OpenAIToGemini(req)
	if err == nil {
		t.Fatal("OpenAIToGemini unsupported system message content should fail")
	}
}

func TestOpenAIToGeminiInvalidToolCallInMessage(t *testing.T) {
	req := &providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{
				Role:    "assistant",
				Content: "ok",
				ToolCalls: []providers.ToolCall{
					{ID: "c1", Type: "unsupported-type", Function: providers.ToolCallFunc{Name: "f"}},
				},
			},
		},
	}
	_, err := OpenAIToGemini(req)
	if err == nil {
		t.Fatal("OpenAIToGemini invalid tool call type should fail")
	}
}

// --- geminiToolContent with ID not in map ---

func TestGeminiToolContentIDNotInMap(t *testing.T) {
	id := "unknown-call-id"
	msg := providers.Message{
		Role:       "tool",
		ToolCallID: &id,
		Content:    "result",
	}
	got, err := geminiToolContent(msg, map[string]string{})
	if err != nil {
		t.Fatalf("geminiToolContent unknown id: %v", err)
	}
	if got.Parts[0].FunctionResponse.Name != id {
		t.Fatalf("name = %q, want %q", got.Parts[0].FunctionResponse.Name, id)
	}
}

func TestGeminiToolContentTextError(t *testing.T) {
	id := "call-1"
	msg := providers.Message{
		Role:       "tool",
		ToolCallID: &id,
		Content:    999, // unsupported content type
	}
	_, err := geminiToolContent(msg, nil)
	if err == nil {
		t.Fatal("geminiToolContent unsupported content type should fail")
	}
}

// --- geminiPartsFromBlocks: non-string text ---

func TestGeminiPartsFromBlocksNonStringText(t *testing.T) {
	blocks := []map[string]any{
		{"type": "text", "text": 123},
	}
	_, err := geminiPartsFromBlocks(blocks)
	if err == nil {
		t.Fatal("geminiPartsFromBlocks non-string text should fail")
	}
}

// --- textFromContent with FunctionCall part (via []any blocks with functionCall) ---

func TestTextFromContentWithNonTextPart(t *testing.T) {
	// Pass a []map[string]any with a non-text block type — triggers error in geminiPartsFromBlocks
	blocks := []map[string]any{
		{"type": "image_url", "image_url": map[string]any{"url": "x"}},
	}
	_, err := textFromContent(blocks)
	if err == nil {
		t.Fatal("textFromContent with non-text block should fail")
	}
}

// --- geminiParts with empty string ---

func TestGeminiPartsEmptyString(t *testing.T) {
	got, err := geminiParts("")
	if err != nil {
		t.Fatalf("geminiParts empty string: %v", err)
	}
	if got != nil {
		t.Fatalf("geminiParts empty string = %v, want nil", got)
	}
}

// --- NormalizeOpenAI parseChatRequest error paths ---

func TestNormalizeOpenAIOpenAIParseError(t *testing.T) {
	// valid format detection but body can't decode if we make it odd
	// Actually parseChatRequest uses liberal json.Unmarshal for ChatRequest struct
	// so we need a body that has "messages" key but has undecodable value
	// Since ChatRequest.Messages is []Message and json is flexible, this is hard.
	// Instead test the anthropic parse error path:
	_, _, err := NormalizeOpenAI([]byte(`{"system":"ok","messages":true}`))
	if err == nil {
		t.Fatal("NormalizeOpenAI anthropic with messages=true should fail on parse")
	}
}

// --- OpenAIChatToResponses: system unsupported type ---

func TestOpenAIChatToResponsesSystemUnsupportedType(t *testing.T) {
	req := &providers.ChatRequest{
		Model:  "gpt-4o",
		System: 42, // not a string
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
	_, err := OpenAIChatToResponses(req)
	if err == nil {
		t.Fatal("OpenAIChatToResponses system unsupported type should fail")
	}
}

func TestOpenAIChatToResponsesSystemMessageUnsupportedType(t *testing.T) {
	req := &providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "system", Content: 42}, // not a string
			{Role: "user", Content: "hello"},
		},
	}
	_, err := OpenAIChatToResponses(req)
	if err == nil {
		t.Fatal("OpenAIChatToResponses system message unsupported type should fail")
	}
}

func TestOpenAIChatToResponsesContentUnsupportedType(t *testing.T) {
	req := &providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "user", Content: 42}, // not a string
		},
	}
	_, err := OpenAIChatToResponses(req)
	if err == nil {
		t.Fatal("OpenAIChatToResponses user content unsupported type should fail")
	}
}

// --- ResponsesToOpenAIChat: content type error causes choice skip ---

func TestResponsesToOpenAIChatSkipsInvalidContentType(t *testing.T) {
	resp := &ResponsesResponse{
		ID: "r1",
		Output: []ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []ResponsesContent{
					{Type: "unsupported_type", Text: "hi"},
				},
			},
		},
	}
	got := ResponsesToOpenAIChat(resp)
	// The choice is skipped when responsesContentText returns error
	if len(got.Choices) != 0 {
		t.Fatalf("choices = %d, want 0 for invalid content type", len(got.Choices))
	}
}

// --- ResponsesRequestToOpenAIChat: non-empty text with empty role ---

func TestResponsesRequestToOpenAIChatEmptyRoleError(t *testing.T) {
	req := &ResponsesRequest{
		Model: "gpt-4o",
		Input: []ResponsesInput{
			{
				Role: "", // empty role with non-empty content
				Content: []ResponsesContent{
					{Type: "input_text", Text: "hello"},
				},
			},
		},
	}
	_, err := ResponsesRequestToOpenAIChat(req)
	if err == nil {
		t.Fatal("ResponsesRequestToOpenAIChat empty role with content should fail")
	}
}
