package bedrock

type converseRequest struct {
	Messages        []bedrockMessage        `json:"messages"`
	InferenceConfig *bedrockInferenceConfig `json:"inferenceConfig,omitempty"`
}

type bedrockMessage struct {
	Role    string                `json:"role"`
	Content []bedrockContentBlock `json:"content"`
}

type bedrockInferenceConfig struct {
	MaxTokens     int      `json:"maxTokens,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
	TopP          *float64 `json:"topP,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type converseResponse struct {
	Output     bedrockOutput         `json:"output"`
	StopReason *string               `json:"stopReason"`
	Usage      *bedrockResponseUsage `json:"usage"`
}

type bedrockOutput struct {
	Message bedrockMessage `json:"message"`
}

type bedrockContentBlock struct {
	Text string `json:"text,omitempty"`
}

type bedrockResponseUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

type errorResponse struct {
	Message string `json:"message"`
}
