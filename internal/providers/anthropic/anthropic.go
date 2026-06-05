package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/valyala/fasthttp"
)

const (
	defaultBaseURL             = "https://api.anthropic.com"
	anthropicVersion           = "2023-06-01"
	defaultMaxTokens           = 1024
	defaultToolInputSchemaJSON = `{"type":"object","properties":{}}`
)

type AnthropicProvider struct {
	name         providers.ModelProvider
	baseURL      string
	headers      map[string]string
	client       *fasthttp.Client
	streamClient *http.Client
}

func New(baseURL string) *AnthropicProvider {
	return NewForProvider(providers.ProviderAnthropic, baseURL)
}

func NewForProvider(name providers.ModelProvider, baseURL string) *AnthropicProvider {
	return NewForProviderWithHeaders(name, baseURL, nil)
}

func NewForProviderWithHeaders(name providers.ModelProvider, baseURL string, headers map[string]string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	copiedHeaders := make(map[string]string, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		copiedHeaders[key] = value
	}
	return &AnthropicProvider{
		name:         name,
		baseURL:      strings.TrimRight(baseURL, "/"),
		headers:      copiedHeaders,
		client:       &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
		streamClient: utils.StreamHTTPClient(0),
	}
}

func (p *AnthropicProvider) Name() providers.ModelProvider {
	return p.name
}

func (p *AnthropicProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	anthropicReq, err := toAnthropicRequest(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/messages", key, anthropicReq)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic messages completion: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded anthropicResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse anthropic message response: %w", err)
	}
	return toChatResponse(decoded), nil
}

func (p *AnthropicProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	anthropicReq, err := toAnthropicRequest(req)
	if err != nil {
		return nil, err
	}
	stream := true
	anthropicReq.Stream = &stream

	httpReq, err := p.newHTTPJSONRequest(ctx, http.MethodPost, "/v1/messages", key, anthropicReq)
	if err != nil {
		return nil, err
	}

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic messages stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read anthropic error response: %w", readErr)
		}
		return nil, mapStatusError(resp.StatusCode, body, resp.Header.Get("Retry-After"))
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		parseSSE(resp.Body, chunks)
	}()
	return chunks, nil
}

func (p *AnthropicProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	req, err := p.newJSONRequest(fasthttp.MethodGet, "/v1/models", key, nil)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(req)

	resp, err := p.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("anthropic list models: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded modelsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse anthropic models response: %w", err)
	}

	models := make([]providers.Model, 0, len(decoded.Data))
	for _, model := range decoded.Data {
		models = append(models, providers.Model{
			ID:       model.ID,
			Object:   model.Type,
			Created:  parseCreatedAt(model.CreatedAt),
			OwnedBy:  p.name.String(),
			Provider: p.name,
		})
	}
	return models, nil
}

func (p *AnthropicProvider) newJSONRequest(method, path string, key providers.Key, body any) (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(p.baseURL + path)
	for key, value := range p.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	if key.AuthType == "oauth" {
		req.Header.Set("Authorization", "Bearer "+key.Value)
	} else {
		req.Header.Set("x-api-key", key.Value)
	}

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			fasthttp.ReleaseRequest(req)
			return nil, fmt.Errorf("marshal anthropic request: %w", err)
		}
		req.SetBody(data)
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *AnthropicProvider) newHTTPJSONRequest(ctx context.Context, method, path string, key providers.Key, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal anthropic request: %w", err)
		}
		reader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create anthropic request: %w", err)
	}
	for key, value := range p.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	if key.AuthType == "oauth" {
		req.Header.Set("Authorization", "Bearer "+key.Value)
	} else {
		req.Header.Set("x-api-key", key.Value)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *AnthropicProvider) do(ctx context.Context, req *fasthttp.Request) (*fasthttp.Response, error) {
	resp := fasthttp.AcquireResponse()
	if err := ctx.Err(); err != nil {
		fasthttp.ReleaseResponse(resp)
		return nil, err
	}

	var err error
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout <= 0 {
			fasthttp.ReleaseResponse(resp)
			return nil, context.DeadlineExceeded
		}
		err = p.client.DoTimeout(req, resp, timeout)
	} else {
		err = p.client.Do(req, resp)
	}
	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		fasthttp.ReleaseResponse(resp)
		return nil, err
	}
	return resp, nil
}

