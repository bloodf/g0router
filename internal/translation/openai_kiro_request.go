package translation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var dataURIRe = regexp.MustCompile(`^data:([^;]+);base64,(.+)$`)

func toolCallToText(name string, input any) string {
	var argStr string
	switch v := input.(type) {
	case string:
		argStr = v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			argStr = "{}"
		} else {
			argStr = string(b)
		}
	}
	if argStr == "" {
		argStr = "{}"
	}
	if name == "" {
		name = "unknown"
	}
	return fmt.Sprintf("[Tool call: %s(%s)]", name, argStr)
}

func toolResultToText(content any) string {
	var text string
	switch v := content.(type) {
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			switch s := item.(type) {
			case string:
				parts = append(parts, s)
			case map[string]any:
				if t, ok := s["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		text = strings.Join(parts, "\n")
	case string:
		text = v
	}
	return fmt.Sprintf("[Tool result: %s]", text)
}

func flattenToolInteractions(messages []any) []any {
	out := make([]any, 0, len(messages))
	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			out = append(out, msg)
			continue
		}
		role, _ := m["role"].(string)

		if role == "tool" {
			out = append(out, map[string]any{
				"role":    "user",
				"content": toolResultToText(m["content"]),
			})
			continue
		}

		if role == "assistant" {
			parts := make([]string, 0)
			if contentArr, ok := m["content"].([]any); ok {
				for _, c := range contentArr {
					cm, ok := c.(map[string]any)
					if !ok {
						continue
					}
					ct, _ := cm["type"].(string)
					if ct == "tool_use" {
						parts = append(parts, toolCallToText(
							coalesceString(cm["name"]),
							cm["input"],
						))
					} else if ct == "text" {
						if txt, ok := cm["text"].(string); ok {
							parts = append(parts, txt)
						}
					}
				}
			} else if contentStr, ok := m["content"].(string); ok {
				parts = append(parts, contentStr)
			}
			if tcs, ok := m["tool_calls"].([]any); ok {
				for _, tc := range tcs {
					tcm, ok := tc.(map[string]any)
					if !ok {
						continue
					}
					fn, _ := tcm["function"].(map[string]any)
					parts = append(parts, toolCallToText(
						coalesceString(fn["name"]),
						fn["arguments"],
					))
				}
			}
			filtered := make([]string, 0, len(parts))
			for _, p := range parts {
				if p != "" {
					filtered = append(filtered, p)
				}
			}
			out = append(out, map[string]any{
				"role":    "assistant",
				"content": strings.Join(filtered, "\n"),
			})
			continue
		}

		if role == "user" {
			if contentArr, ok := m["content"].([]any); ok {
				newContent := make([]any, 0, len(contentArr))
				for _, c := range contentArr {
					cm, ok := c.(map[string]any)
					if !ok {
						newContent = append(newContent, c)
						continue
					}
					if cm["type"] == "tool_result" {
						newContent = append(newContent, map[string]any{
							"type": "text",
							"text": toolResultToText(cm["content"]),
						})
					} else {
						newContent = append(newContent, c)
					}
				}
				out = append(out, map[string]any{
					"role":    m["role"],
					"content": newContent,
				})
				continue
			}
		}

		out = append(out, msg)
	}
	return out
}

func safeJSONParse(str string, fallback any) any {
	var result any
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return fallback
	}
	return result
}

