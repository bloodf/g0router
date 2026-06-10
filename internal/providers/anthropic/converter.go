package anthropic

import (
	"encoding/json"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// MessagesRequest is the Anthropic Messages API request body.
type MessagesRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Messages    []AnthMessage   `json:"messages"`
	Tools       []AnthTool      `json:"tools,omitempty"`
	ToolChoice  *AnthToolChoice `json:"tool_choice,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	TopK        *int            `json:"top_k,omitempty"`
	StopSequences []string      `json:"stop_sequences,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// AnthMessage is a single message in the Anthropic format.
type AnthMessage struct {
	Role    string          `json:"role"`
	Content []ContentBlock  `json:"content"`
}

// ContentBlock represents a piece of content in Anthropic messages.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"`
}

// AnthTool is the Anthropic tool definition.
type AnthTool struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

// AnthToolChoice controls tool usage.
type AnthToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// MessagesResponse is the Anthropic Messages API response.
type MessagesResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Model      string         `json:"model"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamEvent represents a single SSE event from Anthropic.
type StreamEvent struct {
	Type         string       `json:"type"`
	Index        int          `json:"index,omitempty"`
	Message      *MessagesResponse `json:"message,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Delta        *StreamDelta `json:"delta,omitempty"`
	Usage        *Usage       `json:"usage,omitempty"`
}

// StreamDelta carries incremental changes in a stream event.
type StreamDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

func joinSystemMessages(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += p
	}
	return result
}

// ConvertRequest transforms an OpenAI ChatRequest into an Anthropic MessagesRequest.
func ConvertRequest(req *schemas.ChatRequest) *MessagesRequest {
	anthReq := &MessagesRequest{
		Model:  req.Model,
		Stream: req.Stream,
	}

	body := map[string]any{}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		body["max_tokens"] = *req.MaxTokens
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.Thinking != nil {
		body["thinking"] = map[string]any{
			"budget_tokens": req.Thinking.BudgetTokens,
		}
	}
	anthReq.MaxTokens = translation.AdjustMaxTokens(body)
	if req.Temperature != nil {
		anthReq.Temperature = req.Temperature
	}
	if req.TopP != nil {
		anthReq.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		anthReq.StopSequences = req.Stop
	}

	// Extract system messages and convert remaining messages.
	var messages []schemas.Message
	var systemParts []string
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, m.Content)
			continue
		}
		messages = append(messages, m)
	}
	if len(systemParts) > 0 {
		anthReq.System = joinSystemMessages(systemParts)
	}

	anthReq.Messages = convertMessages(messages)

	if len(req.Tools) > 0 {
		anthReq.Tools = convertTools(req.Tools)
	}
	if req.ToolChoice != nil {
		anthReq.ToolChoice = convertToolChoice(req.ToolChoice)
	}

	return anthReq
}

func convertMessages(msgs []schemas.Message) []AnthMessage {
	var result []AnthMessage
	for _, m := range msgs {
		msg := AnthMessage{Role: m.Role}
		if m.Role == "tool" {
			// Tool result in OpenAI → tool_result in Anthropic
			msg.Role = "user"
			block := ContentBlock{Type: "tool_result", Text: m.Content}
			if m.ToolCallID != nil {
				block.ToolUseID = *m.ToolCallID
			}
			msg.Content = []ContentBlock{block}
		} else if len(m.ToolCalls) > 0 {
			// Assistant with tool calls
			for _, tc := range m.ToolCalls {
				msg.Content = append(msg.Content, ContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: json.RawMessage(tc.Function.Arguments),
				})
			}
			if m.Content != "" {
				msg.Content = append([]ContentBlock{{Type: "text", Text: m.Content}}, msg.Content...)
			}
		} else {
			msg.Content = []ContentBlock{{Type: "text", Text: m.Content}}
		}
		result = append(result, msg)
	}
	return result
}

func convertTools(tools []schemas.Tool) []AnthTool {
	var result []AnthTool
	for _, t := range tools {
		result = append(result, AnthTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}
	return result
}

func convertToolChoice(tc *schemas.ToolChoice) *AnthToolChoice {
	switch tc.Type {
	case "auto":
		return &AnthToolChoice{Type: "auto"}
	case "none":
		return &AnthToolChoice{Type: "none"}
	case "function", "required":
		if tc.Function != nil {
			return &AnthToolChoice{Type: "tool", Name: tc.Function.Name}
		}
		return &AnthToolChoice{Type: "any"}
	default:
		return &AnthToolChoice{Type: "auto"}
	}
}

// ConvertResponse transforms an Anthropic MessagesResponse into an OpenAI ChatResponse.
func ConvertResponse(anthResp *MessagesResponse) *schemas.ChatResponse {
	resp := &schemas.ChatResponse{
		ID:      anthResp.ID,
		Object:  "chat.completion",
		Created: 0,
		Model:   anthResp.Model,
		Choices: []schemas.Choice{{
			Index:        0,
			Message:      &schemas.Message{Role: "assistant"},
			FinishReason: mapStopReason(anthResp.StopReason),
		}},
		Usage: &schemas.Usage{
			PromptTokens:     anthResp.Usage.InputTokens,
			CompletionTokens: anthResp.Usage.OutputTokens,
			TotalTokens:      anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
		},
	}

	for _, block := range anthResp.Content {
		switch block.Type {
		case "text":
			resp.Choices[0].Message.Content += block.Text
		case "tool_use":
			resp.Choices[0].Message.ToolCalls = append(resp.Choices[0].Message.ToolCalls, schemas.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: schemas.FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}

	return resp
}

func mapStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}

// ConvertStreamEventToChunk transforms an Anthropic stream event into an OpenAI StreamChunk.
func ConvertStreamEventToChunk(event *StreamEvent, id, model string) *schemas.StreamChunk {
	chunk := &schemas.StreamChunk{
		ID:     id,
		Object: "chat.completion.chunk",
		Model:  model,
		Choices: []schemas.StreamChoice{{
			Index: 0,
			Delta: schemas.Message{Role: "assistant"},
		}},
	}

	switch event.Type {
	case "content_block_delta":
		if event.Delta != nil {
			switch event.Delta.Type {
			case "text_delta":
				chunk.Choices[0].Delta.Content = event.Delta.Text
			case "input_json_delta":
				chunk.Choices[0].Delta.ToolCalls = []schemas.ToolCall{{
					Type: "function",
					Function: schemas.FunctionCall{
						Arguments: event.Delta.PartialJSON,
					},
				}}
			}
		}
	case "message_delta":
		if event.Delta != nil && event.Delta.StopReason != "" {
			reason := mapStopReason(event.Delta.StopReason)
			chunk.Choices[0].FinishReason = &reason
		}
		if event.Usage != nil {
			chunk.Usage = &schemas.Usage{
				PromptTokens:     event.Usage.InputTokens,
				CompletionTokens: event.Usage.OutputTokens,
				TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
			}
		}
	}

	return chunk
}
