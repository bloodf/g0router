package translate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
)

// ErrAnthropicTranslate marks an Anthropic feature that has no representable
// OpenAI equivalent, so the handler can answer 501 rather than 400.
var ErrAnthropicTranslate = errors.New("messages translation unsupported")

type AnthropicMessageContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type AnthropicMessageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicMessageBody struct {
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
	Role       string                    `json:"role"`
	Model      string                    `json:"model"`
	Content    []AnthropicMessageContent `json:"content"`
	StopReason *string                   `json:"stop_reason,omitempty"`
	Usage      AnthropicMessageUsage     `json:"usage"`
}

// AnthropicRequestEnvelope mirrors the inbound Anthropic /v1/messages body
// closely enough to translate its content blocks into the internal/OpenAI
// ChatRequest shape while preserving tool-call identifiers.
type AnthropicRequestEnvelope struct {
	Model      string                    `json:"model"`
	Messages   []AnthropicInboundMessage `json:"messages"`
	Stream     *bool                     `json:"stream,omitempty"`
	Tools      []AnthropicInboundTool    `json:"tools"`
	ToolChoice json.RawMessage           `json:"tool_choice"`
}

type AnthropicInboundTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	Type        string          `json:"type"`
}

type AnthropicInboundMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type AnthropicInboundBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
}

// AnthropicMessageResponse converts an internal/OpenAI ChatResponse into the
// Anthropic messages response shape.
func AnthropicMessageResponse(resp *providers.ChatResponse) AnthropicMessageBody {
	body := AnthropicMessageBody{Type: "message"}
	if resp == nil {
		return body
	}
	body.ID = resp.ID
	body.Model = resp.Model
	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		body.Role = choice.Message.Role
		body.StopReason = AnthropicStopReason(choice.FinishReason)
		if text := MessageContentText(choice.Message.Content); text != "" {
			body.Content = append(body.Content, AnthropicMessageContent{Type: "text", Text: text})
		}
		for _, toolCall := range choice.Message.ToolCalls {
			body.Content = append(body.Content, AnthropicMessageContent{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: ToolCallInput(toolCall.Function.Arguments),
			})
		}
	}
	if body.Role == "" {
		body.Role = "assistant"
	}
	if resp.Usage != nil {
		body.Usage = AnthropicMessageUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
	}
	return body
}

// AnthropicMessagesRequest converts an inbound Anthropic messages body into the
// internal ChatRequest. Anthropic tool_use blocks map to assistant tool_calls
// (preserving ids), and tool_result blocks map to tool-role messages whose
// tool_call_id is the originating tool_use_id, so identifiers survive the
// translation. Bodies whose content is a plain string fall through to the
// standard decoder unchanged.
func AnthropicMessagesRequest(body []byte) (*providers.ChatRequest, error) {
	var req providers.ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("unmarshal chat request: %w", err)
	}

	var envelope AnthropicRequestEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal anthropic envelope: %w", err)
	}

	messages := make([]providers.Message, 0, len(envelope.Messages))
	for _, inbound := range envelope.Messages {
		translated, err := TranslateAnthropicInboundMessage(inbound)
		if err != nil {
			return nil, fmt.Errorf("translate inbound message: %w", err)
		}
		messages = append(messages, translated...)
	}
	req.Messages = messages

	tools, err := TranslateAnthropicTools(envelope.Tools)
	if err != nil {
		return nil, fmt.Errorf("translate anthropic tools: %w", err)
	}
	req.Tools = tools

	choice, err := TranslateAnthropicToolChoice(envelope.ToolChoice)
	if err != nil {
		return nil, fmt.Errorf("translate anthropic tool choice: %w", err)
	}
	req.ToolChoice = choice

	return &req, nil
}

// TranslateAnthropicTools converts inbound Anthropic tool definitions
// ({name, description, input_schema}) into internal/OpenAI function tools
// ({type:"function", function:{name, description, parameters: <input_schema>}}).
// Server-side tools (identified by a non-empty type, e.g. web_search) have no
// OpenAI function equivalent and are rejected specifically.
func TranslateAnthropicTools(tools []AnthropicInboundTool) ([]providers.Tool, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	out := make([]providers.Tool, 0, len(tools))
	for i, tool := range tools {
		if tool.Type != "" && tool.Type != "custom" && tool.Type != "function" {
			return nil, fmt.Errorf("%w: tool %d type %q has no OpenAI equivalent", ErrAnthropicTranslate, i, tool.Type)
		}
		out = append(out, providers.Tool{
			Type: "function",
			Function: providers.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}
	return out, nil
}

// TranslateAnthropicToolChoice maps an Anthropic tool_choice to its OpenAI
// equivalent: {type:auto}->"auto", {type:any}->"required",
// {type:tool,name:X}->{type:"function",function:{name:X}}. Absent/null leaves
// it unset. A bare string passes through. Unknown variants are rejected.
func TranslateAnthropicToolChoice(raw json.RawMessage) (any, error) {
	trimmed := BytesTrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return nil, fmt.Errorf("unmarshal tool choice string: %w", err)
		}
		return s, nil
	}
	var choice struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(trimmed, &choice); err != nil {
		return nil, fmt.Errorf("unmarshal tool choice object: %w", err)
	}
	switch choice.Type {
	case "auto":
		return "auto", nil
	case "any":
		return "required", nil
	case "tool":
		return map[string]any{
			"type":     "function",
			"function": map[string]any{"name": choice.Name},
		}, nil
	default:
		return nil, fmt.Errorf("%w: tool_choice type %q has no OpenAI equivalent", ErrAnthropicTranslate, choice.Type)
	}
}

