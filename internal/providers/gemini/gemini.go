package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/valyala/fasthttp"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

var (
	ErrAuth        = errors.New("gemini auth error")
	ErrRateLimit   = errors.New("gemini rate limit")
	ErrServer      = errors.New("gemini server error")
	ErrUnsupported = errors.New("gemini unsupported operation")
)

type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	if e.Message == "" {
		return ErrRateLimit.Error()
	}
	return fmt.Sprintf("%s: %s", ErrRateLimit, e.Message)
}

func (e *RateLimitError) Is(target error) bool {
	return target == ErrRateLimit
}

type GeminiProvider struct {
	baseURL      string
	client       *fasthttp.Client
	streamClient *http.Client
}

func New(baseURL string) *GeminiProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &GeminiProvider{
		baseURL:      strings.TrimRight(baseURL, "/"),
		client:       &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
		streamClient: utils.StreamHTTPClient(0),
	}
}

func (p *GeminiProvider) Name() providers.ModelProvider {
	return providers.ProviderGemini
}

func (p *GeminiProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	geminiReq, err := buildGenerateContentRequest(req)
	if err != nil {
		return nil, err
	}

	path := "/v1beta/models/" + url.PathEscape(req.Model) + ":generateContent"
	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, path, key, geminiReq)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini generate content: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded generateContentResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse gemini response: %w", err)
	}
	return mapGenerateContentResponse(req.Model, decoded), nil
}

func (p *GeminiProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	geminiReq, err := buildGenerateContentRequest(req)
	if err != nil {
		return nil, err
	}

	path := "/v1beta/models/" + url.PathEscape(req.Model) + ":streamGenerateContent"
	httpReq, err := p.newHTTPJSONRequest(ctx, http.MethodPost, path, key, geminiReq, url.Values{"alt": []string{"sse"}})
	if err != nil {
		return nil, err
	}

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini stream generate content: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read gemini error response: %w", readErr)
		}
		return nil, mapStatusError(resp.StatusCode, body)
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		parseGeminiSSE(req.Model, resp.Body, chunks)
	}()
	return chunks, nil
}

func (p *GeminiProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	req, err := p.newJSONRequest(fasthttp.MethodGet, "/v1beta/models", key, nil)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(req)

	resp, err := p.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gemini list models: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded listModelsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse gemini models response: %w", err)
	}

	models := make([]providers.Model, 0, len(decoded.Models))
	for _, model := range decoded.Models {
		id := strings.TrimPrefix(model.Name, "models/")
		models = append(models, providers.Model{
			ID:       id,
			Object:   "model",
			OwnedBy:  "google",
			Provider: providers.ProviderGemini,
		})
	}
	return models, nil
}

func buildGenerateContentRequest(req *providers.ChatRequest) (*generateContentRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("gemini request: nil chat request")
	}
	geminiReq := &generateContentRequest{
		Contents: make([]content, 0, len(req.Messages)),
		GenerationConfig: &generationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: maxOutputTokens(req),
		},
	}
	if geminiReq.GenerationConfig.Temperature == nil &&
		geminiReq.GenerationConfig.TopP == nil &&
		geminiReq.GenerationConfig.MaxOutputTokens == nil {
		geminiReq.GenerationConfig = nil
	}

	tools, err := geminiTools(req.Tools)
	if err != nil {
		return nil, err
	}
	geminiReq.Tools = tools

	if system, err := textFromContent(req.System); err != nil {
		return nil, fmt.Errorf("build gemini system: %w", err)
	} else if system != "" {
		geminiReq.SystemInstruction = &content{Parts: []part{{Text: system}}}
	}

	toolCallNames := make(map[string]string)
	for _, message := range req.Messages {
		if message.Role == "system" {
			text, err := textFromContent(message.Content)
			if err != nil {
				return nil, fmt.Errorf("build gemini system message: %w", err)
			}
			geminiReq.SystemInstruction = &content{Parts: []part{{Text: text}}}
			continue
		}
		converted, err := geminiContent(message, toolCallNames)
		if err != nil {
			return nil, fmt.Errorf("build gemini content: %w", err)
		}
		for _, toolCall := range message.ToolCalls {
			if toolCall.ID != "" && toolCall.Function.Name != "" {
				toolCallNames[toolCall.ID] = toolCall.Function.Name
			}
		}
		geminiReq.Contents = append(geminiReq.Contents, converted)
	}
	return geminiReq, nil
}

