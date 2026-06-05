package openai

import (
	"bufio"
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

const defaultBaseURL = "https://api.openai.com"

type OpenAIProvider struct {
	baseURL      string
	client       *fasthttp.Client
	streamClient *http.Client
}

func New(baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &OpenAIProvider{
		baseURL:      strings.TrimRight(baseURL, "/"),
		client:       &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
		streamClient: utils.StreamHTTPClient(0),
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

	httpReqNet, err := p.newHTTPJSONRequest(ctx, http.MethodPost, "/v1/chat/completions", key, &streamReq)
	if err != nil {
		return nil, err
	}

	resp, err := p.streamClient.Do(httpReqNet)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read openai error response: %w", readErr)
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

func (p *OpenAIProvider) newHTTPJSONRequest(ctx context.Context, method, path string, key providers.Key, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal openai request: %w", err)
		}
		reader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)
	if body != nil {
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
			done, failed := handleSSEData(dataLines, chunks)
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
		_, failed := handleSSEData(dataLines, chunks)
		if failed {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		chunks <- streamErrorChunk("upstream_stream_error")
	}
}

func handleSSEData(dataLines []string, chunks chan<- providers.StreamChunk) (bool, bool) {
	if len(dataLines) == 0 {
		return false, false
	}

	data := strings.Join(dataLines, "\n")
	if data == "[DONE]" {
		return true, false
	}

	var chunk providers.StreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		chunks <- streamErrorChunk("upstream_stream_malformed")
		return false, true
	}
	chunks <- chunk
	return false, false
}

func streamErrorChunk(code string) providers.StreamChunk {
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
		return fmt.Errorf("openai error status %d: %s", statusCode, message)
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
