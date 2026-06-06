package ollamacloud

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/store"
)

const defaultBaseURL = "https://ollama.com"

type Provider struct {
	baseURL      string
	client       *http.Client
	streamClient *http.Client
	proxyPool    *store.ProxyPool
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []tool        `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
	Options  *options      `json:"options,omitempty"`
}

type chatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"`
}

type tool struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type toolCall struct {
	Type     string       `json:"type"`
	Function toolCallFunc `json:"function"`
}

type toolCallFunc struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

type options struct {
	NumPredict int `json:"num_predict,omitempty"`
}

type chatResponse struct {
	Model           string      `json:"model"`
	Message         chatMessage `json:"message"`
	Done            bool        `json:"done"`
	DoneReason      string      `json:"done_reason"`
	PromptEvalCount int         `json:"prompt_eval_count"`
	EvalCount       int         `json:"eval_count"`
}

type tagsResponse struct {
	Models []tagModel `json:"models"`
}

type tagModel struct {
	Name  string `json:"name"`
	Model string `json:"model"`
}

func New(baseURL string, proxyPool ...*store.ProxyPool) (*Provider, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	var pool *store.ProxyPool
	if len(proxyPool) > 0 {
		pool = proxyPool[0]
	}
	client := utils.HTTPClientForPool(pool)
	client.Timeout = 60 * time.Second
	return &Provider{
		baseURL:      baseURL,
		client:       client,
		streamClient: utils.StreamHTTPClientForPool(0, pool),
		proxyPool:    pool,
	}, nil
}

func NewDefault() (*Provider, error) {
	return New("")
}

func (p *Provider) WithProxyPool(pool *store.ProxyPool) providers.Provider {
	provider, _ := New(p.baseURL, pool)
	return provider
}

func (p *Provider) Name() providers.ModelProvider {
	return providers.ProviderOllamaCloud
}

func (p *Provider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	nativeReq, err := toChatRequest(req, false)
	if err != nil {
		return nil, err
	}
	var decoded chatResponse
	if err := p.doJSON(ctx, http.MethodPost, "/api/chat", key, nativeReq, &decoded); err != nil {
		return nil, fmt.Errorf("ollama-cloud chat completion: %w", err)
	}
	return toChatResponse(decoded), nil
}

func (p *Provider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	nativeReq, err := toChatRequest(req, true)
	if err != nil {
		return nil, err
	}
	httpReq, err := p.newRequest(ctx, http.MethodPost, "/api/chat", key, nativeReq)
	if err != nil {
		return nil, err
	}
	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama-cloud chat completion stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, readStatusError("ollama-cloud stream", resp)
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var decoded chatResponse
			if err := json.Unmarshal([]byte(line), &decoded); err != nil {
				chunks <- providers.StreamChunk{Error: &providers.StreamError{Message: "malformed ollama-cloud stream chunk", Type: "upstream_error", Code: "malformed_stream"}}
				return
			}
			chunks <- toStreamChunk(decoded)
		}
		if err := scanner.Err(); err != nil {
			chunks <- providers.StreamChunk{Error: &providers.StreamError{Message: "read ollama-cloud stream", Type: "upstream_error", Code: "stream_read_error"}}
		}
	}()
	return chunks, nil
}

func (p *Provider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	var decoded tagsResponse
	if err := p.doJSON(ctx, http.MethodGet, "/api/tags", key, nil, &decoded); err != nil {
		return nil, fmt.Errorf("ollama-cloud list models: %w", err)
	}
	models := make([]providers.Model, 0, len(decoded.Models))
	for _, model := range decoded.Models {
		id := strings.TrimSpace(model.Model)
		if id == "" {
			id = strings.TrimSpace(model.Name)
		}
		if id == "" {
			continue
		}
		models = append(models, providers.Model{
			ID:       id,
			Object:   "model",
			OwnedBy:  providers.ProviderOllamaCloud.String(),
			Provider: providers.ProviderOllamaCloud,
		})
	}
	return models, nil
}