func maxOutputTokens(req *providers.ChatRequest) *int {
	if req.MaxCompletionTokens != nil {
		return req.MaxCompletionTokens
	}
	return req.MaxTokens
}

func geminiTools(tools []providers.Tool) ([]geminiTool, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	declarations := make([]geminiFunctionDeclaration, 0, len(tools))
	for i, tool := range tools {
		if tool.Type != "function" {
			return nil, fmt.Errorf("tool %d: unsupported tool type %q", i, tool.Type)
		}
		declarations = append(declarations, geminiFunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  append(json.RawMessage(nil), tool.Function.Parameters...),
		})
	}
	return []geminiTool{{FunctionDeclarations: declarations}}, nil
}

func geminiContent(message providers.Message, toolCallNames map[string]string) (content, error) {
	if message.Role == "tool" {
		return geminiToolContent(message, toolCallNames)
	}

	parts, err := geminiParts(message.Content)
	if err != nil {
		return content{}, err
	}
	for i, toolCall := range message.ToolCalls {
		part, err := geminiFunctionCallPart(toolCall)
		if err != nil {
			return content{}, fmt.Errorf("tool call %d: %w", i, err)
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return content{}, fmt.Errorf("empty content for role %q", message.Role)
	}

	return content{
		Role:  geminiRole(message.Role),
		Parts: parts,
	}, nil
}

func geminiToolContent(message providers.Message, toolCallNames map[string]string) (content, error) {
	name := "tool_result"
	id := ""
	if message.ToolCallID != nil && *message.ToolCallID != "" {
		id = *message.ToolCallID
		if mappedName, ok := toolCallNames[id]; ok && mappedName != "" {
			name = mappedName
		} else {
			name = id
		}
	} else if message.Name != nil && *message.Name != "" {
		name = *message.Name
	}

	text, err := textFromContent(message.Content)
	if err != nil {
		return content{}, err
	}
	return content{
		Role: "user",
		Parts: []part{{
			FunctionResponse: &geminiFunctionResponse{
				Name:     name,
				ID:       id,
				Response: map[string]any{"content": text},
			},
		}},
	}, nil
}

func geminiFunctionCallPart(toolCall providers.ToolCall) (part, error) {
	if toolCall.Type != "function" {
		return part{}, fmt.Errorf("unsupported tool call type %q", toolCall.Type)
	}

	args, err := parseFunctionArgs(toolCall.Function.Arguments)
	if err != nil {
		return part{}, err
	}
	return part{
		FunctionCall: &geminiFunctionCall{
			ID:   toolCall.ID,
			Name: toolCall.Function.Name,
			Args: args,
		},
	}, nil
}

func parseFunctionArgs(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(trimmed), &args); err == nil {
		return args, nil
	}

	var value any
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return map[string]any{"arguments": raw}, nil
	}
	return map[string]any{"arguments": value}, nil
}

func geminiParts(value any) ([]part, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case string:
		if typed == "" {
			return nil, nil
		}
		return []part{{Text: typed}}, nil
	case []map[string]any:
		return geminiPartsFromBlocks(typed)
	case []any:
		return geminiPartsFromAnyBlocks(typed)
	default:
		return nil, fmt.Errorf("unsupported content type %T", value)
	}
}

func textFromContent(value any) (string, error) {
	parts, err := geminiParts(value)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, part := range parts {
		if part.Text == "" {
			return "", fmt.Errorf("unsupported non-text content in text-only field")
		}
		builder.WriteString(part.Text)
	}
	return builder.String(), nil
}

func geminiPartsFromAnyBlocks(blocks []any) ([]part, error) {
	typed := make([]map[string]any, 0, len(blocks))
	for i, block := range blocks {
		value, ok := block.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content block %d: unsupported content block type %T", i, block)
		}
		typed = append(typed, value)
	}
	return geminiPartsFromBlocks(typed)
}

