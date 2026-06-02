package gemini

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
	baseURL string
	client  *fasthttp.Client
}

func New(baseURL string) *GeminiProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &GeminiProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &fasthttp.Client{ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second},
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

func (p *GeminiProvider) ChatCompletionStream(context.Context, providers.Key, *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, fmt.Errorf("%w: gemini streaming is not implemented", ErrUnsupported)
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

	if system, ok := req.System.(string); ok && system != "" {
		geminiReq.SystemInstruction = &content{Parts: []part{{Text: system}}}
	}

	for _, message := range req.Messages {
		text, err := textContent(message.Content)
		if err != nil {
			return nil, fmt.Errorf("build gemini content: %w", err)
		}
		if message.Role == "system" {
			geminiReq.SystemInstruction = &content{Parts: []part{{Text: text}}}
			continue
		}
		geminiReq.Contents = append(geminiReq.Contents, content{
			Role:  geminiRole(message.Role),
			Parts: []part{{Text: text}},
		})
	}
	return geminiReq, nil
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

func (p *GeminiProvider) requestURI(path string, key providers.Key) string {
	uri := p.baseURL + path
	if key.AuthType == "oauth" {
		return uri
	}
	values := url.Values{}
	values.Set("key", key.Value)
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
		return fmt.Errorf("gemini error status %d: %s", resp.StatusCode(), message)
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
