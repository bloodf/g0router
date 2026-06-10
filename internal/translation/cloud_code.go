package translation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	antigravityDefaultSystem = "You are Antigravity, a powerful agentic AI coding assistant designed by the Google Deepmind team working on Advanced Agentic Coding.You are pair programming with a USER to solve their coding task. The task may require creating a new codebase, modifying or debugging an existing codebase, or simply answering a question.**Absolute paths only****Proactiveness**"
)

func randomUUIDString() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("random uuid: %w", err)
	}
	return u.String(), nil
}

func generateProjectId() (string, error) {
	adjectives := []string{"useful", "bright", "swift", "calm", "bold"}
	nouns := []string{"fuze", "wave", "spark", "flow", "core"}
	u, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("random uuid: %w", err)
	}
	// uuid v4 bytes 0 and 1 are fully random; reuse them so no extra
	// randomness source is needed.
	adj := adjectives[int(u[0])%len(adjectives)]
	noun := nouns[int(u[1])%len(nouns)]
	return fmt.Sprintf("%s-%s-%s", adj, noun, u.String()[:5]), nil
}

func generateRequestId() (string, error) {
	raw, err := randomUUIDString()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("agent-%s", raw), nil
}

func generateSessionId() (string, error) {
	raw, err := randomUUIDString()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", raw, time.Now().UnixMilli()), nil
}

func deriveSessionId(key string) (string, error) {
	if key == "" {
		return generateSessionId()
	}
	h := sha256.Sum256([]byte(key))
	return "sess-" + hex.EncodeToString(h[:16]), nil
}

func wrapInCloudCodeEnvelope(model string, geminiCLI map[string]any, credentials map[string]any, isAntigravity bool) (map[string]any, error) {
	projectId, err := generateProjectId()
	if err != nil {
		return nil, fmt.Errorf("wrapInCloudCodeEnvelope: %w", err)
	}
	if credentials != nil {
		if pid, ok := credentials["projectId"].(string); ok && pid != "" {
			projectId = pid
		}
	}

	userAgent := "gemini-cli"
	requestId, err := generateRequestId()
	if err != nil {
		return nil, fmt.Errorf("wrapInCloudCodeEnvelope: %w", err)
	}
	if isAntigravity {
		userAgent = "antigravity"
	}

	sessionId, err := generateSessionId()
	if err != nil {
		return nil, fmt.Errorf("wrapInCloudCodeEnvelope: %w", err)
	}
	if isAntigravity {
		key := ""
		if credentials != nil {
			if email, ok := credentials["email"].(string); ok && email != "" {
				key = email
			} else if connId, ok := credentials["connectionId"].(string); ok && connId != "" {
				key = connId
			}
		}
		sessionId, err = deriveSessionId(key)
		if err != nil {
			return nil, fmt.Errorf("wrapInCloudCodeEnvelope: %w", err)
		}
	}

	request := map[string]any{
		"sessionId":       sessionId,
		"contents":        geminiCLI["contents"],
		"systemInstruction": geminiCLI["systemInstruction"],
		"generationConfig": geminiCLI["generationConfig"],
		"tools":           geminiCLI["tools"],
	}

	if isAntigravity {
		request["requestType"] = "agent"

		systemParts := []map[string]any{
			{"text": antigravityDefaultSystem},
			{"text": fmt.Sprintf("Please ignore the following [ignore]%s[/ignore]", antigravityDefaultSystem)},
		}

		if si, ok := request["systemInstruction"].(map[string]any); ok && si != nil {
			if parts, ok := si["parts"].([]any); ok && len(parts) > 0 {
				newParts := make([]any, 0, len(systemParts)+len(parts))
				for _, p := range systemParts {
					newParts = append(newParts, p)
				}
				newParts = append(newParts, parts...)
				si["parts"] = newParts
			} else {
				request["systemInstruction"] = map[string]any{
					"role":  "user",
					"parts": toAnySlice(systemParts),
				}
			}
		} else {
			request["systemInstruction"] = map[string]any{
				"role":  "user",
				"parts": toAnySlice(systemParts),
			}
		}

		if tools, ok := geminiCLI["tools"].([]any); ok && len(tools) > 0 {
			request["toolConfig"] = map[string]any{
				"functionCallingConfig": map[string]any{"mode": "VALIDATED"},
			}
		}
	} else {
		if ss, ok := geminiCLI["safetySettings"]; ok {
			request["safetySettings"] = ss
		}
	}

	envelope := map[string]any{
		"project":   projectId,
		"model":     model,
		"userAgent": userAgent,
		"requestId": requestId,
		"request":   request,
	}

	if isAntigravity {
		envelope["requestType"] = "agent"
	}

	return envelope, nil
}

