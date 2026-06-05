package bedrock

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

const (
	defaultRuntimeBaseURL = "https://bedrock-runtime.us-east-1.amazonaws.com"
	defaultModelBaseURL   = "https://bedrock.us-east-1.amazonaws.com"
	defaultRegion         = "us-east-1"
	serviceName           = "bedrock"
)

var (
	ErrAuth   = errors.New("bedrock auth error")
	ErrServer = errors.New("bedrock server error")
)

type BedrockProvider struct {
	baseURL      string
	modelBaseURL string
	region       string
	client       *http.Client
	now          func() time.Time
}

type credentials struct {
	accessKey    string
	secretKey    string
	sessionToken string
}

type listModelsResponse struct {
	ModelSummaries []modelSummary `json:"modelSummaries"`
}

type modelSummary struct {
	ModelID      string `json:"modelId"`
	ModelName    string `json:"modelName"`
	ProviderName string `json:"providerName"`
}

func New(baseURL string) *BedrockProvider {
	if baseURL == "" {
		baseURL = defaultRuntimeBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &BedrockProvider{
		baseURL:      baseURL,
		modelBaseURL: modelEndpointFor(baseURL),
		region:       defaultRegion,
		client:       &http.Client{Timeout: 60 * time.Second},
		now:          time.Now,
	}
}

func (p *BedrockProvider) Name() providers.ModelProvider {
	return providers.ProviderBedrock
}

func (p *BedrockProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	creds, err := parseCredentials(key.Value)
	if err != nil {
		return nil, fmt.Errorf("parse bedrock credentials: %w", err)
	}

	httpReq, err := p.newInvokeRequest(ctx, creds, req)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bedrock invoke model: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read bedrock response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mapError(resp.StatusCode, body)
	}

	var decoded converseResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("parse bedrock chat response: %w", err)
	}
	return toChatResponse(req.Model, decoded), nil
}

func (p *BedrockProvider) ChatCompletionStream(context.Context, providers.Key, *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, fmt.Errorf("bedrock chat completion stream: %w", providers.ErrStreamingUnsupported)
}

func (p *BedrockProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	creds, err := parseCredentials(key.Value)
	if err != nil {
		return nil, fmt.Errorf("parse bedrock credentials: %w", err)
	}

	req, err := p.newListModelsRequest(ctx, creds)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bedrock list models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read bedrock models response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mapError(resp.StatusCode, body)
	}

	var decoded listModelsResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("parse bedrock models response: %w", err)
	}
	models := make([]providers.Model, 0, len(decoded.ModelSummaries))
	for _, summary := range decoded.ModelSummaries {
		if summary.ModelID == "" {
			continue
		}
		models = append(models, providers.Model{
			ID:       summary.ModelID,
			Object:   "model",
			OwnedBy:  summary.ProviderName,
			Provider: providers.ProviderBedrock,
		})
	}
	return models, nil
}

func (p *BedrockProvider) newInvokeRequest(ctx context.Context, creds credentials, chatReq *providers.ChatRequest) (*http.Request, error) {
	converseReq, err := toConverseRequest(chatReq)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(converseReq)
	if err != nil {
		return nil, fmt.Errorf("marshal bedrock request: %w", err)
	}

	endpoint := p.baseURL + "/model/" + url.PathEscape(chatReq.Model) + "/converse"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create bedrock request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	if err := p.sign(req, creds, body); err != nil {
		return nil, fmt.Errorf("sign bedrock request: %w", err)
	}
	return req, nil
}

func (p *BedrockProvider) newListModelsRequest(ctx context.Context, creds credentials) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.modelBaseURL+"/foundation-models", nil)
	if err != nil {
		return nil, fmt.Errorf("create bedrock list models request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := p.sign(req, creds, nil); err != nil {
		return nil, fmt.Errorf("sign bedrock list models request: %w", err)
	}
	return req, nil
}

func modelEndpointFor(runtimeBaseURL string) string {
	if runtimeBaseURL == defaultRuntimeBaseURL {
		return defaultModelBaseURL
	}
	return strings.Replace(runtimeBaseURL, "://bedrock-runtime.", "://bedrock.", 1)
}

func toConverseRequest(req *providers.ChatRequest) (converseRequest, error) {
	maxTokens := 1024
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	} else if req.MaxCompletionTokens != nil {
		maxTokens = *req.MaxCompletionTokens
	}

	messages := make([]bedrockMessage, 0, len(req.Messages))
	for _, message := range req.Messages {
		messages = append(messages, bedrockMessage{Role: message.Role, Content: toContentBlocks(message.Content)})
	}

	stop, err := stopSequences(req.Stop)
	if err != nil {
		return converseRequest{}, err
	}

	return converseRequest{
		Messages: messages,
		InferenceConfig: &bedrockInferenceConfig{
			MaxTokens:     maxTokens,
			Temperature:   req.Temperature,
			TopP:          req.TopP,
			StopSequences: stop,
		},
	}, nil
}

