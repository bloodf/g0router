package translation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const claudeSystemPrompt = "You are Claude Code, Anthropic's official CLI for Claude."

// openaiToClaudeRequest converts an OpenAI-format request body to a Claude-
// shaped request body.
func openaiToClaudeRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	toolNameMap := make(map[string]string)
	result := map[string]any{
		"model":      model,
		"max_tokens": AdjustMaxTokens(body),
		"stream":     stream,
	}

	if temp, ok := body["temperature"]; ok && temp != nil {
		result["temperature"] = temp
	}

	result["messages"] = []any{}
	systemParts := []string{}

	if rawMsgs, ok := body["messages"].([]any); ok {
		for _, msg := range rawMsgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			if role, _ := m["role"].(string); role == "system" {
				content := m["content"]
				switch v := content.(type) {
				case string:
					systemParts = append(systemParts, v)
				case []any:
					systemParts = append(systemParts, extractTextContent(v))
				}
			}
		}

		nonSystem := make([]map[string]any, 0, len(rawMsgs))
		for _, msg := range rawMsgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			if role, _ := m["role"].(string); role != "system" {
				nonSystem = append(nonSystem, m)
			}
		}

		var currentRole string
		var currentParts []any

		flushCurrentMessage := func() {
			if currentRole != "" && len(currentParts) > 0 {
				result["messages"] = append(result["messages"].([]any), map[string]any{
					"role":    currentRole,
					"content": currentParts,
				})
				currentParts = nil
			}
		}

		for _, msg := range nonSystem {
			// Missing/non-string role maps to assistant, matching 9router
			// where an undefined role falls into the else branch.
			role, _ := msg["role"].(string)
			newRole := role
			if role == "user" || role == "tool" {
				newRole = "user"
			} else {
				newRole = "assistant"
			}

			blocks := getContentBlocksFromMessage(msg, toolNameMap)
			hasToolUse := false
			hasToolResult := false
			for _, b := range blocks {
				bm, ok := b.(map[string]any)
				if !ok {
					continue
				}
				bt, _ := bm["type"].(string)
				if bt == "tool_use" {
					hasToolUse = true
				}
				if bt == "tool_result" {
					hasToolResult = true
				}
			}

			// Separate tool_result from other content.
			if hasToolResult {
				toolResultBlocks := make([]any, 0)
				otherBlocks := make([]any, 0)
				for _, b := range blocks {
					bm, ok := b.(map[string]any)
					if !ok {
						otherBlocks = append(otherBlocks, b)
						continue
					}
					if bt, _ := bm["type"].(string); bt == "tool_result" {
						toolResultBlocks = append(toolResultBlocks, b)
					} else {
						otherBlocks = append(otherBlocks, b)
					}
				}

				flushCurrentMessage()

				if len(toolResultBlocks) > 0 {
					result["messages"] = append(result["messages"].([]any), map[string]any{
						"role":    "user",
						"content": toolResultBlocks,
					})
				}

				if len(otherBlocks) > 0 {
					currentRole = newRole
					currentParts = append(currentParts, otherBlocks...)
				}
				continue
			}

			if currentRole != newRole {
				flushCurrentMessage()
				currentRole = newRole
			}

			currentParts = append(currentParts, blocks...)

			if hasToolUse {
				flushCurrentMessage()
			}
		}

		flushCurrentMessage()

		// Add cache_control to last assistant message's last cacheable block.
		msgs := result["messages"].([]any)
		for i := len(msgs) - 1; i >= 0; i-- {
			m, ok := msgs[i].(map[string]any)
			if !ok {
				continue
			}
			if role, _ := m["role"].(string); role != "assistant" {
				continue
			}
			content, ok := m["content"].([]any)
			if !ok || len(content) == 0 {
				continue
			}
			validBlockTypes := map[string]bool{"text": true, "tool_use": true, "tool_result": true, "image": true}
			for j := len(content) - 1; j >= 0; j-- {
				block, ok := content[j].(map[string]any)
				if !ok {
					continue
				}
				bt, _ := block["type"].(string)
				if validBlockTypes[bt] {
					block["cache_control"] = map[string]any{"type": "ephemeral"}
					break
				}
			}
			break
		}
	}

	if rf, ok := body["response_format"]; ok && rf != nil {
		responseFormat, ok := rf.(map[string]any)
		if ok {
			rfType, _ := responseFormat["type"].(string)
			if rfType == "json_schema" {
				if jsonSchema, ok := responseFormat["json_schema"].(map[string]any); ok {
					if schema, ok := jsonSchema["schema"]; ok && schema != nil {
						schemaJSON, err := json.MarshalIndent(schema, "", "  ")
						if err == nil {
							systemParts = append(systemParts, fmt.Sprintf(`You must respond with valid JSON that strictly follows this JSON schema:
`+"```json\n%s\n```\n"+`Respond ONLY with the JSON object, no other text.`, string(schemaJSON)))
						}
					}
				}
			} else if rfType == "json_object" {
				systemParts = append(systemParts, "You must respond with valid JSON. Respond ONLY with a JSON object, no other text.")
			}
		}
	}

	claudeCodePrompt := map[string]any{"type": "text", "text": claudeSystemPrompt}
	if len(systemParts) > 0 {
		systemText := strings.Join(systemParts, "\n")
		result["system"] = []any{
			claudeCodePrompt,
			map[string]any{"type": "text", "text": systemText, "cache_control": map[string]any{"type": "ephemeral", "ttl": "1h"}},
		}
	} else {
		result["system"] = []any{claudeCodePrompt}
	}

	if rawTools, ok := body["tools"].([]any); ok {
		result["tools"] = []any{}
		for _, tool := range rawTools {
			toolMap, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			toolType, _ := toolMap["type"].(string)
			if toolType != "" && toolType != "function" {
				result["tools"] = append(result["tools"].([]any), tool)
				continue
			}

			var toolData map[string]any
			if toolType == "function" && toolMap["function"] != nil {
				toolData, _ = toolMap["function"].(map[string]any)
			} else {
				toolData = toolMap
			}
			if toolData == nil {
				toolData = map[string]any{}
			}

			originalName, _ := toolData["name"].(string)
			toolName := originalName

			toolNameMap[toolName] = originalName

			desc := ""
			if d, ok := toolData["description"].(string); ok {
				desc = d
			}

			inputSchema := map[string]any{"type": "object", "properties": map[string]any{}, "required": []any{}}
			if params, ok := toolData["parameters"].(map[string]any); ok {
				inputSchema = params
			} else if is, ok := toolData["input_schema"].(map[string]any); ok {
				inputSchema = is
			}

			result["tools"] = append(result["tools"].([]any), map[string]any{
				"name":         toolName,
				"description":  desc,
				"input_schema": inputSchema,
			})
		}

		tools := result["tools"].([]any)
		if len(tools) > 0 {
			lastTool, _ := tools[len(tools)-1].(map[string]any)
			if lastTool != nil {
				lastTool["cache_control"] = map[string]any{"type": "ephemeral", "ttl": "1h"}
			}
		}
	}

	if tc, ok := body["tool_choice"]; ok && tc != nil {
		result["tool_choice"] = convertOpenAIToolChoice(tc)
	}

	if rawThinking, ok := body["thinking"]; ok && rawThinking != nil {
		if thinking, ok := rawThinking.(map[string]any); ok {
			thinkingResult := map[string]any{
				"type": "enabled",
			}
			if t, ok := thinking["type"].(string); ok && t != "" {
				thinkingResult["type"] = t
			}
			if budget, ok := thinking["budget_tokens"]; ok && budget != nil {
				thinkingResult["budget_tokens"] = budget
			}
			if maxTokens, ok := thinking["max_tokens"]; ok && maxTokens != nil {
				thinkingResult["max_tokens"] = maxTokens
			}
			result["thinking"] = thinkingResult
		}
	}

	if rawEffort, ok := body["reasoning_effort"]; ok && rawEffort != nil {
		if effort, ok := rawEffort.(string); ok && effort != "" {
			if _, hasThinking := result["thinking"]; !hasThinking {
				effortToBudget := map[string]int{
					"none":   0,
					"low":    4096,
					"medium": 8192,
					"high":   16384,
					"xhigh":  32768,
				}
				budget := effortToBudget[strings.ToLower(effort)]
				if budget == 0 {
					// none → no thinking block
				} else if budget > 0 {
					result["thinking"] = map[string]any{"type": "enabled", "budget_tokens": budget}
				}
			}
		}
	}

	// PAR-PR-1264: drop temperature only when outgoing thinking is enabled
	// (Anthropic 400s on temperature + extended thinking).
	if thinking, ok := result["thinking"].(map[string]any); ok {
		if t, _ := thinking["type"].(string); t == "enabled" {
			delete(result, "temperature")
		}
	}

	if len(toolNameMap) > 0 {
		result["_toolNameMap"] = toolNameMap
	}

	return result, nil
}

