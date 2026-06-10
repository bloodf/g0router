package translation

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// openaiToCommandCodeRequest converts an OpenAI-format request body to a
// CommandCode-shaped request body. Port of
// open-sse/translator/request/openai-to-commandcode.js.
func openaiToCommandCodeRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	messages, system := convertCommandCodeMessages(body)
	params := map[string]any{
		"model":       model,
		"messages":    messages,
		"stream":      stream,
		"max_tokens":  64000,
		"temperature": 0.3,
	}

	if mt, ok := body["max_tokens"]; ok && mt != nil {
		params["max_tokens"] = mt
	} else if mot, ok := body["max_output_tokens"]; ok && mot != nil {
		params["max_tokens"] = mot
	}

	if temp, ok := body["temperature"]; ok && temp != nil {
		params["temperature"] = temp
	}

	if system != "" {
		params["system"] = system
	}

	tools := convertCommandCodeTools(body)
	if tools != nil {
		params["tools"] = tools
	}

	if topP, ok := body["top_p"]; ok && topP != nil {
		params["top_p"] = topP
	}

	threadID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate threadId: %w", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = ""
	}

	return map[string]any{
		"threadId": threadID.String(),
		"memory":   "",
		"config": map[string]any{
			"workingDir":    workingDir,
			"date":          time.Now().UTC().Format("2006-01-02"),
			"environment":   runtime.GOOS,
			"structure":     []any{},
			"isGitRepo":     false,
			"currentBranch": "",
			"mainBranch":    "",
			"gitStatus":     "",
			"recentCommits": []any{},
		},
		"params": params,
	}, nil
}

func flattenCommandCodeText(content any) string {
	if content == nil {
		return ""
	}
	if s, ok := content.(string); ok {
		return s
	}
	if arr, ok := content.([]any); ok {
		var parts []string
		for _, item := range arr {
			switch v := item.(type) {
			case string:
				parts = append(parts, v)
			case map[string]any:
				if text, ok := v["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return fmt.Sprintf("%v", content)
}

func toCommandCodeContentBlocks(content any) []any {
	if content == nil {
		return []any{map[string]any{"type": "text", "text": ""}}
	}
	if s, ok := content.(string); ok {
		return []any{map[string]any{"type": "text", "text": s}}
	}
	if arr, ok := content.([]any); ok {
		blocks := make([]any, 0)
		for _, item := range arr {
			switch v := item.(type) {
			case string:
				blocks = append(blocks, map[string]any{"type": "text", "text": v})
			case map[string]any:
				typ, _ := v["type"].(string)
				if typ == "text" {
					if text, ok := v["text"].(string); ok {
						blocks = append(blocks, map[string]any{"type": "text", "text": text})
					}
				} else if typ == "image_url" || typ == "image" {
					blocks = append(blocks, map[string]any{"type": "text", "text": "[image omitted]"})
				} else if text, ok := v["text"].(string); ok {
					blocks = append(blocks, map[string]any{"type": "text", "text": text})
				}
			}
		}
		if len(blocks) > 0 {
			return blocks
		}
		return []any{map[string]any{"type": "text", "text": ""}}
	}
	return []any{map[string]any{"type": "text", "text": fmt.Sprintf("%v", content)}}
}

func safeParseJSON(s any) any {
	if s == nil {
		return map[string]any{}
	}
	str, ok := s.(string)
	if !ok {
		return s
	}
	var result any
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return map[string]any{}
	}
	return result
}

func convertCommandCodeMessages(body map[string]any) ([]any, string) {
	rawMsgs, _ := body["messages"].([]any)
	out := make([]any, 0)
	var systemTexts []string

	for _, msg := range rawMsgs {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)

		if role == "system" {
			t := flattenCommandCodeText(m["content"])
			if t != "" {
				systemTexts = append(systemTexts, t)
			}
			continue
		}

		if role == "tool" {
			value := ""
			if s, ok := m["content"].(string); ok {
				value = s
			} else {
				value = flattenCommandCodeText(m["content"])
			}
			out = append(out, map[string]any{
				"role": "tool",
				"content": []any{
					map[string]any{
						"type":       "tool-result",
						"toolCallId": m["tool_call_id"],
						"toolName":   m["name"],
						"output": map[string]any{
							"type":  "text",
							"value": value,
						},
					},
				},
			})
			continue
		}

		if role == "assistant" {
			blocks := make([]any, 0)
			text := flattenCommandCodeText(m["content"])
			if text != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": text})
			}
			if rawToolCalls, ok := m["tool_calls"].([]any); ok {
				for _, tc := range rawToolCalls {
					tcMap, ok := tc.(map[string]any)
					if !ok {
						continue
					}
					var fn map[string]any
					if f, ok := tcMap["function"].(map[string]any); ok {
						fn = f
					}
					blocks = append(blocks, map[string]any{
						"type":       "tool-call",
						"toolCallId": tcMap["id"],
						"toolName":   fn["name"],
						"input":      safeParseJSON(fn["arguments"]),
					})
				}
			}
			if len(blocks) == 0 {
				blocks = append(blocks, map[string]any{"type": "text", "text": ""})
			}
			out = append(out, map[string]any{
				"role":    "assistant",
				"content": blocks,
			})
			continue
		}

		out = append(out, map[string]any{
			"role":    "user",
			"content": toCommandCodeContentBlocks(m["content"]),
		})
	}

	systemStr := ""
	if len(systemTexts) > 0 {
		systemStr = ""
		for i, s := range systemTexts {
			if i > 0 {
				systemStr += "\n\n"
			}
			systemStr += s
		}
	}
	return out, systemStr
}

func convertCommandCodeTools(body map[string]any) []any {
	rawTools, ok := body["tools"].([]any)
	if !ok || len(rawTools) == 0 {
		return nil
	}
	result := make([]any, 0)
	for _, tool := range rawTools {
		toolMap, ok := tool.(map[string]any)
		if !ok {
			continue
		}
		if toolMap == nil {
			continue
		}

		if toolType, _ := toolMap["type"].(string); toolType == "function" {
			if fn, ok := toolMap["function"].(map[string]any); ok && fn != nil {
				name := ""
				if n, ok := fn["name"].(string); ok {
					name = n
				}
				desc := ""
				if d, ok := fn["description"].(string); ok {
					desc = d
				}
				schema := map[string]any{"type": "object"}
				if params, ok := fn["parameters"].(map[string]any); ok {
					schema = params
				}
				result = append(result, map[string]any{
					"name":         name,
					"description":  desc,
					"input_schema": schema,
				})
				continue
			}
		}

		// Claude-like shape fallback.
		if name, ok := toolMap["name"].(string); ok && name != "" {
			schema := map[string]any{"type": "object"}
			if is, ok := toolMap["input_schema"].(map[string]any); ok {
				schema = is
			} else if params, ok := toolMap["parameters"].(map[string]any); ok {
				schema = params
			}
			result = append(result, map[string]any{
				"name":         name,
				"description":  toolMap["description"],
				"input_schema": schema,
			})
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
