package translation

import (
	"fmt"
	"strings"
	"time"
)

const defaultBypassText = "CLI Command Execution: Clear Terminal"

var skipPatterns = []string{
	"Please write a 5-10 word title for the following conversation:",
}

// detectBypassSourceFormat detects request format from body structure.
// Ported from 9router provider.js:49.
func detectBypassSourceFormat(body map[string]any) string {
	if input, ok := body["input"]; ok {
		switch input.(type) {
		case []any, string:
			if _, hasMessages := body["messages"]; !hasMessages {
				return "openai-responses"
			}
		}
	}

	if req, ok := body["request"].(map[string]any); ok {
		if _, hasContents := req["contents"]; hasContents {
			if ua, ok := body["userAgent"].(string); ok && ua == "antigravity" {
				return "antigravity"
			}
		}
	}

	if contents, ok := body["contents"].([]any); ok && len(contents) > 0 {
		return "gemini"
	}

	openaiIndicators := []string{
		"stream_options", "response_format", "logprobs", "top_logprobs",
		"n", "presence_penalty", "frequency_penalty", "logit_bias", "user",
	}
	for _, key := range openaiIndicators {
		if _, ok := body[key]; ok {
			return "openai"
		}
	}

	if msgs, ok := body["messages"].([]any); ok && len(msgs) > 0 {
		firstMsg, ok := msgs[0].(map[string]any)
		if ok {
			if content, ok := firstMsg["content"].([]any); ok && len(content) > 0 {
				firstContent, ok := content[0].(map[string]any)
				if ok {
					if firstContent["type"] == "text" {
						if model, ok := body["model"].(string); ok && !strings.Contains(model, "/") {
							if _, hasSystem := body["system"]; hasSystem {
								return "claude"
							}
							if _, hasVersion := body["anthropic_version"]; hasVersion {
								return "claude"
							}
							for _, c := range content {
								block, ok := c.(map[string]any)
								if !ok {
									continue
								}
								if block["type"] == "image" {
									if src, ok := block["source"].(map[string]any); ok {
										if src["type"] == "base64" {
											return "claude"
										}
									}
								}
								if block["type"] == "image_url" {
									if url, ok := block["image_url"].(map[string]any); ok {
										if url["url"] != nil {
											return "openai"
										}
									}
								}
								if block["type"] == "tool_use" || block["type"] == "tool_result" {
									return "claude"
								}
							}
						}
					}
				}
			}
			if _, hasSystem := body["system"]; hasSystem {
				return "claude"
			}
			if _, hasVersion := body["anthropic_version"]; hasVersion {
				return "claude"
			}
		}
	}

	return "openai"
}