func wrapInCloudCodeEnvelopeForClaude(model string, claudeRequest map[string]any, credentials map[string]any) (map[string]any, error) {
	projectId, err := generateProjectId()
	if err != nil {
		return nil, fmt.Errorf("wrapInCloudCodeEnvelopeForClaude: %w", err)
	}
	if credentials != nil {
		if pid, ok := credentials["projectId"].(string); ok && pid != "" {
			projectId = pid
		}
	}

	key := ""
	if credentials != nil {
		if email, ok := credentials["email"].(string); ok && email != "" {
			key = email
		} else if connId, ok := credentials["connectionId"].(string); ok && connId != "" {
			key = connId
		}
	}

	// Build tool_use id -> name map
	toolUseIdToName := make(map[string]string)
	if msgs, ok := claudeRequest["messages"].([]any); ok {
		for _, msg := range msgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			content, ok := m["content"].([]any)
			if !ok {
				continue
			}
			for _, block := range content {
				b, ok := block.(map[string]any)
				if !ok {
					continue
				}
				if b["type"] == "tool_use" {
					if id, ok := b["id"].(string); ok {
						if name, ok := b["name"].(string); ok {
							toolUseIdToName[id] = name
						}
					}
				}
			}
		}
	}

	sessionId, err := deriveSessionId(key)
	if err != nil {
		return nil, fmt.Errorf("wrapInCloudCodeEnvelopeForClaude: %w", err)
	}
	requestId, err := generateRequestId()
	if err != nil {
		return nil, fmt.Errorf("wrapInCloudCodeEnvelopeForClaude: %w", err)
	}

	request := map[string]any{
		"sessionId": sessionId,
		"contents":  []any{},
		"generationConfig": map[string]any{
			"temperature":     claudeRequest["temperature"],
			"maxOutputTokens": claudeRequest["max_tokens"],
		},
	}

	// Convert Claude messages to Gemini contents
	if msgs, ok := claudeRequest["messages"].([]any); ok {
		for _, msg := range msgs {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			parts := []map[string]any{}
			content := m["content"]
			if contentArr, ok := content.([]any); ok {
				for _, block := range contentArr {
					b, ok := block.(map[string]any)
					if !ok {
						continue
					}
					switch b["type"] {
					case "text":
						if text, ok := b["text"].(string); ok {
							parts = append(parts, map[string]any{"text": text})
						}
					case "tool_use":
						id := ""
						if s, ok := b["id"].(string); ok {
							id = s
						}
						name := ""
						if s, ok := b["name"].(string); ok {
							name = s
						}
						input := map[string]any{}
						if inp, ok := b["input"].(map[string]any); ok {
							input = inp
						}
						parts = append(parts, map[string]any{
							"functionCall": map[string]any{
								"id":   id,
								"name": sanitizeGeminiFunctionName(name),
								"args": input,
							},
						})
					case "tool_result":
						contentVal := b["content"]
						if arr, ok := contentVal.([]any); ok {
							texts := make([]string, 0, len(arr))
							for _, item := range arr {
								if im, ok := item.(map[string]any); ok {
									if im["type"] == "text" {
										if t, ok := im["text"].(string); ok {
											texts = append(texts, t)
										}
									}
								}
							}
							contentVal = texts
						}
						toolUseID := ""
						if s, ok := b["tool_use_id"].(string); ok {
							toolUseID = s
						}
						resolvedName := toolUseIdToName[toolUseID]
						if resolvedName == "" {
							resolvedName = "tool"
						}
						parts = append(parts, map[string]any{
							"functionResponse": map[string]any{
								"id":   toolUseID,
								"name": sanitizeGeminiFunctionName(resolvedName),
								"response": map[string]any{
									"result": tryParseJSONValue(contentVal),
								},
							},
						})
					}
				}
			} else if text, ok := content.(string); ok && text != "" {
				parts = append(parts, map[string]any{"text": text})
			}

			if len(parts) > 0 {
				role := "user"
				if r, ok := m["role"].(string); ok && r == "assistant" {
					role = "model"
				}
				request["contents"] = append(request["contents"].([]any), map[string]any{
					"role":  role,
					"parts": toAnySlice(parts),
				})
			}
		}
	}

	// Convert Claude tools to Gemini functionDeclarations
	if tools, ok := claudeRequest["tools"].([]any); ok && len(tools) > 0 {
		functionDeclarations := []map[string]any{}
		for _, tool := range tools {
			tm, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			name := ""
			if s, ok := tm["name"].(string); ok {
				name = s
			}
			desc := ""
			if s, ok := tm["description"].(string); ok {
				desc = s
			}
			schema := map[string]any{"type": "object", "properties": map[string]any{}}
			if is, ok := tm["input_schema"].(map[string]any); ok {
				schema = is
			}
			functionDeclarations = append(functionDeclarations, map[string]any{
				"name":        sanitizeGeminiFunctionName(name),
				"description": desc,
				"parameters":  schema,
			})
		}
		if len(functionDeclarations) > 0 {
			request["tools"] = []any{map[string]any{"functionDeclarations": toAnySlice(functionDeclarations)}}
			request["toolConfig"] = map[string]any{
				"functionCallingConfig": map[string]any{"mode": "VALIDATED"},
			}
		}
	}

	// System instruction
	systemParts := []map[string]any{
		{"text": antigravityDefaultSystem},
		{"text": fmt.Sprintf("Please ignore the following [ignore]%s[/ignore]", antigravityDefaultSystem)},
	}

	if sys, ok := claudeRequest["system"]; ok && sys != nil {
		switch v := sys.(type) {
		case []any:
			for _, block := range v {
				if b, ok := block.(map[string]any); ok {
					if text, ok := b["text"].(string); ok {
						systemParts = append(systemParts, map[string]any{"text": text})
					}
				}
			}
		case string:
			if v != "" {
				systemParts = append(systemParts, map[string]any{"text": v})
			}
		}
	}

	if si, ok := request["systemInstruction"].(map[string]any); ok && si != nil {
		if parts, ok := si["parts"].([]any); ok && len(parts) > 0 {
			newParts := make([]any, 0, len(systemParts)+len(parts))
			for _, p := range systemParts {
				newParts = append(newParts, p)
			}
			newParts = append(newParts, parts...)
			si["parts"] = newParts
		} else {
			request["systemInstruction"] = map[string]any{
				"role":  "user",
				"parts": toAnySlice(systemParts),
			}
		}
	} else {
		request["systemInstruction"] = map[string]any{
			"role":  "user",
			"parts": toAnySlice(systemParts),
		}
	}

	return map[string]any{
		"project":     projectId,
		"model":       model,
		"userAgent":   "antigravity",
		"requestId":   requestId,
		"requestType": "agent",
		"request":     request,
	}, nil
}
