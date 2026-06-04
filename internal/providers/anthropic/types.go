package anthropic

import "encoding/json"

type anthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"`
	Messages      []anthropicMessage `json:"messages"`
	System        any                `json:"system,omitempty"`
	Stream        *bool              `json:"stream,omitempty"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Tools         []anthropicTool    `json:"tools,omitempty"`
	ToolChoice    *anthropicChoice   `json:"tool_choice,omitempty"`
	Thinking      any                `json:"thinking,omitempty"`
}

type anthropicChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Model      string                  `json:"model"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason *string                 `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type errorResponse struct {
	Error anthropicError `json:"error"`
}

type anthropicError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type modelsResponse struct {
	Data []modelResponse `json:"data"`
}

type modelResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

type streamEvent struct {
	Type         string                 `json:"type"`
	Index        int                    `json:"index,omitempty"`
	Message      *anthropicResponse     `json:"message,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Delta        streamDelta            `json:"delta,omitempty"`
	Usage        anthropicUsage         `json:"usage,omitempty"`
	Error        *anthropicError        `json:"error,omitempty"`
}

type streamDelta struct {
	Type        string  `json:"type"`
	Text        string  `json:"text,omitempty"`
	PartialJSON string  `json:"partial_json,omitempty"`
	StopReason  *string `json:"stop_reason,omitempty"`
}