func toAnthropicRequest(req *providers.ChatRequest) (*anthropicRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("anthropic request: nil chat request")
	}
	maxTokens := defaultMaxTokens
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	} else if req.MaxCompletionTokens != nil {
		maxTokens = *req.MaxCompletionTokens
	}

	var system any = req.System
	inputMessages := make([]providers.Message, 0, len(req.Messages))
	for _, message := range req.Messages {
		if message.Role == "system" && system == nil {
			system = message.Content
			continue
		}
		inputMessages = append(inputMessages, message)
	}

	messages := make([]anthropicMessage, 0, len(inputMessages))
	for i := 0; i < len(inputMessages); i++ {
		message := inputMessages[i]
		content, err := toContentBlocks(message)
		if err != nil {
			return nil, fmt.Errorf("anthropic request message content: %w", err)
		}
		if message.Role == "tool" {
			for i+1 < len(inputMessages) && inputMessages[i+1].Role == "tool" {
				nextContent, err := toContentBlocks(inputMessages[i+1])
				if err != nil {
					return nil, fmt.Errorf("anthropic request message content: %w", err)
				}
				content = append(content, nextContent...)
				i++
			}
		}
		messages = append(messages, anthropicMessage{Role: anthropicRole(message.Role), Content: content})
	}

	return &anthropicRequest{
		Model:         req.Model,
		MaxTokens:     maxTokens,
		Messages:      messages,
		System:        system,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		StopSequences: stopSequences(req.Stop),
		Tools:         anthropicTools(req.Tools),
		ToolChoice:    anthropicToolChoice(req.ToolChoice),
		Thinking:      req.Thinking,
	}, nil
}

func anthropicRole(role string) string {
	if role == "tool" {
		return "user"
	}
	return role
}

func anthropicTools(tools []providers.Tool) []anthropicTool {
	if len(tools) == 0 {
		return nil
	}
	converted := make([]anthropicTool, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" {
			continue
		}
		converted = append(converted, anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: anthropicInputSchema(tool.Function.Parameters),
		})
	}
	return converted
}

func anthropicInputSchema(parameters json.RawMessage) json.RawMessage {
	if len(parameters) == 0 {
		return json.RawMessage(defaultToolInputSchemaJSON)
	}
	return append(json.RawMessage(nil), parameters...)
}

func anthropicToolChoice(choice any) *anthropicChoice {
	switch value := choice.(type) {
	case nil:
		return nil
	case string:
		switch value {
		case "auto", "":
			return &anthropicChoice{Type: "auto"}
		case "none":
			return &anthropicChoice{Type: "none"}
		case "required":
			return &anthropicChoice{Type: "any"}
		default:
			return nil
		}
	case map[string]any:
		return anthropicToolChoiceFromMap(value)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil
		}
		return anthropicToolChoiceFromMap(decoded)
	}
}

func anthropicToolChoiceFromMap(value map[string]any) *anthropicChoice {
	choiceType, _ := value["type"].(string)
	switch choiceType {
	case "auto", "none":
		return &anthropicChoice{Type: choiceType}
	case "required":
		return &anthropicChoice{Type: "any"}
	case "function":
		function, _ := value["function"].(map[string]any)
		name, _ := function["name"].(string)
		if name == "" {
			return nil
		}
		return &anthropicChoice{Type: "tool", Name: name}
	default:
		return nil
	}
}

