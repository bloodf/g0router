package schemas

// ModelProvider identifies a specific LLM provider backend.
type ModelProvider string

// Supported provider constants.
const (
	ProviderOpenAI     ModelProvider = "openai"
	ProviderAnthropic  ModelProvider = "anthropic"
	ProviderGemini     ModelProvider = "gemini"
	ProviderGroq       ModelProvider = "groq"
	ProviderMistral    ModelProvider = "mistral"
	ProviderCohere     ModelProvider = "cohere"
	ProviderFireworks  ModelProvider = "fireworks"
	ProviderTogether   ModelProvider = "together"
	ProviderDeepSeek   ModelProvider = "deepseek"
	ProviderMiniMax    ModelProvider = "minimax"
	ProviderOllama     ModelProvider = "ollama"
	ProviderBedrock    ModelProvider = "bedrock"
	ProviderVertex     ModelProvider = "vertex"
	ProviderOpenRouter ModelProvider = "openrouter"
)

// GatewayContext carries per-request metadata through the gateway.
type GatewayContext struct {
	RequestID string
}

// Key represents a provider API key.
type Key struct {
	ID       string
	Provider string
	Value    string
}

// NetworkConfig holds HTTP/network tuning for a provider.
type NetworkConfig struct {
	Timeout    int    `json:"timeout"`
	ProxyURL   string `json:"proxy_url,omitempty"`
	MaxRetries int    `json:"max_retries,omitempty"`
}

// PostHookRunner is called after a response is received.
type PostHookRunner interface {
	Run(ctx *GatewayContext, response any) error
}

// ListModelsResponse is the payload for GET /v1/models.
type ListModelsResponse struct {
	Object string       `json:"object"`
	Data   []ModelEntry `json:"data"`
}

// ModelEntry describes a single model in the list.
type ModelEntry struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// TokenCountResponse is the result of a token-count operation.
type TokenCountResponse struct {
	Tokens int `json:"tokens"`
}

// Provider is the unified interface that every backend must implement.
type Provider interface {
	GetProvider() ModelProvider
	SetNetworkConfig(config NetworkConfig)

	ListModels(ctx *GatewayContext, key Key) (*ListModelsResponse, *ProviderError)

	ChatCompletion(ctx *GatewayContext, key Key, request *ChatRequest) (*ChatResponse, *ProviderError)
	ChatCompletionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ChatRequest) (chan *StreamChunk, *ProviderError)

	TextCompletion(ctx *GatewayContext, key Key, request *TextCompletionRequest) (*TextCompletionResponse, *ProviderError)
	TextCompletionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *TextCompletionRequest) (chan *StreamChunk, *ProviderError)

	Responses(ctx *GatewayContext, key Key, request *ResponsesRequest) (*ResponsesResponse, *ProviderError)
	ResponsesStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ResponsesRequest) (chan *StreamChunk, *ProviderError)

	Embedding(ctx *GatewayContext, key Key, request *EmbeddingRequest) (*EmbeddingResponse, *ProviderError)

	ImageGeneration(ctx *GatewayContext, key Key, request *ImageGenerationRequest) (*ImageGenerationResponse, *ProviderError)
	ImageGenerationStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ImageGenerationRequest) (chan *StreamChunk, *ProviderError)
	ImageEdit(ctx *GatewayContext, key Key, request *ImageEditRequest) (*ImageGenerationResponse, *ProviderError)
	ImageVariation(ctx *GatewayContext, key Key, request *ImageVariationRequest) (*ImageGenerationResponse, *ProviderError)

	Speech(ctx *GatewayContext, key Key, request *SpeechRequest) (*SpeechResponse, *ProviderError)
	SpeechStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *SpeechRequest) (chan *StreamChunk, *ProviderError)
	Transcription(ctx *GatewayContext, key Key, request *TranscriptionRequest) (*TranscriptionResponse, *ProviderError)
	TranscriptionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *TranscriptionRequest) (chan *StreamChunk, *ProviderError)

	FileUpload(ctx *GatewayContext, key Key, request *FileUploadRequest) (*FileObject, *ProviderError)
	FileList(ctx *GatewayContext, key Key) (*FileListResponse, *ProviderError)
	FileRetrieve(ctx *GatewayContext, key Key, fileID string) (*FileObject, *ProviderError)
	FileDelete(ctx *GatewayContext, key Key, fileID string) (*FileDeleteResponse, *ProviderError)
	FileContent(ctx *GatewayContext, key Key, fileID string) ([]byte, *ProviderError)

	BatchCreate(ctx *GatewayContext, key Key, request *BatchCreateRequest) (*Batch, *ProviderError)
	BatchList(ctx *GatewayContext, key Key) (*BatchListResponse, *ProviderError)
	BatchRetrieve(ctx *GatewayContext, key Key, batchID string) (*Batch, *ProviderError)
	BatchCancel(ctx *GatewayContext, key Key, batchID string) (*Batch, *ProviderError)

	CountTokens(ctx *GatewayContext, key Key, request *ChatRequest) (*TokenCountResponse, *ProviderError)
}