func convertMessages(messages, tools []any, model string) (history []any, currentMessage map[string]any) {
	clientProvidedTools := len(tools) > 0

	if !clientProvidedTools {
		messages = flattenToolInteractions(messages)
	}

	var pendingUserContent []string
	var pendingAssistantContent []string
	var pendingToolResults []any
	var pendingImages []any
	var currentRole string
	var toolsInjectedToFirstUserMsg bool

	flushPending := func() {
		if currentRole == "user" {
			content := strings.TrimSpace(strings.Join(pendingUserContent, "\n\n"))
			if content == "" {
				content = "continue"
			}
			userMsg := map[string]any{
				"userInputMessage": map[string]any{
					"content": content,
					"modelId": "",
				},
			}
			if len(pendingImages) > 0 {
				userMsg["userInputMessage"].(map[string]any)["images"] = pendingImages
			}
			if len(pendingToolResults) > 0 {
				userMsg["userInputMessage"].(map[string]any)["userInputMessageContext"] = map[string]any{
					"toolResults": pendingToolResults,
				}
			}
			if clientProvidedTools && !toolsInjectedToFirstUserMsg {
				uim := userMsg["userInputMessage"].(map[string]any)
				ctx, ok := uim["userInputMessageContext"].(map[string]any)
				if !ok {
					ctx = map[string]any{}
					uim["userInputMessageContext"] = ctx
				}
				ctx["tools"] = mapTools(tools)
				toolsInjectedToFirstUserMsg = true
			}
			history = append(history, userMsg)
			currentMessage = userMsg
			pendingUserContent = nil
			pendingToolResults = nil
			pendingImages = nil
		} else if currentRole == "assistant" {
			content := strings.TrimSpace(strings.Join(pendingAssistantContent, "\n\n"))
			if content == "" {
				content = "..."
			}
			assistantMsg := map[string]any{
				"assistantResponseMessage": map[string]any{
					"content": content,
				},
			}
			history = append(history, assistantMsg)
			pendingAssistantContent = nil
		}
	}

	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role == "system" || role == "tool" {
			role = "user"
		}

		if role != currentRole && currentRole != "" {
			flushPending()
		}
		currentRole = role

		if role == "user" {
			var content string
			if contentStr, ok := m["content"].(string); ok {
				content = contentStr
			} else if contentArr, ok := m["content"].([]any); ok {
				textParts := make([]string, 0)
				for _, c := range contentArr {
					cm, ok := c.(map[string]any)
					if !ok {
						continue
					}
					ct, _ := cm["type"].(string)
					if ct == "text" {
						if txt, ok := cm["text"].(string); ok {
							textParts = append(textParts, txt)
						}
					} else if ct == "image_url" {
						if iu, ok := cm["image_url"].(map[string]any); ok {
							url := coalesceString(iu["url"])
							matches := dataURIRe.FindStringSubmatch(url)
							if len(matches) == 3 {
								mediaType := matches[1]
								format := mediaType
								if idx := strings.Index(mediaType, "/"); idx != -1 {
									format = mediaType[idx+1:]
								}
								pendingImages = append(pendingImages, map[string]any{
									"format": format,
									"source": map[string]any{"bytes": matches[2]},
								})
							} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
								textParts = append(textParts, fmt.Sprintf("[Image: %s]", url))
							}
						}
					} else if ct == "image" {
						if src, ok := cm["source"].(map[string]any); ok {
							if src["type"] == "base64" && src["data"] != nil {
								mediaType := coalesceString(src["media_type"])
								if mediaType == "" {
									mediaType = "image/png"
								}
								format := mediaType
								if idx := strings.Index(mediaType, "/"); idx != -1 {
									format = mediaType[idx+1:]
								}
								pendingImages = append(pendingImages, map[string]any{
									"format": format,
									"source": map[string]any{"bytes": src["data"]},
								})
							}
						}
					}
				}
				content = strings.Join(textParts, "\n")

				for _, c := range contentArr {
					cm, ok := c.(map[string]any)
					if !ok {
						continue
					}
					if cm["type"] != "tool_result" {
						continue
					}
					var text string
					switch v := cm["content"].(type) {
					case []any:
						parts := make([]string, 0, len(v))
						for _, item := range v {
							switch s := item.(type) {
							case string:
								parts = append(parts, s)
							case map[string]any:
								if t, ok := s["text"].(string); ok {
									parts = append(parts, t)
								}
							}
						}
						text = strings.Join(parts, "\n")
					case string:
						text = v
					}
					pendingToolResults = append(pendingToolResults, map[string]any{
						"toolUseId": cm["tool_use_id"],
						"status":    "success",
						"content":   []any{map[string]any{"text": text}},
					})
				}
			}

			if origRole, _ := m["role"].(string); origRole == "tool" {
				var toolContent string
				if s, ok := m["content"].(string); ok {
					toolContent = s
				}
				pendingToolResults = append(pendingToolResults, map[string]any{
					"toolUseId": m["tool_call_id"],
					"status":    "success",
					"content":   []any{map[string]any{"text": toolContent}},
				})
			} else if content != "" {
				pendingUserContent = append(pendingUserContent, content)
			}
		} else if role == "assistant" {
			var textContent string
			var toolUses []any

			if contentArr, ok := m["content"].([]any); ok {
				textBlocks := make([]string, 0)
				for _, c := range contentArr {
					cm, ok := c.(map[string]any)
					if !ok {
						continue
					}
					if cm["type"] == "text" {
						if txt, ok := cm["text"].(string); ok {
							textBlocks = append(textBlocks, txt)
						}
					} else if cm["type"] == "tool_use" {
						toolUses = append(toolUses, cm)
					}
				}
				textContent = strings.TrimSpace(strings.Join(textBlocks, "\n"))
			} else if contentStr, ok := m["content"].(string); ok {
				textContent = strings.TrimSpace(contentStr)
			}

			if tcs, ok := m["tool_calls"].([]any); ok && len(tcs) > 0 {
				toolUses = tcs
			}

			if textContent != "" {
				pendingAssistantContent = append(pendingAssistantContent, textContent)
			}

			if len(toolUses) > 0 {
				flushPending()
				if len(history) > 0 {
					lastMsg := history[len(history)-1]
					if lm, ok := lastMsg.(map[string]any); ok {
						if arm, ok := lm["assistantResponseMessage"].(map[string]any); ok {
							mapped := make([]any, 0, len(toolUses))
							for _, tu := range toolUses {
								tum, ok := tu.(map[string]any)
								if !ok {
									continue
								}
								if fn, ok := tum["function"].(map[string]any); ok {
									args := coalesceString(fn["arguments"])
									input := safeJSONParse(args, map[string]any{})
									if input == nil {
										input = map[string]any{}
									}
									id := coalesceString(tum["id"])
									if id == "" {
										id = uuid.New().String()
									}
									mapped = append(mapped, map[string]any{
										"toolUseId": id,
										"name":      coalesceString(fn["name"]),
										"input":     input,
									})
								} else {
									id := coalesceString(tum["id"])
									if id == "" {
										id = uuid.New().String()
									}
									input := tum["input"]
									if input == nil {
										input = map[string]any{}
									}
									mapped = append(mapped, map[string]any{
										"toolUseId": id,
										"name":      coalesceString(tum["name"]),
										"input":     input,
									})
								}
							}
							arm["toolUses"] = mapped
						}
					}
				}
				currentRole = ""
			}
		}
	}

	if currentRole != "" {
		flushPending()
	}

	// Pop last userInputMessage as currentMessage.
	for i := len(history) - 1; i >= 0; i-- {
		if h, ok := history[i].(map[string]any); ok {
			if _, ok := h["userInputMessage"]; ok {
				currentMessage = h
				history = append(history[:i], history[i+1:]...)
				break
			}
		}
	}

	// Grab tools from first history item BEFORE cleanup.
	var firstHistoryTools []any
	if len(history) > 0 {
		if h, ok := history[0].(map[string]any); ok {
			if uim, ok := h["userInputMessage"].(map[string]any); ok {
				if ctx, ok := uim["userInputMessageContext"].(map[string]any); ok {
					if tools, ok := ctx["tools"].([]any); ok {
						firstHistoryTools = tools
					}
				}
			}
		}
	}

	// Cleanup history.
	for _, item := range history {
		h, ok := item.(map[string]any)
		if !ok {
			continue
		}
		uim, ok := h["userInputMessage"].(map[string]any)
		if !ok {
			continue
		}
		if ctx, ok := uim["userInputMessageContext"].(map[string]any); ok {
			delete(ctx, "tools")
			if len(ctx) == 0 {
				delete(uim, "userInputMessageContext")
			}
		}
		if modelId, ok := uim["modelId"].(string); !ok || modelId == "" {
			uim["modelId"] = model
		}
	}

	// Merge consecutive user messages.
	mergedHistory := make([]any, 0)
	for _, item := range history {
		h, ok := item.(map[string]any)
		if !ok {
			mergedHistory = append(mergedHistory, item)
			continue
		}
		uim, ok := h["userInputMessage"].(map[string]any)
		if !ok {
			mergedHistory = append(mergedHistory, item)
			continue
		}
		if len(mergedHistory) == 0 {
			mergedHistory = append(mergedHistory, item)
			continue
		}
		prev, ok := mergedHistory[len(mergedHistory)-1].(map[string]any)
		if !ok {
			mergedHistory = append(mergedHistory, item)
			continue
		}
		prevUim, ok := prev["userInputMessage"].(map[string]any)
		if !ok {
			mergedHistory = append(mergedHistory, item)
			continue
		}
		prevUim["content"] = prevUim["content"].(string) + "\n\n" + uim["content"].(string)
		prevCtx, _ := prevUim["userInputMessageContext"].(map[string]any)
		curCtx, hasCurCtx := uim["userInputMessageContext"].(map[string]any)
		if hasCurCtx {
			if prevCtx == nil {
				prevUim["userInputMessageContext"] = curCtx
			} else {
				if curTR, ok := curCtx["toolResults"].([]any); ok && len(curTR) > 0 {
					prevTR, _ := prevCtx["toolResults"].([]any)
					prevCtx["toolResults"] = append(prevTR, curTR...)
				}
				if curTools, ok := curCtx["tools"].([]any); ok && len(curTools) > 0 {
					prevTools, _ := prevCtx["tools"].([]any)
					prevCtx["tools"] = append(prevTools, curTools...)
				}
			}
		}
	}

	if currentMessage == nil {
		currentMessage = map[string]any{
			"userInputMessage": map[string]any{
				"content": "",
				"modelId": model,
			},
		}
	}

	if clientProvidedTools {
		reconcileOrphanedToolResults(mergedHistory, currentMessage)
	}

	// Inject tools into currentMessage.
	if len(firstHistoryTools) > 0 {
		uim := currentMessage["userInputMessage"].(map[string]any)
		ctx, ok := uim["userInputMessageContext"].(map[string]any)
		if !ok {
			ctx = map[string]any{}
			uim["userInputMessageContext"] = ctx
		}
		if _, hasTools := ctx["tools"]; !hasTools {
			ctx["tools"] = firstHistoryTools
		}
	}

	return mergedHistory, currentMessage
}