func toContentBlocks(message providers.Message) ([]anthropicContentBlock, error) {
	if message.Role == "tool" {
		return toToolResultBlock(message)
	}

	blocks, err := contentBlocksFromContent(message.Content)
	if err != nil {
		return nil, err
	}
	for _, toolCall := range message.ToolCalls {
		block, err := toToolUseBlock(toolCall)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func toToolResultBlock(message providers.Message) ([]anthropicContentBlock, error) {
	text, err := contentString(message.Content)
	if err != nil {
		return nil, err
	}
	toolUseID := ""
	if message.ToolCallID != nil {
		toolUseID = *message.ToolCallID
	}
	if toolUseID == "" {
		return nil, fmt.Errorf("tool result missing tool_call_id")
	}
	return []anthropicContentBlock{{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   text,
	}}, nil
}

func toToolUseBlock(toolCall providers.ToolCall) (anthropicContentBlock, error) {
	if toolCall.Type != "function" {
		return anthropicContentBlock{}, fmt.Errorf("unsupported tool call type %q", toolCall.Type)
	}
	input, err := rawJSONObject(toolCall.Function.Arguments)
	if err != nil {
		return anthropicContentBlock{}, fmt.Errorf("tool call arguments: %w", err)
	}
	return anthropicContentBlock{
		Type:  "tool_use",
		ID:    toolCall.ID,
		Name:  toolCall.Function.Name,
		Input: input,
	}, nil
}

func contentBlocksFromContent(content any) ([]anthropicContentBlock, error) {
	switch value := content.(type) {
	case nil:
		return nil, nil
	case string:
		if value == "" {
			return nil, nil
		}
		return []anthropicContentBlock{{Type: "text", Text: value}}, nil
	case []anthropicContentBlock:
		return value, nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal content blocks: %w", err)
		}
		var blocks []anthropicContentBlock
		if err := json.Unmarshal(data, &blocks); err != nil {
			return nil, fmt.Errorf("decode content blocks: %w", err)
		}
		return blocks, nil
	}
}

func contentString(content any) (string, error) {
	switch value := content.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	default:
		blocks, err := contentBlocksFromContent(value)
		if err != nil {
			return "", err
		}
		return contentText(blocks), nil
	}
}

func rawJSONObject(raw string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return json.RawMessage(`{}`), nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return nil, err
	}
	return compactJSON(json.RawMessage(trimmed))
}

func stopSequences(stop any) []string {
	switch value := stop.(type) {
	case string:
		if value == "" {
			return nil
		}
		return []string{value}
	case []string:
		return value
	default:
		return nil
	}
}

func toChatResponse(resp anthropicResponse) *providers.ChatResponse {
	finishReason := mapStopReason(resp.StopReason)
	usage := toUsage(resp.Usage)
	return &providers.ChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:      resp.Role,
				Content:   contentText(resp.Content),
				ToolCalls: toolCallsFromContent(resp.Content),
			},
			FinishReason: finishReason,
		}},
		Usage: usage,
	}
}

func contentText(blocks []anthropicContentBlock) string {
	var parts []string
	for _, block := range blocks {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "")
}

func toolCallsFromContent(blocks []anthropicContentBlock) []providers.ToolCall {
	var toolCalls []providers.ToolCall
	for _, block := range blocks {
		if block.Type != "tool_use" {
			continue
		}
		arguments, err := compactJSONString(block.Input)
		if err != nil {
			arguments = string(block.Input)
		}
		toolCalls = append(toolCalls, providers.ToolCall{
			ID:   block.ID,
			Type: "function",
			Function: providers.ToolCallFunc{
				Name:      block.Name,
				Arguments: arguments,
			},
		})
	}
	return toolCalls
}

func compactJSONString(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "{}", nil
	}
	compact, err := compactJSON(raw)
	if err != nil {
		return "", err
	}
	return string(compact), nil
}

func compactJSON(raw json.RawMessage) (json.RawMessage, error) {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), buffer.Bytes()...), nil
}

func mapStopReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	mapped := *reason
	switch *reason {
	case "end_turn", "stop_sequence":
		mapped = "stop"
	case "max_tokens":
		mapped = "length"
	case "tool_use":
		mapped = "tool_calls"
	}
	return &mapped
}

func toUsage(usage anthropicUsage) *providers.Usage {
	if usage.InputTokens == 0 && usage.OutputTokens == 0 {
		return nil
	}
	return &providers.Usage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		TotalTokens:      usage.InputTokens + usage.OutputTokens,
	}
}

