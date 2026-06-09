package schemas

// ChatRequest is the payload for POST /v1/chat/completions.
type ChatRequest struct {
	Model            string           `json:"model"`
	Messages         []Message        `json:"messages"`
	Temperature      *float64         `json:"temperature,omitempty"`
	MaxTokens        *int             `json:"max_tokens,omitempty"`
	TopP             *float64         `json:"top_p,omitempty"`
	N                *int             `json:"n,omitempty"`
	Stream           bool             `json:"stream"`
	Stop             []string         `json:"stop,omitempty"`
	PresencePenalty  *float64         `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64         `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int   `json:"logit_bias,omitempty"`
	User             string           `json:"user,omitempty"`
	Tools            []Tool           `json:"tools,omitempty"`
	ToolChoice       *ToolChoice      `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat  `json:"response_format,omitempty"`
	Seed             *int             `json:"seed,omitempty"`
}

// ChatResponse is the non-streaming response for chat completions.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Message represents a single message in the conversation.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       *string    `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID *string    `json:"tool_call_id,omitempty"`
}

// Choice represents one completion choice.
type Choice struct {
	Index        int       `json:"index"`
	Message      *Message  `json:"message,omitempty"`
	FinishReason string    `json:"finish_reason,omitempty"`
	Logprobs     *Logprobs `json:"logprobs,omitempty"`
}

// Tool defines a callable tool (function) available to the model.
type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// ToolCall represents a tool invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionDefinition describes the schema of a callable function.
type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// FunctionCall carries the arguments for a tool invocation.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolChoice controls how the model uses tools.
type ToolChoice struct {
	Type     string            `json:"type"`
	Function *FunctionChoice   `json:"function,omitempty"`
}

// FunctionChoice is used inside ToolChoice when targeting a specific function.
type FunctionChoice struct {
	Name string `json:"name"`
}

// ResponseFormat controls structured output mode.
type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// JSONSchema carries a JSON Schema definition for structured outputs.
type JSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema,omitempty"`
	Strict *bool          `json:"strict,omitempty"`
}

// Logprobs holds log-probability information.
type Logprobs struct {
	Content []LogprobContent `json:"content,omitempty"`
}

// LogprobContent is a single token's logprob data.
type LogprobContent struct {
	Token       string       `json:"token"`
	Logprob     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes,omitempty"`
	TopLogprobs []TopLogprob `json:"top_logprobs,omitempty"`
}

// TopLogprob is a candidate token in the top-logprobs list.
type TopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

// Usage tracks token consumption for a request.
type Usage struct {
	PromptTokens            int             `json:"prompt_tokens"`
	CompletionTokens        int             `json:"completion_tokens"`
	TotalTokens             int             `json:"total_tokens"`
	PromptTokensDetails     *TokensDetails  `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *TokensDetails  `json:"completion_tokens_details,omitempty"`
}

// TokensDetails breaks down token counts by modality.
type TokensDetails struct {
	AudioTokens  int `json:"audio_tokens,omitempty"`
	CachedTokens int `json:"cached_tokens,omitempty"`
	TextTokens   int `json:"text_tokens,omitempty"`
	ImageTokens  int `json:"image_tokens,omitempty"`
}

// StreamChunk is a single SSE chunk in a streaming chat completion.
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// StreamChoice is a choice inside a streaming chunk.
type StreamChoice struct {
	Index        int       `json:"index"`
	Delta        Message   `json:"delta"`
	FinishReason *string   `json:"finish_reason,omitempty"`
	Logprobs     *Logprobs `json:"logprobs,omitempty"`
}
