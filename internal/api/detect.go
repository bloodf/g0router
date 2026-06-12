package api

import "strings"

// DetectFormat returns the request format for body without calling the translation package.
// Possible values: "openai", "claude", "gemini", "antigravity", "openai-responses".
// Inlined from translation.detectBypassSourceFormat (PAR-ROUTE-033) to avoid
// modifying internal/translation/* within the w4-f ownership scope.
func DetectFormat(body map[string]any) string {
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

	for _, key := range []string{
		"stream_options", "response_format", "logprobs", "top_logprobs",
		"n", "presence_penalty", "frequency_penalty", "logit_bias", "user",
	} {
		if _, ok := body[key]; ok {
			return "openai"
		}
	}

	if msgs, ok := body["messages"].([]any); ok && len(msgs) > 0 {
		firstMsg, ok := msgs[0].(map[string]any)
		if ok {
			if content, ok := firstMsg["content"].([]any); ok && len(content) > 0 {
				firstContent, ok := content[0].(map[string]any)
				if ok && firstContent["type"] == "text" {
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
								if src, ok := block["source"].(map[string]any); ok && src["type"] == "base64" {
									return "claude"
								}
							}
							if block["type"] == "image_url" {
								if url, ok := block["image_url"].(map[string]any); ok && url["url"] != nil {
									return "openai"
								}
							}
							if block["type"] == "tool_use" || block["type"] == "tool_result" {
								return "claude"
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
