package azure

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

const (
	defaultBaseURL    = "https://api.openai.azure.com"
	defaultAPIVersion = "2024-02-15-preview"
)

var (
	ErrAuth      = errors.New("azure auth error")
	ErrRateLimit = errors.New("azure rate limit")
	ErrServer    = errors.New("azure server error")
)

type AzureProvider struct {
	baseURL      string
	apiVersion   string
	client       *fasthttp.Client
	streamClient *http.Client
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

func New(baseURL, apiVersion string) *AzureProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if apiVersion == "" {
		apiVersion = defaultAPIVersion
	}
	return &AzureProvider{
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiVersion:   apiVersion,
		client:       &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
		streamClient: &http.Client{},
	}
}

func (p *AzureProvider) Name() providers.ModelProvider {
	return providers.ProviderAzure
}

func (p *AzureProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, chatPath(req.Model), key, req)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("azure chat completion: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var chatResp providers.ChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("parse azure chat response: %w", err)
	}
	return &chatResp, nil
}

func (p *AzureProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	stream := true
	streamReq := *req
	streamReq.Stream = &stream

	httpReq, err := p.newHTTPJSONRequest(ctx, http.MethodPost, chatPath(req.Model), key, &streamReq)
	if err != nil {
		return nil, err
	}

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("azure chat completion stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read azure error response: %w", readErr)
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

func (p *AzureProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	req, err := p.newJSONRequest(fasthttp.MethodGet, "/openai/deployments", key, nil)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(req)

	resp, err := p.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("azure list models: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded deploymentsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse azure deployments response: %w", err)
	}

	models := make([]providers.Model, 0, len(decoded.Data))
	for _, deployment := range decoded.Data {
		models = append(models, providers.Model{
			ID:       deployment.ID,
			Object:   deployment.Object,
			Created:  deployment.CreatedAt,
			OwnedBy:  "azure",
			Provider: providers.ProviderAzure,
		})
	}
	return models, nil
}

func (p *AzureProvider) newJSONRequest(method, path string, key providers.Key, body any) (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(p.baseURL + path + "?api-version=" + url.QueryEscape(p.apiVersion))
	req.Header.Set("api-key", key.Value)

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			fasthttp.ReleaseRequest(req)
			return nil, fmt.Errorf("marshal azure request: %w", err)
		}
		req.SetBody(data)
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *AzureProvider) newHTTPJSONRequest(ctx context.Context, method, path string, key providers.Key, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal azure request: %w", err)
		}
		reader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path+"?api-version="+url.QueryEscape(p.apiVersion), reader)
	if err != nil {
		return nil, fmt.Errorf("create azure request: %w", err)
	}
	req.Header.Set("api-key", key.Value)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *AzureProvider) do(ctx context.Context, req *fasthttp.Request) (*fasthttp.Response, error) {
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

func chatPath(deployment string) string {
	return "/openai/deployments/" + url.PathEscape(deployment) + "/chat/completions"
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
		return fmt.Errorf("azure error status %d: %s", statusCode, message)
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