func geminiPartsFromBlocks(blocks []map[string]any) ([]part, error) {
	parts := make([]part, 0, len(blocks))
	for i, block := range blocks {
		blockType, _ := block["type"].(string)
		if blockType != "text" {
			return nil, fmt.Errorf("content block %d: unsupported content block type %q", i, blockType)
		}

		text, ok := block["text"].(string)
		if !ok {
			return nil, fmt.Errorf("content block %d: text must be a string", i)
		}
		if text != "" {
			parts = append(parts, part{Text: text})
		}
	}
	return parts, nil
}

func geminiRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return "user"
}

func (p *GeminiProvider) newJSONRequest(method, path string, key providers.Key, body any) (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(p.requestURI(path, key))

	if key.AuthType == "oauth" {
		req.Header.Set("Authorization", "Bearer "+key.Value)
	}

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			fasthttp.ReleaseRequest(req)
			return nil, fmt.Errorf("marshal gemini request: %w", err)
		}
		req.SetBody(data)
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *GeminiProvider) newHTTPJSONRequest(ctx context.Context, method, path string, key providers.Key, body any, extraQuery url.Values) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal gemini request: %w", err)
		}
		reader = strings.NewReader(string(data))
	}

	req, err := http.NewRequestWithContext(ctx, method, p.requestURIWithQuery(path, key, extraQuery), reader)
	if err != nil {
		return nil, fmt.Errorf("create gemini request: %w", err)
	}
	if key.AuthType == "oauth" {
		req.Header.Set("Authorization", "Bearer "+key.Value)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *GeminiProvider) requestURI(path string, key providers.Key) string {
	return p.requestURIWithQuery(path, key, nil)
}

func (p *GeminiProvider) requestURIWithQuery(path string, key providers.Key, extra url.Values) string {
	uri := p.baseURL + path
	values := url.Values{}
	for name, entries := range extra {
		for _, entry := range entries {
			values.Add(name, entry)
		}
	}
	if key.AuthType != "oauth" {
		values.Set("key", key.Value)
	}
	if len(values) == 0 {
		return uri
	}
	return uri + "?" + values.Encode()
}

