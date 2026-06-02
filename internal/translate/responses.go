package translate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
)

type ResponsesRequest struct {
	Model           string           `json:"model"`
	Input           []ResponsesInput `json:"input,omitempty"`
	Instructions    *string          `json:"instructions,omitempty"`
	Stream          *bool            `json:"stream,omitempty"`
	Temperature     *float64         `json:"temperature,omitempty"`
	TopP            *float64         `json:"top_p,omitempty"`
	MaxOutputTokens *int             `json:"max_output_tokens,omitempty"`
	Tools           []ResponsesTool  `json:"tools,omitempty"`
	Text            any              `json:"text,omitempty"`
	ToolChoice      any              `json:"tool_choice,omitempty"`
}

type ResponsesInput struct {
	Role    string             `json:"role"`
	Content []ResponsesContent `json:"content"`
}

type ResponsesContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type ResponsesResponse struct {
	ID         string            `json:"id"`
	Object     string            `json:"object"`
	CreatedAt  int64             `json:"created_at"`
	Model      string            `json:"model"`
	Status     string            `json:"status"`
	OutputText string            `json:"output_text,omitempty"`
	Output     []ResponsesOutput `json:"output,omitempty"`
	Usage      *ResponsesUsage   `json:"usage,omitempty"`
}

type ResponsesOutput struct {
	Type    string             `json:"type"`
	Role    string             `json:"role,omitempty"`
	Content []ResponsesContent `json:"content,omitempty"`
}

type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func OpenAIChatToResponses(req *providers.ChatRequest) (*ResponsesRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("openai chat to responses: nil request")
	}

	translated := &ResponsesRequest{
		Model:           req.Model,
		Stream:          req.Stream,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
		MaxOutputTokens: maxResponsesOutputTokens(req),
		Text:            req.ResponseFormat,
		ToolChoice:      req.ToolChoice,
		Input:           make([]ResponsesInput, 0, len(req.Messages)),
	}

	var instructions []string
	if req.System != nil {
		text, err := responseText(req.System)
		if err != nil {
			return nil, fmt.Errorf("openai chat to responses system: %w", err)
		}
		if text != "" {
			instructions = append(instructions, text)
		}
	}

	for i, message := range req.Messages {
		if message.Role == "system" {
			text, err := responseText(message.Content)
			if err != nil {
				return nil, fmt.Errorf("openai chat to responses message %d: %w", i, err)
			}
			if text != "" {
				instructions = append(instructions, text)
			}
			continue
		}

		content, err := responseContent(message.Content)
		if err != nil {
			return nil, fmt.Errorf("openai chat to responses message %d: %w", i, err)
		}
		translated.Input = append(translated.Input, ResponsesInput{Role: message.Role, Content: content})
	}

	if len(instructions) > 0 {
		joined := strings.Join(instructions, "\n\n")
		translated.Instructions = &joined
	}

	translated.Tools = responseTools(req.Tools)
	return translated, nil
}

func ResponsesToOpenAIChat(resp *ResponsesResponse) providers.ChatResponse {
	if resp == nil {
		return providers.ChatResponse{Object: "chat.completion"}
	}

	choices := make([]providers.Choice, 0, len(resp.Output))
	for _, output := range resp.Output {
		if output.Type != "message" {
			continue
		}
		text := responsesContentText(output.Content)
		choices = append(choices, providers.Choice{
			Index: len(choices),
			Message: providers.Message{
				Role:    output.Role,
				Content: text,
			},
		})
	}
	if len(choices) == 0 && resp.OutputText != "" {
		choices = append(choices, providers.Choice{
			Message: providers.Message{Role: "assistant", Content: resp.OutputText},
		})
	}

	return providers.ChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.CreatedAt,
		Model:   resp.Model,
		Choices: choices,
		Usage:   responseUsage(resp.Usage),
	}
}

func maxResponsesOutputTokens(req *providers.ChatRequest) *int {
	if req.MaxCompletionTokens != nil {
		return req.MaxCompletionTokens
	}
	return req.MaxTokens
}

func responseContent(content any) ([]ResponsesContent, error) {
	text, err := responseText(content)
	if err != nil {
		return nil, err
	}
	if text == "" {
		return nil, nil
	}
	return []ResponsesContent{{Type: "input_text", Text: text}}, nil
}

func responseText(content any) (string, error) {
	switch value := content.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported content type %T", content)
	}
}

func responseTools(tools []providers.Tool) []ResponsesTool {
	if len(tools) == 0 {
		return nil
	}

	translated := make([]ResponsesTool, 0, len(tools))
	for _, tool := range tools {
		translated = append(translated, ResponsesTool{
			Type:        tool.Type,
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  append(json.RawMessage(nil), tool.Function.Parameters...),
		})
	}
	return translated
}

func responsesContentText(content []ResponsesContent) string {
	var builder strings.Builder
	for _, part := range content {
		if part.Type == "output_text" || part.Type == "text" {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

func responseUsage(usage *ResponsesUsage) *providers.Usage {
	if usage == nil {
		return nil
	}
	return &providers.Usage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
	}
}
