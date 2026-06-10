package gemini

import (
	"encoding/json"

	"github.com/bloodf/g0router/internal/schemas"
)

// GenerateContentRequest is the Gemini chat request body.
type GenerateContentRequest struct {
	Model             string              `json:"-"`
	Contents          []Content           `json:"contents"`
	SystemInstruction *Content            `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig   `json:"generationConfig,omitempty"`
	Tools             []Tool              `json:"tools,omitempty"`
	ToolConfig        *ToolConfig         `json:"toolConfig,omitempty"`
}

// Content represents a message in Gemini format.
type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

// Part is a single piece of content (text, image, function call, etc.).
type Part struct {
	Text             string            `json:"text,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

// FunctionCall represents a tool invocation in Gemini.
type FunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// FunctionResponse represents a tool result in Gemini.
type FunctionResponse struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// GenerationConfig holds generation parameters.
type GenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// Tool is the Gemini tool definition wrapper.
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// FunctionDeclaration describes a callable function.
type FunctionDeclaration struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolConfig controls tool usage.
type ToolConfig struct {
	FunctionCallingConfig FunctionCallingConfig `json:"functionCallingConfig"`
}

// FunctionCallingConfig specifies the tool calling mode.
type FunctionCallingConfig struct {
	Mode string `json:"mode"`
}

// GenerateContentResponse is the Gemini chat response.
type GenerateContentResponse struct {
	Candidates    []Candidate     `json:"candidates"`
	UsageMetadata *UsageMetadata  `json:"usageMetadata,omitempty"`
}

// Candidate is a single response candidate.
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason,omitempty"`
}

// UsageMetadata tracks token consumption.
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// EmbedContentRequest is the Gemini embedding request.
type EmbedContentRequest struct {
	Model   string  `json:"-"`
	Content Content `json:"content"`
}

// EmbedContentResponse is the Gemini embedding response.
type EmbedContentResponse struct {
	Embedding Embedding `json:"embedding"`
}

// Embedding holds the vector values.
type Embedding struct {
	Values []float32 `json:"values"`
}

// ConvertChatRequest transforms an OpenAI ChatRequest into a Gemini GenerateContentRequest.
func ConvertChatRequest(req *schemas.ChatRequest) *GenerateContentRequest {
	gemReq := &GenerateContentRequest{
		Model: req.Model,
	}

	if req.Temperature != nil {
		gemReq.GenerationConfig = &GenerationConfig{Temperature: req.Temperature}
	}
	if req.MaxTokens != nil {
		if gemReq.GenerationConfig == nil {
			gemReq.GenerationConfig = &GenerationConfig{}
		}
		gemReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
	}
	if req.TopP != nil {
		if gemReq.GenerationConfig == nil {
			gemReq.GenerationConfig = &GenerationConfig{}
		}
		gemReq.GenerationConfig.TopP = req.TopP
	}
	if len(req.Stop) > 0 {
		if gemReq.GenerationConfig == nil {
			gemReq.GenerationConfig = &GenerationConfig{}
		}
		gemReq.GenerationConfig.StopSequences = req.Stop
	}

	var contents []schemas.Message
	var systemParts []string
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, m.Content)
			continue
		}
		contents = append(contents, m)
	}
	if len(systemParts) > 0 {
		gemReq.SystemInstruction = &Content{
			Parts: []Part{{Text: joinSystemMessages(systemParts)}},
		}
	}

	gemReq.Contents = convertMessages(contents)

	if len(req.Tools) > 0 {
		gemReq.Tools = convertTools(req.Tools)
	}
	if req.ToolChoice != nil {
		gemReq.ToolConfig = convertToolChoice(req.ToolChoice)
	}

	return gemReq
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

func convertMessages(msgs []schemas.Message) []Content {
	var result []Content
	for _, m := range msgs {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "tool" {
			role = "user"
		}

		content := Content{Role: role}
		if m.Role == "tool" {
			var response map[string]any
			if err := json.Unmarshal([]byte(m.Content), &response); err != nil {
				response = map[string]any{"result": m.Content}
			}
			fr := &FunctionResponse{
				Name:     "",
				Response: response,
			}
			if m.Name != nil {
				fr.Name = *m.Name
			}
			if m.ToolCallID != nil {
				fr.ID = *m.ToolCallID
			}
			content.Parts = []Part{{FunctionResponse: fr}}
		} else if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				content.Parts = append(content.Parts, Part{
					FunctionCall: &FunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}
			if m.Content != "" {
				content.Parts = append([]Part{{Text: m.Content}}, content.Parts...)
			}
		} else {
			content.Parts = []Part{{Text: m.Content}}
		}
		result = append(result, content)
	}
	return result
}