// getContentBlocksFromMessage mirrors the frozen ref's signature
// (openai-to-claude.js:210: `getContentBlocksFromMessage(msg, toolNameMap)`);
// the ref also never reads the map here because CLAUDE_OAUTH_TOOL_PREFIX is
// the empty string, so tool names pass through uncloaked.
func getContentBlocksFromMessage(msg map[string]any, toolNameMap map[string]string) []any {
	blocks := []any{}
	role, _ := msg["role"].(string)

	if role == "tool" {
		toolCallID := ""
		if id, ok := msg["tool_call_id"].(string); ok {
			toolCallID = id
		}
		content := msg["content"]
		blocks = append(blocks, map[string]any{
			"type":        "tool_result",
			"tool_use_id": toolCallID,
			"content":     content,
		})
	} else if role == "user" {
		if content, ok := msg["content"].(string); ok {
			if content != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": content})
			}
		} else if parts, ok := msg["content"].([]any); ok {
			for _, part := range parts {
				partMap, ok := part.(map[string]any)
				if !ok {
					continue
				}
				pt, _ := partMap["type"].(string)
				switch pt {
				case "text":
					if text, ok := partMap["text"].(string); ok && text != "" {
						blocks = append(blocks, map[string]any{"type": "text", "text": text})
					}
				case "tool_result":
					tr := map[string]any{
						"type":        "tool_result",
						"tool_use_id": partMap["tool_use_id"],
						"content":     partMap["content"],
					}
					if isError, ok := partMap["is_error"].(bool); ok {
						tr["is_error"] = isError
					}
					blocks = append(blocks, tr)
				case "image_url":
					if imageURL, ok := partMap["image_url"].(map[string]any); ok {
						url, _ := imageURL["url"].(string)
						var match []string
						if dataURI, err := regexp.Compile(`^data:([^;]+);base64,(.+)$`); err == nil {
							match = dataURI.FindStringSubmatch(url)
						}
						if len(match) == 3 {
							blocks = append(blocks, map[string]any{
								"type": "image",
								"source": map[string]any{
									"type":       "base64",
									"media_type": match[1],
									"data":       match[2],
								},
							})
						} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
							blocks = append(blocks, map[string]any{
								"type":   "image",
								"source": map[string]any{"type": "url", "url": url},
							})
						}
					}
				case "image":
					if source, ok := partMap["source"]; ok && source != nil {
						blocks = append(blocks, map[string]any{"type": "image", "source": source})
					}
				}
			}
		}
	} else if role == "assistant" {
		if content, ok := msg["content"].(string); ok && content != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": content})
		} else if parts, ok := msg["content"].([]any); ok {
			for _, part := range parts {
				partMap, ok := part.(map[string]any)
				if !ok {
					continue
				}
				pt, _ := partMap["type"].(string)
				switch pt {
				case "text":
					if text, ok := partMap["text"].(string); ok && text != "" {
						blocks = append(blocks, map[string]any{"type": "text", "text": text})
					}
				case "tool_use":
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    partMap["id"],
						"name":  partMap["name"],
						"input": partMap["input"],
					})
				case "thinking":
					// Strip cache_control (not allowed on thinking blocks)
					thinkingBlock := map[string]any{
						"type":     "thinking",
						"thinking": partMap["thinking"],
					}
					blocks = append(blocks, thinkingBlock)
				}
			}
		}

		if rawToolCalls, ok := msg["tool_calls"].([]any); ok {
			for _, tc := range rawToolCalls {
				tcMap, ok := tc.(map[string]any)
				if !ok {
					continue
				}
				if tcType, _ := tcMap["type"].(string); tcType != "function" {
					continue
				}
				var fn map[string]any
				if f, ok := tcMap["function"].(map[string]any); ok {
					fn = f
				}
				toolName := ""
				if name, ok := fn["name"].(string); ok {
					toolName = name
				}
				arguments := ""
				if args, ok := fn["arguments"].(string); ok {
					arguments = args
				}
				blocks = append(blocks, map[string]any{
					"type":  "tool_use",
					"id":    tcMap["id"],
					"name":  toolName,
					"input": tryParseJSON(arguments),
				})
			}
		}
	}

	return blocks
}

func convertOpenAIToolChoice(choice any) any {
	if choice == nil {
		return map[string]any{"type": "auto"}
	}

	if s, ok := choice.(string); ok {
		if s == "required" {
			return map[string]any{"type": "any"}
		}
		return map[string]any{"type": "auto"}
	}

	if m, ok := choice.(map[string]any); ok {
		if fn, ok := m["function"].(map[string]any); ok && fn != nil {
			if name, ok := fn["name"].(string); ok && name != "" {
				return map[string]any{"type": "tool", "name": name}
			}
		}
		if t, ok := m["type"].(string); ok {
			switch t {
			case "auto", "any", "tool", "none":
				return m
			}
		}
	}

	return map[string]any{"type": "auto"}
}

func extractTextContent(content []any) string {
	parts := make([]string, 0)
	for _, item := range content {
		if m, ok := item.(map[string]any); ok {
			if text, ok := m["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func tryParseJSON(str string) any {
	var result any
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return str
	}
	return result
}