func parseSSE(body io.Reader, chunks chan<- providers.StreamChunk) {
	scanner := bufio.NewScanner(body)
	var dataLines []string
	state := streamState{}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if handleSSEData(dataLines, chunks, &state) {
				return
			}
			dataLines = nil
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if len(dataLines) > 0 {
		if handleSSEData(dataLines, chunks, &state) {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		chunks <- anthropicStreamErrorChunk("upstream_stream_error")
	}
}

type streamState struct {
	id          string
	model       string
	inputTokens int
	toolBlocks  map[int]*streamToolBlock
}

type streamToolBlock struct {
	id          string
	name        string
	partialJSON strings.Builder
}

func handleSSEData(dataLines []string, chunks chan<- providers.StreamChunk, state *streamState) bool {
	if len(dataLines) == 0 {
		return false
	}

	data := strings.Join(dataLines, "\n")
	if data == "[DONE]" {
		return true
	}

	var event streamEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		chunks <- anthropicStreamErrorChunk("upstream_stream_malformed")
		return true
	}

	switch event.Type {
	case "error":
		chunks <- anthropicStreamErrorChunk("upstream_stream_error")
		return true
	case "message_start":
		if event.Message == nil {
			return false
		}
		state.id = event.Message.ID
		state.model = event.Message.Model
		state.inputTokens = event.Message.Usage.InputTokens
		role := event.Message.Role
		chunks <- streamChunk(state, providers.StreamDelta{Role: &role}, nil, nil)
	case "content_block_start":
		if event.ContentBlock == nil || event.ContentBlock.Type != "tool_use" {
			return false
		}
		if state.toolBlocks == nil {
			state.toolBlocks = make(map[int]*streamToolBlock)
		}
		block := &streamToolBlock{
			id:   event.ContentBlock.ID,
			name: event.ContentBlock.Name,
		}
		if len(event.ContentBlock.Input) > 0 && string(event.ContentBlock.Input) != "{}" {
			block.partialJSON.WriteString(string(event.ContentBlock.Input))
		}
		state.toolBlocks[event.Index] = block
	case "content_block_delta":
		switch event.Delta.Type {
		case "input_json_delta":
			block := state.toolBlocks[event.Index]
			if block == nil {
				return false
			}
			block.partialJSON.WriteString(event.Delta.PartialJSON)
		default:
			if event.Delta.Text == "" {
				return false
			}
			text := event.Delta.Text
			chunks <- streamChunk(state, providers.StreamDelta{Content: &text}, nil, nil)
		}
	case "content_block_stop":
		block := state.toolBlocks[event.Index]
		if block == nil {
			return false
		}
		delete(state.toolBlocks, event.Index)
		arguments, err := compactJSONString(json.RawMessage(block.partialJSON.String()))
		if err != nil {
			arguments = block.partialJSON.String()
		}
		chunks <- streamChunk(state, providers.StreamDelta{
			ToolCalls: []providers.ToolCall{{
				ID:   block.id,
				Type: "function",
				Function: providers.ToolCallFunc{
					Name:      block.name,
					Arguments: arguments,
				},
			}},
		}, nil, nil)
	case "message_delta":
		finishReason := mapStopReason(event.Delta.StopReason)
		usage := event.Usage
		usage.InputTokens = state.inputTokens
		chunks <- streamChunk(state, providers.StreamDelta{}, finishReason, toUsage(usage))
	}
	return false
}

func streamChunk(state *streamState, delta providers.StreamDelta, finishReason *string, usage *providers.Usage) providers.StreamChunk {
	return providers.StreamChunk{
		ID:      state.id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   state.model,
		Choices: []providers.StreamChoice{{
			Index:        0,
			Delta:        delta,
			FinishReason: finishReason,
		}},
		Usage: usage,
	}
}

func anthropicStreamErrorChunk(code string) providers.StreamChunk {
	return providers.StreamChunk{
		Error: &providers.StreamError{
			Message: "upstream provider stream error",
			Type:    "server_error",
			Code:    code,
		},
	}
}

func mapError(resp *fasthttp.Response) error {
	return mapStatusError(resp.StatusCode(), resp.Body(), string(resp.Header.Peek("Retry-After")))
}

func mapStatusError(statusCode int, body []byte, retryAfter string) error {
	message := parseErrorMessage(body)

	switch statusCode {
	case fasthttp.StatusUnauthorized, fasthttp.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrAuth, message)
	case fasthttp.StatusTooManyRequests:
		return &RateLimitError{Message: message, RetryAfter: retryAfterSeconds(retryAfter)}
	default:
		if statusCode >= 500 {
			return fmt.Errorf("%w: %s", ErrServer, message)
		}
		return fmt.Errorf("anthropic error status %d: %s", statusCode, message)
	}
}

func parseErrorMessage(body []byte) string {
	var decoded errorResponse
	if err := json.Unmarshal(body, &decoded); err == nil && decoded.Error.Message != "" {
		return decoded.Error.Message
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "empty response"
	}
	return text
}

func retryAfterSeconds(value string) int {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return seconds
}

func parseCreatedAt(value string) int64 {
	if value == "" {
		return 0
	}
	created, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return created.Unix()
}