func getTextContent(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	arr, ok := content.([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, item := range arr {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if block["type"] == "text" {
			if text, ok := block["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, " ")
}

// HandleBypassRequest checks for bypass patterns and returns a fake response
// without calling the provider. Only works for Claude CLI requests.
// The response is shaped in the detected source format (OpenAI, Claude, etc.).
func HandleBypassRequest(body map[string]any, model string, userAgent string, ccFilterNaming bool, reg *Registry) ([]map[string]any, bool) {
	if !strings.Contains(userAgent, "claude-cli") {
		return nil, false
	}

	messages, ok := body["messages"].([]any)
	if !ok || len(messages) == 0 {
		return nil, false
	}

	shouldBypass := false
	namingBypass := false

	// Pattern 1: Title extraction (assistant message = "{")
	lastMsg, ok := messages[len(messages)-1].(map[string]any)
	if ok && lastMsg["role"] == "assistant" {
		if content, ok := lastMsg["content"].([]any); ok && len(content) > 0 {
			if firstBlock, ok := content[0].(map[string]any); ok {
				if firstBlock["text"] == "{" {
					shouldBypass = true
				}
			}
		}
	}

	// Pattern 2: Warmup
	if !shouldBypass {
		firstMsg, ok := messages[0].(map[string]any)
		if ok {
			if getTextContent(firstMsg["content"]) == "Warmup" {
				shouldBypass = true
			}
		}
	}

	// Pattern 3: Count
	if !shouldBypass && len(messages) == 1 {
		firstMsg, ok := messages[0].(map[string]any)
		if ok && firstMsg["role"] == "user" {
			if getTextContent(firstMsg["content"]) == "count" {
				shouldBypass = true
			}
		}
	}

	// Pattern 4: Skip patterns
	if !shouldBypass {
		var userTexts []string
		for _, m := range messages {
			msg, ok := m.(map[string]any)
			if !ok || msg["role"] != "user" {
				continue
			}
			userTexts = append(userTexts, getTextContent(msg["content"]))
		}
		userText := strings.Join(userTexts, " ")
		for _, p := range skipPatterns {
			if strings.Contains(userText, p) {
				shouldBypass = true
				break
			}
		}
	}

	// Pattern 5: CC naming request
	if !shouldBypass && ccFilterNaming {
		var systemText string
		for _, m := range messages {
			msg, ok := m.(map[string]any)
			if !ok || msg["role"] != "system" {
				continue
			}
			systemText = getTextContent(msg["content"])
			if systemText != "" {
				break
			}
		}
		if systemText == "" {
			if sys, ok := body["system"].([]any); ok {
				var parts []string
				for _, s := range sys {
					block, ok := s.(map[string]any)
					if !ok {
						continue
					}
					if block["type"] == "text" {
						if text, ok := block["text"].(string); ok {
							parts = append(parts, text)
						}
					}
				}
				systemText = strings.Join(parts, " ")
			} else if sysStr, ok := body["system"].(string); ok {
				systemText = sysStr
			}
		}
		if strings.Contains(systemText, "isNewTopic") {
			shouldBypass = true
			namingBypass = true
		}
	}

	if !shouldBypass {
		return nil, false
	}

	stream := true
	if s, ok := body["stream"].(bool); ok {
		stream = s
	}

	var text string
	if namingBypass {
		for _, m := range messages {
			msg, ok := m.(map[string]any)
			if !ok || msg["role"] != "user" {
				continue
			}
			userText := strings.TrimSpace(getTextContent(msg["content"]))
			words := strings.Fields(userText)
			if len(words) > 3 {
				words = words[:3]
			}
			title := strings.Join(words, " ")
			text = fmt.Sprintf(`{"isNewTopic":true,"title":"%s"}`, title)
			break
		}
	}
	if text == "" {
		text = defaultBypassText
	}

	sourceFormat := Format(detectBypassSourceFormat(body))
	if sourceFormat == "" {
		sourceFormat = FormatOpenAI
	}

	if stream {
		return createStreamingBypassResponse(reg, sourceFormat, model, text), true
	}
	return createNonStreamingBypassResponse(reg, sourceFormat, model, text), true
}

func createNonStreamingBypassResponse(reg *Registry, sourceFormat Format, model string, text string) []map[string]any {
	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
	created := time.Now().Unix()
	openaiChunk := map[string]any{
		"id":      id,
		"object":  "chat.completion",
		"created": created,
		"model":   model,
		"choices": []any{
			map[string]any{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": text,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     1,
			"completion_tokens": 1,
			"total_tokens":      2,
		},
	}
	if reg == nil || sourceFormat == FormatOpenAI {
		return []map[string]any{openaiChunk}
	}
	state := NewStreamState()
	translated, err := reg.TranslateResponse(FormatOpenAI, sourceFormat, openaiChunk, state)
	if err != nil {
		return []map[string]any{openaiChunk}
	}
	return translated
}

func createStreamingBypassResponse(reg *Registry, sourceFormat Format, model string, text string) []map[string]any {
	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli())
	created := time.Now().Unix()
	usage := map[string]any{
		"prompt_tokens":     1,
		"completion_tokens": 1,
		"total_tokens":      2,
	}
	openaiChunks := []map[string]any{
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []any{
				map[string]any{
					"index": 0,
					"delta": map[string]any{
						"role":    "assistant",
						"content": text,
					},
					"finish_reason": nil,
				},
			},
		},
		{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []any{
				map[string]any{
					"index": 0,
					"delta": map[string]any{},
					"finish_reason": "stop",
				},
			},
			"usage": usage,
		},
	}
	if reg == nil || sourceFormat == FormatOpenAI {
		return openaiChunks
	}
	state := NewStreamState()
	var results []map[string]any
	for _, chunk := range openaiChunks {
		translated, err := reg.TranslateResponse(FormatOpenAI, sourceFormat, chunk, state)
		if err != nil {
			results = append(results, chunk)
			continue
		}
		results = append(results, translated...)
	}
	return results
}
