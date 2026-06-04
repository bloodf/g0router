package openaicompat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

var (
	ErrUnknownProvider = errors.New("unknown openai-compatible provider")
	ErrAuth            = errors.New("openai-compatible auth error")
	ErrRateLimit       = errors.New("openai-compatible rate limit")
	ErrServer          = errors.New("openai-compatible server error")
)

type Config struct {
	Provider            providers.ModelProvider
	BaseURL             string
	Headers             map[string]string
	ChatCompletionsPath string
}

type Provider struct {
	provider            providers.ModelProvider
	baseURL             string
	headers             map[string]string
	chatCompletionsPath string
	client              *fasthttp.Client
	streamClient        *http.Client
}

type RateLimitError struct {
	Message    string
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	if e.Message == "" {
		return ErrRateLimit.Error()
	}
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s: %s (retry after %ds)", ErrRateLimit, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("%s: %s", ErrRateLimit, e.Message)
}

func (e *RateLimitError) Is(target error) bool {
	return target == ErrRateLimit
}

func New(config Config) (*Provider, error) {
	if config.Provider == "" {
		return nil, fmt.Errorf("provider: %w", ErrUnknownProvider)
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("%s base URL: empty", config.Provider)
	}
	chatCompletionsPath := strings.TrimSpace(config.ChatCompletionsPath)
	if chatCompletionsPath == "" {
		chatCompletionsPath = "/v1/chat/completions"
	}
	if !strings.HasPrefix(chatCompletionsPath, "/") {
		return nil, fmt.Errorf("%s chat completions path: must start with /", config.Provider)
	}
	headers := make(map[string]string, len(config.Headers))
	for key, value := range config.Headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		headers[key] = value
	}
	return &Provider{
		provider:            config.Provider,
		baseURL:             strings.TrimRight(config.BaseURL, "/"),
		headers:             headers,
		chatCompletionsPath: chatCompletionsPath,
		client:              &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
		streamClient:        &http.Client{},
	}, nil
}

func (p *Provider) Name() providers.ModelProvider {
	return p.provider
}

func (p *Provider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, p.chatCompletionsPath, key, req)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s chat completion: %w", p.provider, err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(p.provider, resp)
	}

	var chatResp providers.ChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("parse %s chat response: %w", p.provider, err)
	}
	return &chatResp, nil
}

func (p *Provider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	stream := true
	streamReq := *req
	streamReq.Stream = &stream

	httpReq, err := p.newHTTPJSONRequest(ctx, http.MethodPost, p.chatCompletionsPath, key, &streamReq)
	if err != nil {
		return nil, err
	}

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s chat completion stream: %w", p.provider, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read %s error response: %w", p.provider, readErr)
		}
		return nil, mapStatusError(p.provider, resp.StatusCode, body, resp.Header.Get("Retry-After"))
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		parseSSE(resp.Body, chunks)
	}()
	return chunks, nil
}

func (p *Provider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	req, err := p.newJSONRequest(fasthttp.MethodGet, "/v1/models", key, nil)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(req)

	resp, err := p.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s list models: %w", p.provider, err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(p.provider, resp)
	}

	var decoded modelsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse %s models response: %w", p.provider, err)
	}

	models := make([]providers.Model, 0, len(decoded.Data))
	for _, model := range decoded.Data {
		models = append(models, providers.Model{
			ID:       model.ID,
			Object:   model.Object,
			Created:  model.Created,
			OwnedBy:  model.OwnedBy,
			Provider: p.provider,
		})
	}
	return models, nil
}

func (p *Provider) newJSONRequest(method, path string, key providers.Key, body any) (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(p.endpoint(path))
	for key, value := range p.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			fasthttp.ReleaseRequest(req)
			return nil, fmt.Errorf("marshal %s request: %w", p.provider, err)
		}
		req.SetBody(data)
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *Provider) newHTTPJSONRequest(ctx context.Context, method, path string, key providers.Key, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal %s request: %w", p.provider, err)
		}
		reader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, p.endpoint(path), reader)
	if err != nil {
		return nil, fmt.Errorf("create %s request: %w", p.provider, err)
	}
	for key, value := range p.headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *Provider) endpoint(path string) string {
	if strings.HasSuffix(p.baseURL, "/v1") && strings.HasPrefix(path, "/v1/") {
		return p.baseURL + strings.TrimPrefix(path, "/v1")
	}
	return p.baseURL + path
}

func (p *Provider) do(ctx context.Context, req *fasthttp.Request) (*fasthttp.Response, error) {
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

func mapError(provider providers.ModelProvider, resp *fasthttp.Response) error {
	return mapStatusError(provider, resp.StatusCode(), resp.Body(), string(resp.Header.Peek("Retry-After")))
}

func mapStatusError(provider providers.ModelProvider, statusCode int, body []byte, retryAfter string) error {
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
		return fmt.Errorf("%s error status %d: %s", provider, statusCode, message)
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

type errorResponse struct {
	Error openAIError `json:"error"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

type modelsResponse struct {
	Data []modelResponse `json:"data"`
}

type modelResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}
