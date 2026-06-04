package vertex

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
	"github.com/valyala/fasthttp"
)

const defaultBaseURL = "https://aiplatform.googleapis.com"

var (
	ErrAuth        = errors.New("vertex auth error")
	ErrRateLimit   = errors.New("vertex rate limit")
	ErrServer      = errors.New("vertex server error")
	ErrUnsupported = errors.New("vertex unsupported operation")
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

type VertexProvider struct {
	baseURL      string
	config       Config
	client       *fasthttp.Client
	streamClient *http.Client
}

func New(baseURL string, config Config) *VertexProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &VertexProvider{
		baseURL:      strings.TrimRight(baseURL, "/"),
		config:       config,
		client:       &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
		streamClient: &http.Client{},
	}
}

func (p *VertexProvider) Name() providers.ModelProvider {
	return providers.ProviderVertex
}

func (p *VertexProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if err := p.validateConfig(); err != nil {
		return nil, err
	}
	vertexReq, err := buildGenerateContentRequest(req)
	if err != nil {
		return nil, err
	}

	path := p.modelPath(req.Model) + ":generateContent"
	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, path, key, vertexReq)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("vertex generate content: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded generateContentResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse vertex response: %w", err)
	}
	return mapGenerateContentResponse(req.Model, decoded), nil
}

func (p *VertexProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	if err := p.validateConfig(); err != nil {
		return nil, err
	}
	vertexReq, err := buildGenerateContentRequest(req)
	if err != nil {
		return nil, err
	}

	path := p.modelPath(req.Model) + ":streamGenerateContent"
	httpReq, err := p.newHTTPJSONRequest(ctx, http.MethodPost, path, key, vertexReq, url.Values{"alt": []string{"sse"}})
	if err != nil {
		return nil, err
	}

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vertex stream generate content: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read vertex error response: %w", readErr)
		}
		return nil, mapStatusError(resp.StatusCode, body)
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		parseVertexSSE(req.Model, resp.Body, chunks)
	}()
	return chunks, nil
}

func (p *VertexProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	if err := p.validateConfig(); err != nil {
		return nil, err
	}
	req, err := p.newJSONRequest(fasthttp.MethodGet, p.publisherModelsPath(), key, nil)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(req)

	resp, err := p.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vertex list models: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded listModelsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse vertex models response: %w", err)
	}

	models := make([]providers.Model, 0, len(decoded.Models))
	for _, model := range decoded.Models {
		id := model.Name
		if idx := strings.LastIndex(id, "/"); idx >= 0 {
			id = id[idx+1:]
		}
		models = append(models, providers.Model{
			ID:       id,
			Object:   "model",
			OwnedBy:  "google",
			Provider: providers.ProviderVertex,
		})
	}
	return models, nil
}

func (p *VertexProvider) validateConfig() error {
	if strings.TrimSpace(p.config.ProjectID) == "" || strings.TrimSpace(p.config.Location) == "" {
		return fmt.Errorf("%w: vertex project and location are required", ErrUnsupported)
	}
	return nil
}

func buildGenerateContentRequest(req *providers.ChatRequest) (*generateContentRequest, error) {
	vertexReq := &generateContentRequest{
		Contents: make([]content, 0, len(req.Messages)),
		GenerationConfig: &generationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: maxOutputTokens(req),
		},
	}
	if vertexReq.GenerationConfig.Temperature == nil &&
		vertexReq.GenerationConfig.TopP == nil &&
		vertexReq.GenerationConfig.MaxOutputTokens == nil {
		vertexReq.GenerationConfig = nil
	}

	if system, ok := req.System.(string); ok && system != "" {
		vertexReq.SystemInstruction = &content{Parts: []part{{Text: system}}}
	}

	for _, message := range req.Messages {
		text, err := textContent(message.Content)
		if err != nil {
			return nil, fmt.Errorf("build vertex content: %w", err)
		}
		if message.Role == "system" {
			vertexReq.SystemInstruction = &content{Parts: []part{{Text: text}}}
			continue
		}
		vertexReq.Contents = append(vertexReq.Contents, content{
			Role:  vertexRole(message.Role),
			Parts: []part{{Text: text}},
		})
	}
	return vertexReq, nil
}

func maxOutputTokens(req *providers.ChatRequest) *int {
	if req.MaxCompletionTokens != nil {
		return req.MaxCompletionTokens
	}
	return req.MaxTokens
}

func textContent(value any) (string, error) {
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("unsupported content type %T", value)
	}
	return text, nil
}

func vertexRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return "user"
}

func (p *VertexProvider) newJSONRequest(method, path string, key providers.Key, body any) (*fasthttp.Request, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(p.baseURL + path)
	req.Header.Set("Authorization", "Bearer "+key.Value)

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			fasthttp.ReleaseRequest(req)
			return nil, fmt.Errorf("marshal vertex request: %w", err)
		}
		req.SetBody(data)
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *VertexProvider) newHTTPJSONRequest(ctx context.Context, method, path string, key providers.Key, body any, extraQuery url.Values) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal vertex request: %w", err)
		}
		reader = strings.NewReader(string(data))
	}

	endpoint := p.baseURL + path
	if len(extraQuery) > 0 {
		endpoint += "?" + extraQuery.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("create vertex request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (p *VertexProvider) do(ctx context.Context, req *fasthttp.Request) (*fasthttp.Response, error) {
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

func (p *VertexProvider) modelPath(model string) string {
	return p.publisherModelsPath() + "/" + url.PathEscape(model)
}

func (p *VertexProvider) publisherModelsPath() string {
	return "/v1/projects/" + url.PathEscape(p.config.ProjectID) +
		"/locations/" + url.PathEscape(p.config.Location) +
		"/publishers/google/models"
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
		resp.Choices = append(resp.Choices, providers.Choice{
			Index: i,
			Message: providers.Message{
				Role:    "assistant",
				Content: textFromParts(candidate.Content.Parts),
			},
			FinishReason: &finishReason,
		})
	}
	return resp
}

func parseVertexSSE(model string, body io.Reader, chunks chan<- providers.StreamChunk) {
	scanner := bufio.NewScanner(body)
	var dataLines []string

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			done, failed := handleVertexSSEData(model, dataLines, chunks)
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
		_, failed := handleVertexSSEData(model, dataLines, chunks)
		if failed {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		chunks <- vertexStreamErrorChunk("upstream_stream_error")
	}
}

func handleVertexSSEData(model string, dataLines []string, chunks chan<- providers.StreamChunk) (bool, bool) {
	if len(dataLines) == 0 {
		return false, false
	}

	data := strings.Join(dataLines, "\n")
	if data == "[DONE]" {
		return true, false
	}

	var decoded generateContentResponse
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		chunks <- vertexStreamErrorChunk("upstream_stream_malformed")
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

func vertexStreamErrorChunk(code string) providers.StreamChunk {
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
		return fmt.Errorf("vertex error status %d: %s", statusCode, message)
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
