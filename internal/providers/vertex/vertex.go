package vertex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	baseURL string
	config  Config
	client  *fasthttp.Client
}

func New(baseURL string, config Config) *VertexProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &VertexProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		config:  config,
		client:  &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
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

func (p *VertexProvider) ChatCompletionStream(context.Context, providers.Key, *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, fmt.Errorf("%w: vertex streaming is not implemented", ErrUnsupported)
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
	message := parseErrorMessage(resp.Body())

	switch resp.StatusCode() {
	case fasthttp.StatusUnauthorized, fasthttp.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrAuth, message)
	case fasthttp.StatusTooManyRequests:
		return &RateLimitError{Message: message}
	default:
		if resp.StatusCode() >= 500 {
			return fmt.Errorf("%w: %s", ErrServer, message)
		}
		return fmt.Errorf("vertex error status %d: %s", resp.StatusCode(), message)
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