func convertTools(tools []schemas.Tool) []Tool {
	var decls []FunctionDeclaration
	for _, t := range tools {
		decls = append(decls, FunctionDeclaration{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  t.Function.Parameters,
		})
	}
	return []Tool{{FunctionDeclarations: decls}}
}

func convertToolChoice(tc *schemas.ToolChoice) *ToolConfig {
	mode := "AUTO"
	switch tc.Type {
	case "none":
		mode = "NONE"
	case "function", "required":
		mode = "ANY"
	}
	return &ToolConfig{
		FunctionCallingConfig: FunctionCallingConfig{Mode: mode},
	}
}

// ConvertChatResponse transforms a Gemini GenerateContentResponse into an OpenAI ChatResponse.
func ConvertChatResponse(gemResp *GenerateContentResponse, model string) *schemas.ChatResponse {
	resp := &schemas.ChatResponse{
		Object:  "chat.completion",
		Model:   model,
		Choices: []schemas.Choice{},
	}

	if len(gemResp.Candidates) == 0 {
		resp.Choices = append(resp.Choices, schemas.Choice{
			Index:        0,
			Message:      &schemas.Message{Role: "assistant"},
			FinishReason: "stop",
		})
		return resp
	}

	for i, cand := range gemResp.Candidates {
		choice := schemas.Choice{
			Index:        i,
			Message:      &schemas.Message{Role: "assistant"},
			FinishReason: mapFinishReason(cand.FinishReason),
		}

		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				choice.Message.Content += part.Text
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				choice.Message.ToolCalls = append(choice.Message.ToolCalls, schemas.ToolCall{
					ID:   "call_" + part.FunctionCall.Name,
					Type: "function",
					Function: schemas.FunctionCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}

		resp.Choices = append(resp.Choices, choice)
	}

	if gemResp.UsageMetadata != nil {
		resp.Usage = &schemas.Usage{
			PromptTokens:     gemResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: gemResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gemResp.UsageMetadata.TotalTokenCount,
		}
	}

	return resp
}

func mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	default:
		return reason
	}
}

// ConvertEmbeddingRequest transforms an OpenAI EmbeddingRequest into a Gemini EmbedContentRequest.
func ConvertEmbeddingRequest(req *schemas.EmbeddingRequest) *EmbedContentRequest {
	text := ""
	if s, ok := req.Input.(string); ok {
		text = s
	} else if arr, ok := req.Input.([]any); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok {
			text = s
		}
	}
	return &EmbedContentRequest{
		Model: req.Model,
		Content: Content{
			Parts: []Part{{Text: text}},
		},
	}
}

// ConvertEmbeddingResponse transforms a Gemini EmbedContentResponse into an OpenAI EmbeddingResponse.
func ConvertEmbeddingResponse(gemResp *EmbedContentResponse, model string) *schemas.EmbeddingResponse {
	embedding := make([]float64, len(gemResp.Embedding.Values))
	for i, v := range gemResp.Embedding.Values {
		embedding[i] = float64(v)
	}

	return &schemas.EmbeddingResponse{
		Object: "list",
		Data: []schemas.Embedding{{
			Object:    "embedding",
			Index:     0,
			Embedding: embedding,
		}},
		Model: model,
		Usage: &schemas.Usage{TotalTokens: 0},
	}
}

// ConvertStreamChunk transforms a Gemini stream response into an OpenAI StreamChunk.
func ConvertStreamChunk(gemResp *GenerateContentResponse, model string) *schemas.StreamChunk {
	chunk := &schemas.StreamChunk{
		Object:  "chat.completion.chunk",
		Model:   model,
		Choices: []schemas.StreamChoice{},
	}

	if len(gemResp.Candidates) == 0 {
		return chunk
	}

	for i, cand := range gemResp.Candidates {
		choice := schemas.StreamChoice{
			Index: i,
			Delta: schemas.Message{Role: "assistant"},
		}
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				choice.Delta.Content += part.Text
			}
		}
		if cand.FinishReason != "" {
			reason := mapFinishReason(cand.FinishReason)
			choice.FinishReason = &reason
		}
		chunk.Choices = append(chunk.Choices, choice)
	}

	return chunk
}