func reconcileOrphanedToolResults(history []any, currentMessage map[string]any) {
	validIds := make(map[string]bool)
	for _, item := range history {
		h, ok := item.(map[string]any)
		if !ok {
			continue
		}
		arm, ok := h["assistantResponseMessage"].(map[string]any)
		if !ok {
			continue
		}
		uses, ok := arm["toolUses"].([]any)
		if !ok {
			continue
		}
		for _, u := range uses {
			um, ok := u.(map[string]any)
			if !ok {
				continue
			}
			if id, ok := um["toolUseId"].(string); ok && id != "" {
				validIds[id] = true
			}
		}
	}

	carriers := history
	if currentMessage != nil {
		carriers = append(carriers, currentMessage)
	}
	for _, item := range carriers {
		h, ok := item.(map[string]any)
		if !ok {
			continue
		}
		uim, ok := h["userInputMessage"].(map[string]any)
		if !ok {
			continue
		}
		ctx, ok := uim["userInputMessageContext"].(map[string]any)
		if !ok {
			continue
		}
		trs, ok := ctx["toolResults"].([]any)
		if !ok || len(trs) == 0 {
			continue
		}

		var kept []any
		var salvaged []string
		for _, tr := range trs {
			trm, ok := tr.(map[string]any)
			if !ok {
				continue
			}
			if id, ok := trm["toolUseId"].(string); ok && validIds[id] {
				kept = append(kept, tr)
			} else {
				var content any
				if c, ok := trm["content"].([]any); ok && len(c) > 0 {
					if cm, ok := c[0].(map[string]any); ok {
						content = cm["text"]
					}
				}
				salvaged = append(salvaged, toolResultToText(content))
			}
		}

		if len(salvaged) == 0 {
			continue
		}

		extra := strings.Join(salvaged, "\n")
		if existing, ok := uim["content"].(string); ok && existing != "" {
			uim["content"] = existing + "\n\n" + extra
		} else {
			uim["content"] = extra
		}
		ctx["toolResults"] = kept
		if len(kept) == 0 {
			delete(ctx, "toolResults")
		}
		if len(ctx) == 0 {
			delete(uim, "userInputMessageContext")
		}
	}
}

