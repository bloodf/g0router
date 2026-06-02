package translate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
)

type GeminiGenerateContentRequest struct {
	Contents          []GeminiContent         `json:"contents"`
	SystemInstruction *GeminiContent          `json:"system_instruction,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools             []GeminiTool            `json:"tools,omitempty"`
}

type GeminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type GeminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

type GeminiFunctionDeclaration struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

func OpenAIToGemini(req *providers.ChatRequest) (*GeminiGenerateContentRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("openai to gemini: nil request")
	}

	geminiReq := &GeminiGenerateContentRequest{
		Contents: make([]GeminiContent, 0, len(req.Messages)),
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: maxGeminiOutputTokens(req),
		},
	}
	if emptyGenerationConfig(geminiReq.GenerationConfig) {
		geminiReq.GenerationConfig = nil
	}

	tools, err := geminiTools(req.Tools)
	if err != nil {
		return nil, fmt.Errorf("openai to gemini: %w", err)
	}
	geminiReq.Tools = tools

	var systemParts []string
	if req.System != nil {
		text, err := textFromContent(req.System)
		if err != nil {
			return nil, fmt.Errorf("openai to gemini system: %w", err)
		}
		if text != "" {
			systemParts = append(systemParts, text)
		}
	}

	for i, message := range req.Messages {
		if message.Role == "system" {
			text, err := textFromContent(message.Content)
			if err != nil {
				return nil, fmt.Errorf("openai to gemini message %d: %w", i, err)
			}
			if text != "" {
				systemParts = append(systemParts, text)
			}
			continue
		}

		content, err := geminiContent(message)
		if err != nil {
			return nil, fmt.Errorf("openai to gemini message %d: %w", i, err)
		}
		geminiReq.Contents = append(geminiReq.Contents, content)
	}

	if len(systemParts) > 0 {
		geminiReq.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: strings.Join(systemParts, "\n\n")}},
		}
	}
	return geminiReq, nil
}

func maxGeminiOutputTokens(req *providers.ChatRequest) *int {
	if req.MaxCompletionTokens != nil {
		return req.MaxCompletionTokens
	}
	return req.MaxTokens
}

func emptyGenerationConfig(config *GeminiGenerationConfig) bool {
	return config.Temperature == nil && config.TopP == nil && config.MaxOutputTokens == nil
}

func geminiTools(tools []providers.Tool) ([]GeminiTool, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	declarations := make([]GeminiFunctionDeclaration, 0, len(tools))
	for i, tool := range tools {
		if tool.Type != "function" {
			return nil, fmt.Errorf("tool %d: unsupported tool type %q", i, tool.Type)
		}
		declarations = append(declarations, GeminiFunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  append(json.RawMessage(nil), tool.Function.Parameters...),
		})
	}
	return []GeminiTool{{FunctionDeclarations: declarations}}, nil
}

func geminiContent(message providers.Message) (GeminiContent, error) {
	if message.Role == "tool" {
		return geminiToolContent(message)
	}

	parts, err := geminiParts(message.Content)
	if err != nil {
		return GeminiContent{}, err
	}
	for i, toolCall := range message.ToolCalls {
		part, err := geminiFunctionCallPart(toolCall)
		if err != nil {
			return GeminiContent{}, fmt.Errorf("tool call %d: %w", i, err)
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return GeminiContent{}, fmt.Errorf("empty content for role %q", message.Role)
	}

	return GeminiContent{
		Role:  geminiMessageRole(message.Role),
		Parts: parts,
	}, nil
}

func geminiToolContent(message providers.Message) (GeminiContent, error) {
	name := "tool_result"
	if message.ToolCallID != nil && *message.ToolCallID != "" {
		name = *message.ToolCallID
	}

	text, err := textFromContent(message.Content)
	if err != nil {
		return GeminiContent{}, err
	}
	return GeminiContent{
		Role: "user",
		Parts: []GeminiPart{
			{
				FunctionResponse: &GeminiFunctionResponse{
					Name:     name,
					Response: map[string]any{"content": text},
				},
			},
		},
	}, nil
}

func geminiFunctionCallPart(toolCall providers.ToolCall) (GeminiPart, error) {
	if toolCall.Type != "function" {
		return GeminiPart{}, fmt.Errorf("unsupported tool call type %q", toolCall.Type)
	}

	args, err := parseFunctionArgs(toolCall.Function.Arguments)
	if err != nil {
		return GeminiPart{}, err
	}
	return GeminiPart{
		FunctionCall: &GeminiFunctionCall{
			Name: toolCall.Function.Name,
			Args: args,
		},
	}, nil
}

func parseFunctionArgs(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err == nil {
		return args, nil
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]any{"arguments": raw}, nil
	}
	return map[string]any{"arguments": value}, nil
}

func geminiParts(content any) ([]GeminiPart, error) {
	switch value := content.(type) {
	case nil:
		return nil, nil
	case string:
		if value == "" {
			return nil, nil
		}
		return []GeminiPart{{Text: value}}, nil
	case []map[string]any:
		return geminiPartsFromBlocks(value)
	case []any:
		return geminiPartsFromAnyBlocks(value)
	default:
		return nil, fmt.Errorf("unsupported content type %T", content)
	}
}

func textFromContent(content any) (string, error) {
	parts, err := geminiParts(content)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, part := range parts {
		if part.Text == "" {
			return "", fmt.Errorf("unsupported non-text content in text-only field")
		}
		builder.WriteString(part.Text)
	}
	return builder.String(), nil
}

func geminiPartsFromAnyBlocks(blocks []any) ([]GeminiPart, error) {
	typed := make([]map[string]any, 0, len(blocks))
	for i, block := range blocks {
		value, ok := block.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content block %d: unsupported content block type %T", i, block)
		}
		typed = append(typed, value)
	}
	return geminiPartsFromBlocks(typed)
}

func geminiPartsFromBlocks(blocks []map[string]any) ([]GeminiPart, error) {
	parts := make([]GeminiPart, 0, len(blocks))
	for i, block := range blocks {
		blockType, _ := block["type"].(string)
		if blockType != "text" {
			return nil, fmt.Errorf("content block %d: unsupported content block type %q", i, blockType)
		}

		text, ok := block["text"].(string)
		if !ok {
			return nil, fmt.Errorf("content block %d: text must be a string", i)
		}
		if text != "" {
			parts = append(parts, GeminiPart{Text: text})
		}
	}
	return parts, nil
}

func geminiMessageRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return "user"
}
