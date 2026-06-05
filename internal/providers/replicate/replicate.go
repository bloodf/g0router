package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

const (
	defaultBaseURL      = "https://api.replicate.com"
	defaultPollInterval = 500 * time.Millisecond
	defaultMaxPolls     = 120
)

type Config struct {
	BaseURL      string
	HTTPClient   *http.Client
	PollInterval time.Duration
	MaxPolls     int
}

type Provider struct {
	baseURL      string
	client       *http.Client
	pollInterval time.Duration
	maxPolls     int
}

type predictionCreateRequest struct {
	Model string         `json:"model"`
	Input map[string]any `json:"input"`
}

type predictionResponse struct {
	ID     string            `json:"id"`
	Status string            `json:"status"`
	Output any               `json:"output"`
	Error  any               `json:"error"`
	URLs   map[string]string `json:"urls"`
}

func New(config Config) *Provider {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	pollInterval := config.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	maxPolls := config.MaxPolls
	if maxPolls <= 0 {
		maxPolls = defaultMaxPolls
	}
	return &Provider{baseURL: baseURL, client: client, pollInterval: pollInterval, maxPolls: maxPolls}
}

func NewDefault() *Provider {
	return New(Config{})
}

func (p *Provider) Name() providers.ModelProvider {
	return providers.ProviderReplicate
}

func (p *Provider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("replicate request: nil chat request")
	}
	createReq := predictionCreateRequest{
		Model: req.Model,
		Input: map[string]any{"prompt": flattenMessages(req.Messages)},
	}
	prediction, err := p.createPrediction(ctx, key, createReq)
	if err != nil {
		return nil, fmt.Errorf("replicate chat completion: %w", err)
	}
	terminal, err := p.waitPrediction(ctx, key, prediction)
	if err != nil {
		return nil, fmt.Errorf("replicate chat completion: %w", err)
	}
	content, err := predictionOutputText(terminal.Output)
	if err != nil {
		return nil, fmt.Errorf("replicate chat completion: %w", err)
	}
	finish := "stop"
	return &providers.ChatResponse{
		ID:           nonEmpty(terminal.ID, prediction.ID),
		Object:       "chat.completion",
		Created:      time.Now().Unix(),
		Model:        req.Model,
		Provider:     providers.ProviderReplicate,
		ConnectionID: key.ConnID,
		AuthType:     key.AuthType,
		Choices: []providers.Choice{{
			Index:        0,
			Message:      providers.Message{Role: "assistant", Content: content},
			FinishReason: &finish,
		}},
	}, nil
}

func (p *Provider) ChatCompletionStream(context.Context, providers.Key, *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, fmt.Errorf("replicate streaming unsupported")
}

func (p *Provider) ListModels(context.Context, providers.Key) ([]providers.Model, error) {
	return nil, fmt.Errorf("replicate list models unsupported")
}

func (p *Provider) createPrediction(ctx context.Context, key providers.Key, body predictionCreateRequest) (predictionResponse, error) {
	var out predictionResponse
	if err := p.doJSON(ctx, http.MethodPost, "/v1/predictions", key, body, &out); err != nil {
		return predictionResponse{}, err
	}
	return out, nil
}

func (p *Provider) waitPrediction(ctx context.Context, key providers.Key, prediction predictionResponse) (predictionResponse, error) {
	current := prediction
	for attempt := 0; attempt < p.maxPolls; attempt++ {
		switch strings.ToLower(strings.TrimSpace(current.Status)) {
		case "succeeded":
			return current, nil
		case "failed", "canceled":
			return predictionResponse{}, fmt.Errorf("prediction failed: %v", current.Error)
		}

		getPath := current.URLs["get"]
		if getPath == "" {
			getPath = "/v1/predictions/" + current.ID
		}
		if current.ID == "" && current.URLs["get"] == "" {
			return predictionResponse{}, fmt.Errorf("prediction missing id and poll URL")
		}
		if err := sleepContext(ctx, p.pollInterval); err != nil {
			return predictionResponse{}, err
		}
		var next predictionResponse
		if err := p.doJSON(ctx, http.MethodGet, getPath, key, nil, &next); err != nil {
			return predictionResponse{}, err
		}
		current = next
	}
	return predictionResponse{}, fmt.Errorf("prediction timed out after %d polls", p.maxPolls)
}

func (p *Provider) doJSON(ctx context.Context, method, path string, key providers.Key, body any, out any) error {
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
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (p *Provider) newRequest(ctx context.Context, method, path string, key providers.Key, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal replicate request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	url := path
	if strings.HasPrefix(path, "/") {
		url = p.baseURL + path
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("create replicate request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func flattenMessages(messages []providers.Message) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		content := contentText(message.Content)
		if content == "" {
			continue
		}
		role := strings.TrimSpace(message.Role)
		if role == "" {
			role = "user"
		}
		parts = append(parts, role+": "+content)
	}
	return strings.Join(parts, "\n")
}

func contentText(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	case nil:
		return ""
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(data)
	}
}

func predictionOutputText(output any) (string, error) {
	switch value := output.(type) {
	case string:
		return value, nil
	case []any:
		var b strings.Builder
		for _, item := range value {
			typed, ok := item.(string)
			if !ok {
				return "", fmt.Errorf("unsupported prediction output item %T", item)
			}
			b.WriteString(typed)
		}
		return b.String(), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("unsupported prediction output %T", output)
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
