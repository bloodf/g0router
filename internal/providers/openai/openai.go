package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

const defaultBaseURL = "https://api.openai.com"

type OpenAIProvider struct {
	baseURL string
	client  *fasthttp.Client
}

func New(baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &OpenAIProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
	}
}

func (p *OpenAIProvider) Name() providers.ModelProvider {
	return providers.ProviderOpenAI
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/chat/completions", key, req)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var chatResp providers.ChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("parse openai chat response: %w", err)
	}
	return &chatResp, nil
}

func (p *OpenAIProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	stream := true
	streamReq := *req
	streamReq.Stream = &stream

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/chat/completions", key, &streamReq)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion stream: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		defer fasthttp.ReleaseResponse(resp)
		return nil, mapError(resp)
	}

	chunks := make(chan providers.StreamChunk)
	body := append([]byte(nil), resp.Body()...)
	fasthttp.ReleaseResponse(resp)
	go func() {
		defer close(chunks)
		parseSSE(bytes.NewReader(body), chunks)
	}()
	return chunks, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	req, err := p.newJSONRequest(fasthttp.MethodGet, "/v1/models", key, nil)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(req)

	resp, err := p.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai list models: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded modelsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
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

func (p *OpenAIProvider) newJSONRequest(method, path string, key providers.Key, body any) (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(p.baseURL + path)
	req.Header.Set("Authorization", "Bearer "+key.Value)

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			fasthttp.ReleaseRequest(req)
			return nil, fmt.Errorf("marshal openai request: %w", err)
		}
		req.SetBody(data)
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *OpenAIProvider) do(ctx context.Context, req *fasthttp.Request) (*fasthttp.Response, error) {
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

func mapError(resp *fasthttp.Response) error {
	body := resp.Body()
	message := parseErrorMessage(body)

	switch resp.StatusCode() {
	case fasthttp.StatusUnauthorized, fasthttp.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrAuth, message)
	case fasthttp.StatusTooManyRequests:
		return &RateLimitError{Message: message, RetryAfter: retryAfterSeconds(string(resp.Header.Peek("Retry-After")))}
	default:
		if resp.StatusCode() >= 500 {
			return fmt.Errorf("%w: %s", ErrServer, message)
		}
		return fmt.Errorf("openai error status %d: %s", resp.StatusCode(), message)
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