func TranslateAnthropicInboundMessage(inbound AnthropicInboundMessage) ([]providers.Message, error) {
	trimmed := BytesTrimSpace(inbound.Content)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		var content any
		if len(trimmed) > 0 && string(trimmed) != "null" {
			if err := json.Unmarshal(trimmed, &content); err != nil {
				return nil, fmt.Errorf("unmarshal inbound content: %w", err)
			}
		}
		return []providers.Message{{Role: inbound.Role, Content: content}}, nil
	}

	var blocks []AnthropicInboundBlock
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return nil, fmt.Errorf("unmarshal inbound blocks: %w", err)
	}

	var textParts []string
	var toolCalls []providers.ToolCall
	var results []providers.Message
	for _, block := range blocks {
		switch block.Type {
		case "", "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		case "tool_use":
			toolCalls = append(toolCalls, providers.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: providers.ToolCallFunc{
					Name:      block.Name,
					Arguments: AnthropicToolInputArguments(block.Input),
				},
			})
		case "tool_result":
			id := block.ToolUseID
			results = append(results, providers.Message{
				Role:       "tool",
				Content:    AnthropicToolResultText(block.Content),
				ToolCallID: &id,
			})
		}
	}

	var out []providers.Message
	if len(textParts) > 0 || len(toolCalls) > 0 || len(results) == 0 {
		message := providers.Message{Role: inbound.Role, Content: strings.Join(textParts, "")}
		if len(toolCalls) > 0 {
			message.ToolCalls = toolCalls
		}
		out = append(out, message)
	}
	out = append(out, results...)
	return out, nil
}

func AnthropicToolInputArguments(input json.RawMessage) string {
	trimmed := BytesTrimSpace(input)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return "{}"
	}
	return string(trimmed)
}

func AnthropicToolResultText(content json.RawMessage) string {
	trimmed := BytesTrimSpace(content)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return ""
	}
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err == nil {
			return text
		}
		return string(trimmed)
	}
	if trimmed[0] == '[' {
		var blocks []AnthropicInboundBlock
		if err := json.Unmarshal(trimmed, &blocks); err == nil {
			var parts []string
			for _, block := range blocks {
				if block.Text != "" {
					parts = append(parts, block.Text)
				}
			}
			return strings.Join(parts, "")
		}
	}
	return string(trimmed)
}

// RejectUnsupportedAnthropicMessageShape rejects only content block types we
// cannot represent. Native tool definitions and tool_choice are now translated
// (see TranslateAnthropicTools / TranslateAnthropicToolChoice), so they are no
// longer rejected here; genuinely-unsupported tool variants are reported during
// translation instead.
func RejectUnsupportedAnthropicMessageShape(body []byte) error {
	var req struct {
		Messages []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil
	}
	for i, message := range req.Messages {
		if err := RejectUnsupportedAnthropicContent(message.Content); err != nil {
			return fmt.Errorf("messages content %d: %w", i, err)
		}
	}
	return nil
}

// RejectUnsupportedAnthropicContent rejects a single content value if it
// contains block types that cannot be represented in the OpenAI shape.
func RejectUnsupportedAnthropicContent(raw json.RawMessage) error {
	trimmed := BytesTrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] == '"' || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] != '[' {
		return fmt.Errorf("unsupported content shape")
	}
	var blocks []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return nil
	}
	for i, block := range blocks {
		switch block.Type {
		case "", "text", "tool_use", "tool_result":
		default:
			return fmt.Errorf("unsupported content block %d type %q", i, block.Type)
		}
	}
	return nil
}

func BytesTrimSpace(raw []byte) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

func AnthropicStopReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	if *reason == "tool_calls" {
		toolUse := "tool_use"
		return &toolUse
	}
	return reason
}

func AnthropicStreamStopReason(reason *string) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return *reason
	}
}

func ToolCallInput(arguments string) json.RawMessage {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(trimmed), &object); err == nil {
		return json.RawMessage(trimmed)
	}
	wrapped, err := json.Marshal(map[string]string{"arguments": arguments})
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return wrapped
}

func MessageContentText(content any) string {
	switch value := content.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}