func mapTools(tools []any) []any {
	out := make([]any, 0, len(tools))
	for _, tool := range tools {
		tm, ok := tool.(map[string]any)
		if !ok {
			continue
		}
		var name, description string
		var schema map[string]any

		if fn, ok := tm["function"].(map[string]any); ok {
			name = coalesceString(fn["name"])
			description = coalesceString(fn["description"])
			schema, _ = fn["parameters"].(map[string]any)
		} else {
			name = coalesceString(tm["name"])
			description = coalesceString(tm["description"])
			schema, _ = tm["parameters"].(map[string]any)
			if schema == nil {
				schema, _ = tm["input_schema"].(map[string]any)
			}
		}

		if description == "" {
			description = fmt.Sprintf("Tool: %s", name)
		}

		normalized := map[string]any{"type": "object", "properties": map[string]any{}, "required": []any{}}
		if len(schema) > 0 {
			normalized = make(map[string]any, len(schema)+1)
			for k, v := range schema {
				normalized[k] = v
			}
			if _, ok := normalized["required"]; !ok {
				normalized["required"] = []any{}
			}
		}

		out = append(out, map[string]any{
			"toolSpecification": map[string]any{
				"name":        name,
				"description": description,
				"inputSchema": map[string]any{"json": normalized},
			},
		})
	}
	return out
}

