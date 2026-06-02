package bedrock

type invokeRequest struct {
	AnthropicVersion string           `json:"anthropic_version"`
	Messages         []bedrockMessage `json:"messages"`
	MaxTokens        int              `json:"max_tokens"`
	Temperature      *float64         `json:"temperature,omitempty"`
	TopP             *float64         `json:"top_p,omitempty"`
	StopSequences    any              `json:"stop_sequences,omitempty"`
}

type bedrockMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type invokeResponse struct {
	ID         string                `json:"id"`
	Role       string                `json:"role"`
	Content    []bedrockContentBlock `json:"content"`
	StopReason *string               `json:"stop_reason"`
	Usage      *bedrockResponseUsage `json:"usage"`
}

type bedrockContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type bedrockResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type errorResponse struct {
	Message string `json:"message"`
}
