package providers

import "encoding/json"

// ModelProvider identifies an upstream LLM provider.
type ModelProvider string

const (
	ProviderOpenAI        ModelProvider = "openai"
	ProviderAnthropic     ModelProvider = "anthropic"
	ProviderGemini        ModelProvider = "gemini"
	ProviderGroq          ModelProvider = "groq"
	ProviderCerebras      ModelProvider = "cerebras"
	ProviderMistral       ModelProvider = "mistral"
	ProviderOllama        ModelProvider = "ollama"
	ProviderBedrock       ModelProvider = "bedrock"
	ProviderAzure         ModelProvider = "azure"
	ProviderVertex        ModelProvider = "vertex"
	ProviderOpenRouter    ModelProvider = "openrouter"
	ProviderDeepSeek      ModelProvider = "deepseek"
	ProviderPerplexity    ModelProvider = "perplexity"
	ProviderFireworks     ModelProvider = "fireworks"
	ProviderTogether      ModelProvider = "together"
	ProviderNVIDIA        ModelProvider = "nvidia"
	ProviderHuggingFace   ModelProvider = "huggingface"
	ProviderCloudflare    ModelProvider = "cloudflare-ai-gateway"
	ProviderCohere        ModelProvider = "cohere"
	ProviderReplicate     ModelProvider = "replicate"
	ProviderXAI           ModelProvider = "xai"
	ProviderNebius        ModelProvider = "nebius"
	ProviderMiniMax       ModelProvider = "minimax"
	ProviderQwen          ModelProvider = "qwen"
	ProviderVercelGateway ModelProvider = "vercel-ai-gateway"
	ProviderLiteLLM       ModelProvider = "litellm"
	ProviderVLLM          ModelProvider = "vllm"
	ProviderLMStudio      ModelProvider = "lm-studio"
	ProviderGitHubCopilot ModelProvider = "github-copilot"
	ProviderCursor        ModelProvider = "cursor"
	ProviderAlibaba       ModelProvider = "alibaba"
	ProviderKimi          ModelProvider = "kimi"
	ProviderQianfan       ModelProvider = "qianfan"
	ProviderZhipu         ModelProvider = "zhipu"
)

func (p ModelProvider) String() string {
	return string(p)
}

// Key holds credentials for a single provider request.
type Key struct {
	Value     string        `json:"value"`
	Provider  ModelProvider `json:"provider"`
	ConnID    string        `json:"conn_id"`
	AuthType  string        `json:"auth_type"`
	AccountID string        `json:"account_id,omitempty"`
}

// Model represents an available model.
type Model struct {
	ID       string        `json:"id"`
	Object   string        `json:"object"`
	Created  int64         `json:"created"`
	OwnedBy  string        `json:"owned_by"`
	Provider ModelProvider `json:"-"`
}

type ChatRequest struct {
	Model               string    `json:"model"`
	Messages            []Message `json:"messages"`
	Stream              *bool     `json:"stream,omitempty"`
	Temperature         *float64  `json:"temperature,omitempty"`
	TopP                *float64  `json:"top_p,omitempty"`
	MaxTokens           *int      `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int      `json:"max_completion_tokens,omitempty"`
	Stop                any       `json:"stop,omitempty"`
	Tools               []Tool    `json:"tools,omitempty"`
	ToolChoice          any       `json:"tool_choice,omitempty"`
	ResponseFormat      any       `json:"response_format,omitempty"`
	Seed                *int      `json:"seed,omitempty"`
	FrequencyPenalty    *float64  `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64  `json:"presence_penalty,omitempty"`
	N                   *int      `json:"n,omitempty"`
	User                *string   `json:"user,omitempty"`
	StreamOptions       any       `json:"stream_options,omitempty"`
	System              any       `json:"system,omitempty"`
	Thinking            any       `json:"thinking,omitempty"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	Name       *string    `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID *string    `json:"tool_call_id,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatResponse struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	Choices           []Choice      `json:"choices"`
	Usage             *Usage        `json:"usage,omitempty"`
	SystemFingerprint *string       `json:"system_fingerprint,omitempty"`
	Provider          ModelProvider `json:"-"`
	ConnectionID      string        `json:"-"`
	AuthType          string        `json:"-"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason *string `json:"finish_reason"`
}

type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type StreamChunk struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []StreamChoice `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
	SystemFingerprint *string        `json:"system_fingerprint,omitempty"`
	Error             *StreamError   `json:"error,omitempty"`
}

type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type StreamError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type StreamDelta struct {
	Role      *string    `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