func coalesceString(v any) string {
	s, _ := v.(string)
	return s
}

// buildKiroPayload builds the Kiro-format payload from an OpenAI-format request
// body. Port of open-sse/translator/request/openai-to-kiro.js:511-581.
//
// NOTE: the ref's non-enumerable _kiroUpstreamModel tag (:575-578) is NOT
// emitted because non-enumerable properties do not serialize in JS; adding it
// would change wire bytes vs the ref. The Go executor (Wave 2) re-derives the
// upstream id via the exported ResolveKiroModel(model).
func buildKiroPayload(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	var messages, tools []any
	if m, ok := body["messages"].([]any); ok {
		messages = m
	}
	if t, ok := body["tools"].([]any); ok {
		tools = t
	}
	maxTokens := 32000
	var temperature, topP any
	temperature = body["temperature"]
	topP = body["top_p"]

	upstreamModel, agentic, modelImpliesThinking := ResolveKiroModel(model)
	thinkingEnabled := modelImpliesThinking || isThinkingEnabled(body, nil, model)

	history, currentMessage := convertMessages(messages, tools, upstreamModel)

	var profileArn string
	if credentials != nil {
		if psd, ok := credentials["providerSpecificData"].(map[string]any); ok {
			profileArn = coalesceString(psd["profileArn"])
		}
	}

	finalContent := ""
	if currentMessage != nil {
		if uim, ok := currentMessage["userInputMessage"].(map[string]any); ok {
			finalContent = coalesceString(uim["content"])
		}
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	prefixParts := make([]string, 0)
	if thinkingEnabled {
		prefixParts = append(prefixParts, buildThinkingSystemPrefix(kiroThinkingBudgetDefault))
	}
	prefixParts = append(prefixParts, fmt.Sprintf("[Context: Current time is %s]", timestamp))
	if agentic {
		prefixParts = append(prefixParts, kiroAgenticSystemPrompt)
	}
	finalContent = strings.Join(prefixParts, "\n\n") + "\n\n" + finalContent

	uim := map[string]any{
		"content": finalContent,
		"modelId": upstreamModel,
		"origin":  "AI_EDITOR",
	}

	if currentMessage != nil {
		if curUim, ok := currentMessage["userInputMessage"].(map[string]any); ok {
			if imgs, ok := curUim["images"].([]any); ok && len(imgs) > 0 {
				uim["images"] = imgs
			}
			if ctx, ok := curUim["userInputMessageContext"].(map[string]any); ok && len(ctx) > 0 {
				uim["userInputMessageContext"] = ctx
			}
		}
	}

	payload := map[string]any{
		"conversationState": map[string]any{
			"chatTriggerType": "MANUAL",
			"conversationId":  uuid.New().String(),
			"currentMessage": map[string]any{
				"userInputMessage": uim,
			},
			"history": history,
		},
	}

	if profileArn != "" {
		payload["profileArn"] = profileArn
	}

	ic := map[string]any{}
	if maxTokens > 0 {
		ic["maxTokens"] = maxTokens
	}
	if temperature != nil {
		ic["temperature"] = temperature
	}
	if topP != nil {
		ic["topP"] = topP
	}
	if len(ic) > 0 {
		payload["inferenceConfig"] = ic
	}

	return payload, nil
}
