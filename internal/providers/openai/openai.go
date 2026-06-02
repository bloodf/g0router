package openai

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
)

const defaultBaseURL = "https://api.openai.com"

type OpenAIProvider struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &OpenAIProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OpenAIProvider) Name() providers.ModelProvider {
	return providers.ProviderOpenAI
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	httpReq, err := p.newJSONRequest(ctx, http.MethodPost, "/v1/chat/completions", key, req)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mapError(resp)
	}

	var chatResp providers.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("parse openai chat response: %w", err)
	}
	return &chatResp, nil
}

func (p *OpenAIProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	stream := true
	streamReq := *req
	streamReq.Stream = &stream

	httpReq, err := p.newJSONRequest(ctx, http.MethodPost, "/v1/chat/completions", key, &streamReq)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, mapError(resp)
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		parseSSE(resp.Body, chunks)
	}()
	return chunks, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	req, err := p.newJSONRequest(ctx, http.MethodGet, "/v1/models", key, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mapError(resp)
	}

	var decoded modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("parse openai models response: %w", err)
	}

	models := make([]providers.Model, 0, len(decoded.Data))
	for _, model := range decoded.Data {
		models = append(models, providers.Model{
			ID:       model.ID,
			Object:   model.Object,
			Created:  model.Created,
			OwnedBy:  model.OwnedBy,
			Provider: providers.ProviderOpenAI,
		})
	}
	return models, nil
}

func (p *OpenAIProvider) newJSONRequest(ctx context.Context, method, path string, key providers.Key, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal openai request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("build openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func parseSSE(body io.Reader, chunks chan<- providers.StreamChunk) {
	scanner := bufio.NewScanner(body)
	var dataLines []string

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if handleSSEData(dataLines, chunks) {
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
		handleSSEData(dataLines, chunks)
	}
}

func handleSSEData(dataLines []string, chunks chan<- providers.StreamChunk) bool {
	if len(dataLines) == 0 {
		return false
	}

	data := strings.Join(dataLines, "\n")
	if data == "[DONE]" {
		return true
	}

	var chunk providers.StreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return false
	}
	chunks <- chunk
	return false
}

func mapError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	message := parseErrorMessage(body)

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrAuth, message)
	case http.StatusTooManyRequests:
		return &RateLimitError{Message: message, RetryAfter: retryAfterSeconds(resp.Header.Get("Retry-After"))}
	default:
		if resp.StatusCode >= 500 {
			return fmt.Errorf("%w: %s", ErrServer, message)
		}
		return fmt.Errorf("openai error status %d: %s", resp.StatusCode, message)
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