func stopSequences(stop any) ([]string, error) {
	switch value := stop.(type) {
	case nil:
		return nil, nil
	case string:
		if value == "" {
			return nil, nil
		}
		return []string{value}, nil
	case []string:
		return nonEmptyStrings(value), nil
	case []any:
		sequences := make([]string, 0, len(value))
		for i, item := range value {
			sequence, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("bedrock request stop sequence %d: unsupported stop value %T", i, item)
			}
			if sequence != "" {
				sequences = append(sequences, sequence)
			}
		}
		return sequences, nil
	default:
		return nil, fmt.Errorf("bedrock request: unsupported stop type %T", stop)
	}
}

func nonEmptyStrings(values []string) []string {
	sequences := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			sequences = append(sequences, value)
		}
	}
	return sequences
}

func toContentBlocks(content any) []bedrockContentBlock {
	switch value := content.(type) {
	case string:
		return []bedrockContentBlock{{Text: value}}
	default:
		return []bedrockContentBlock{{Text: fmt.Sprint(value)}}
	}
}

func (p *BedrockProvider) sign(req *http.Request, creds credentials, body []byte) error {
	signedAt := p.now().UTC()
	amzDate := signedAt.Format("20060102T150405Z")
	shortDate := signedAt.Format("20060102")
	payloadHash := hashHex(body)

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if creds.sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.sessionToken)
	}

	signedHeaders, canonicalHeaders := canonicalHeaders(req)
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := strings.Join([]string{shortDate, p.region, serviceName, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hashHex([]byte(canonicalRequest)),
	}, "\n")

	signature := hmacHex(signingKey(creds.secretKey, shortDate, p.region, serviceName), stringToSign)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+creds.accessKey+"/"+credentialScope+", SignedHeaders="+signedHeaders+", Signature="+signature)
	return nil
}

func canonicalHeaders(req *http.Request) (string, string) {
	headers := map[string]string{"host": req.URL.Host}
	for key, values := range req.Header {
		lowerKey := strings.ToLower(key)
		if lowerKey == "accept" {
			continue
		}
		trimmed := make([]string, 0, len(values))
		for _, value := range values {
			trimmed = append(trimmed, strings.Join(strings.Fields(value), " "))
		}
		headers[lowerKey] = strings.Join(trimmed, ",")
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for _, key := range keys {
		canonical.WriteString(key)
		canonical.WriteByte(':')
		canonical.WriteString(headers[key])
		canonical.WriteByte('\n')
	}
	return strings.Join(keys, ";"), canonical.String()
}

func toChatResponse(model string, resp converseResponse) *providers.ChatResponse {
	content := ""
	for _, block := range resp.Output.Message.Content {
		content += block.Text
	}

	id := "bedrock-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	role := resp.Output.Message.Role
	if role == "" {
		role = "assistant"
	}

	var usage *providers.Usage
	if resp.Usage != nil {
		usage = &providers.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
		if usage.TotalTokens == 0 {
			usage.TotalTokens = resp.Usage.InputTokens + resp.Usage.OutputTokens
		}
	}

	return &providers.ChatResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []providers.Choice{{
			Index:        0,
			Message:      providers.Message{Role: role, Content: content},
			FinishReason: resp.StopReason,
		}},
		Usage: usage,
	}
}

func mapError(statusCode int, body []byte) error {
	message := parseErrorMessage(body)
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrAuth, message)
	default:
		if statusCode >= 500 {
			return fmt.Errorf("%w: %s", ErrServer, message)
		}
		return fmt.Errorf("bedrock error status %d: %s", statusCode, message)
	}
}

func parseErrorMessage(body []byte) string {
	var decoded errorResponse
	if err := json.Unmarshal(body, &decoded); err == nil && decoded.Message != "" {
		return decoded.Message
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "empty response"
	}
	return text
}

func parseCredentials(value string) (credentials, error) {
	parts := strings.SplitN(value, ":", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return credentials{}, fmt.Errorf("expected access_key:secret_key")
	}
	creds := credentials{accessKey: parts[0], secretKey: parts[1]}
	if len(parts) == 3 {
		creds.sessionToken = parts[2]
	}
	return creds, nil
}

func signingKey(secretKey, shortDate, region, service string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secretKey), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, service)
	return hmacSHA256(serviceKey, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func hmacHex(key []byte, data string) string {
	return hex.EncodeToString(hmacSHA256(key, data))
}

func hashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