func (p *GeminiProvider) do(ctx context.Context, req *fasthttp.Request) (*fasthttp.Response, error) {
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

func mapGenerateContentResponse(model string, decoded generateContentResponse) *providers.ChatResponse {
	resp := &providers.ChatResponse{
		Object: "chat.completion",
		Model:  model,
		Usage: &providers.Usage{
			PromptTokens:     decoded.UsageMetadata.PromptTokenCount,
			CompletionTokens: decoded.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      decoded.UsageMetadata.TotalTokenCount,
		},
	}

	resp.Choices = make([]providers.Choice, 0, len(decoded.Candidates))
	for i, candidate := range decoded.Candidates {
		finishReason := mapFinishReason(candidate.FinishReason)
		toolCalls := toolCallsFromParts(candidate.Content.Parts)
		if len(toolCalls) > 0 {
			finishReason = "tool_calls"
		}
		resp.Choices = append(resp.Choices, providers.Choice{
			Index: i,
			Message: providers.Message{
				Role:      "assistant",
				Content:   textFromParts(candidate.Content.Parts),
				ToolCalls: toolCalls,
			},
			FinishReason: &finishReason,
		})
	}
	return resp
}

func parseGeminiSSE(model string, body io.Reader, chunks chan<- providers.StreamChunk) {
	scanner := bufio.NewScanner(body)
	var dataLines []string

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			done, failed := handleGeminiSSEData(model, dataLines, chunks)
			if done || failed {
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
		_, failed := handleGeminiSSEData(model, dataLines, chunks)
		if failed {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		chunks <- geminiStreamErrorChunk("upstream_stream_error")
	}
}

func handleGeminiSSEData(model string, dataLines []string, chunks chan<- providers.StreamChunk) (bool, bool) {
	if len(dataLines) == 0 {
		return false, false
	}

	data := strings.Join(dataLines, "\n")
	if data == "[DONE]" {
		return true, false
	}

	var decoded generateContentResponse
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		chunks <- geminiStreamErrorChunk("upstream_stream_malformed")
		return false, true
	}
	for _, chunk := range mapGenerateContentStreamChunks(model, decoded) {
		chunks <- chunk
	}
	return false, false
}

func mapGenerateContentStreamChunks(model string, decoded generateContentResponse) []providers.StreamChunk {
	chunks := make([]providers.StreamChunk, 0, len(decoded.Candidates))
	usage := usageFromMetadata(decoded.UsageMetadata)
	for i, candidate := range decoded.Candidates {
		finishReason := mapFinishReason(candidate.FinishReason)
		var finishReasonPtr *string
		if candidate.FinishReason != "" {
			finishReasonPtr = &finishReason
		}
		delta := providers.StreamDelta{}
		if text := textFromParts(candidate.Content.Parts); text != "" {
			delta.Content = &text
		}
		if toolCalls := toolCallsFromParts(candidate.Content.Parts); len(toolCalls) > 0 {
			delta.ToolCalls = toolCalls
			toolFinish := "tool_calls"
			finishReasonPtr = &toolFinish
		}
		chunks = append(chunks, providers.StreamChunk{
			Object:  "chat.completion.chunk",
			Model:   model,
			Created: time.Now().Unix(),
			Choices: []providers.StreamChoice{{
				Index:        i,
				Delta:        delta,
				FinishReason: finishReasonPtr,
			}},
			Usage: usage,
		})
	}
	if len(chunks) == 0 && usage != nil {
		chunks = append(chunks, providers.StreamChunk{
			Object:  "chat.completion.chunk",
			Model:   model,
			Created: time.Now().Unix(),
			Choices: []providers.StreamChoice{{
				Index: 0,
				Delta: providers.StreamDelta{},
			}},
			Usage: usage,
		})
	}
	return chunks
}

func usageFromMetadata(metadata usageMetadata) *providers.Usage {
	if metadata.PromptTokenCount == 0 && metadata.CandidatesTokenCount == 0 && metadata.TotalTokenCount == 0 {
		return nil
	}
	return &providers.Usage{
		PromptTokens:     metadata.PromptTokenCount,
		CompletionTokens: metadata.CandidatesTokenCount,
		TotalTokens:      metadata.TotalTokenCount,
	}
}

func geminiStreamErrorChunk(code string) providers.StreamChunk {
	return providers.StreamChunk{
		Error: &providers.StreamError{
			Message: "upstream provider stream error",
			Type:    "server_error",
			Code:    code,
		},
	}
}

func textFromParts(parts []part) string {
	var builder strings.Builder
	for _, part := range parts {
		builder.WriteString(part.Text)
	}
	return builder.String()
}

func toolCallsFromParts(parts []part) []providers.ToolCall {
	var toolCalls []providers.ToolCall
	for _, part := range parts {
		if part.FunctionCall == nil {
			continue
		}
		arguments := "{}"
		if part.FunctionCall.Args != nil {
			data, err := json.Marshal(part.FunctionCall.Args)
			if err == nil {
				arguments = string(data)
			}
		}
		toolCalls = append(toolCalls, providers.ToolCall{
			ID:   geminiToolCallID(part.FunctionCall, len(toolCalls)),
			Type: "function",
			Function: providers.ToolCallFunc{
				Name:      part.FunctionCall.Name,
				Arguments: arguments,
			},
		})
	}
	return toolCalls
}

func geminiToolCallID(call *geminiFunctionCall, index int) string {
	if call.ID != "" {
		return call.ID
	}
	return fmt.Sprintf("gemini_call_%d", index)
}

func mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION":
		return "content_filter"
	default:
		return strings.ToLower(reason)
	}
}

func mapError(resp *fasthttp.Response) error {
	return mapStatusError(resp.StatusCode(), resp.Body())
}

func mapStatusError(statusCode int, body []byte) error {
	message := parseErrorMessage(body)

	switch statusCode {
	case fasthttp.StatusUnauthorized, fasthttp.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrAuth, message)
	case fasthttp.StatusTooManyRequests:
		return &RateLimitError{Message: message}
	default:
		if statusCode >= 500 {
			return fmt.Errorf("%w: %s", ErrServer, message)
		}
		return fmt.Errorf("gemini error status %d: %s", statusCode, message)
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