func (p *Provider) doJSON(ctx context.Context, method string, path string, key providers.Key, body any, out any) error {
	req, err := p.newRequest(ctx, method, path, key, body)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readStatusError("ollama-cloud", resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (p *Provider) newRequest(ctx context.Context, method string, path string, key providers.Key, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal ollama-cloud request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create ollama-cloud request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func toChatRequest(req *providers.ChatRequest, stream bool) (chatRequest, error) {
	if req == nil {
		return chatRequest{}, fmt.Errorf("ollama-cloud request: nil chat request")
	}
	messages := make([]chatMessage, 0, len(req.Messages))
	for index, msg := range req.Messages {
		converted, err := toChatMessage(msg)
		if err != nil {
			return chatRequest{}, fmt.Errorf("ollama-cloud message %d: %w", index, err)
		}
		messages = append(messages, converted)
	}
	out := chatRequest{
		Model:    req.Model,
		Messages: messages,
		Tools:    toTools(req.Tools),
		Stream:   stream,
	}
	maxTokens := 0
	if req.MaxCompletionTokens != nil {
		maxTokens = *req.MaxCompletionTokens
	} else if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	if maxTokens > 0 {
		out.Options = &options{NumPredict: maxTokens}
	}
	return out, nil
}

func toChatMessage(msg providers.Message) (chatMessage, error) {
	role := msg.Role
	if role == "developer" {
		role = "system"
	}
	converted := chatMessage{
		Role:      role,
		ToolCalls: toToolCalls(msg.ToolCalls),
	}
	if msg.ToolCallID != nil {
		converted.Role = "tool"
		if msg.Name != nil {
			converted.ToolName = *msg.Name
		}
	}
	content, err := contentText(msg.Content)
	if err != nil {
		return chatMessage{}, err
	}
	converted.Content = content
	return converted, nil
}

func contentText(content any) (string, error) {
	switch value := content.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	case []any:
		var parts []string
		for _, part := range value {
			text, err := contentText(part)
			if err != nil {
				return "", err
			}
			if text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n"), nil
	case map[string]any:
		if value["type"] == "text" {
			if text, ok := value["text"].(string); ok {
				return text, nil
			}
		}
		return "", fmt.Errorf("unsupported content block")
	default:
		return "", fmt.Errorf("unsupported content type %T", content)
	}
}

func toTools(input []providers.Tool) []tool {
	if len(input) == 0 {
		return nil
	}
	out := make([]tool, 0, len(input))
	for _, item := range input {
		out = append(out, tool{
			Type: item.Type,
			Function: toolFunction{
				Name:        item.Function.Name,
				Description: item.Function.Description,
				Parameters:  item.Function.Parameters,
			},
		})
	}
	return out
}

func toToolCalls(input []providers.ToolCall) []toolCall {
	if len(input) == 0 {
		return nil
	}
	out := make([]toolCall, 0, len(input))
	for _, item := range input {
		out = append(out, toolCall{
			Type: item.Type,
			Function: toolCallFunc{
				Name:      item.Function.Name,
				Arguments: item.Function.Arguments,
			},
		})
	}
	return out
}

func toChatResponse(resp chatResponse) *providers.ChatResponse {
	finishReason := finishReason(resp.DoneReason)
	return &providers.ChatResponse{
		ID:       "ollama-cloud-" + resp.Model,
		Object:   "chat.completion",
		Created:  time.Now().Unix(),
		Model:    resp.Model,
		Provider: providers.ProviderOllamaCloud,
		Choices: []providers.Choice{{
			Index:        0,
			Message:      providers.Message{Role: resp.Message.Role, Content: resp.Message.Content},
			FinishReason: &finishReason,
		}},
		Usage: toUsage(resp),
	}
}

func toStreamChunk(resp chatResponse) providers.StreamChunk {
	finishReason := finishReason(resp.DoneReason)
	content := resp.Message.Content
	role := resp.Message.Role
	chunk := providers.StreamChunk{
		ID:      "ollama-cloud-" + resp.Model,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []providers.StreamChoice{{
			Index: 0,
			Delta: providers.StreamDelta{
				Role:    stringPtr(role),
				Content: stringPtr(content),
			},
		}},
	}
	if resp.Done {
		chunk.Choices[0].FinishReason = &finishReason
		chunk.Usage = toUsage(resp)
	}
	return chunk
}

func finishReason(reason string) string {
	switch reason {
	case "", "stop":
		return "stop"
	case "tool_calls":
		return "tool_calls"
	default:
		return reason
	}
}

func toUsage(resp chatResponse) *providers.Usage {
	total := resp.PromptEvalCount + resp.EvalCount
	if total == 0 {
		return nil
	}
	return &providers.Usage{
		PromptTokens:     resp.PromptEvalCount,
		CompletionTokens: resp.EvalCount,
		TotalTokens:      total,
	}
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func readStatusError(label string, resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("%s status %d: read error body: %w", label, resp.StatusCode, err)
	}
	return fmt.Errorf("%s status %d: %s", label, resp.StatusCode, strings.TrimSpace(string(body)))
}
