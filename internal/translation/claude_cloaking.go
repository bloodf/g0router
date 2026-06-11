package translation

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	claudeVersion = "2.1.92"
	ccEntrypoint  = "sdk-cli"
)

func generateBillingHeader(payload map[string]any) (string, error) {
	content, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}
	h := sha256.Sum256(content)
	cch := hex.EncodeToString(h[:])[:5]

	buildBytes := make([]byte, 2)
	if _, err := rand.Read(buildBytes); err != nil {
		return "", fmt.Errorf("read rand: %w", err)
	}
	buildHash := hex.EncodeToString(buildBytes)[:3]

	return fmt.Sprintf("x-anthropic-billing-header: cc_version=%s.%s; cc_entrypoint=%s; cch=%s;", claudeVersion, buildHash, ccEntrypoint, cch), nil
}

func generateFakeUserID(sessionID string) (string, error) {
	deviceBytes := make([]byte, 32)
	if _, err := rand.Read(deviceBytes); err != nil {
		return "", fmt.Errorf("read rand: %w", err)
	}
	deviceID := hex.EncodeToString(deviceBytes)

	accountUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("account uuid: %w", err)
	}

	sessionUUID := sessionID
	if sessionUUID == "" {
		s, err := uuid.NewRandom()
		if err != nil {
			return "", fmt.Errorf("session uuid: %w", err)
		}
		sessionUUID = s.String()
	}

	obj := map[string]any{
		"device_id":    deviceID,
		"account_uuid": accountUUID.String(),
		"session_id":   sessionUUID,
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal fake user id: %w", err)
	}
	return string(b), nil
}

func cloakClaudeTools(body map[string]any) (map[string]any, map[string]string, error) {
	rawTools, ok := body["tools"].([]any)
	if !ok || len(rawTools) == 0 {
		return body, nil, nil
	}

	toolNameMap := make(map[string]string)
	clientToolNames := make(map[string]struct{})
	clientDeclarations := make([]any, 0, len(rawTools))

	for _, rt := range rawTools {
		tool, ok := rt.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tool["name"].(string)
		suffixed := name + claudeToolSuffix
		toolNameMap[suffixed] = name
		clientToolNames[name] = struct{}{}

		cloned := make(map[string]any, len(tool))
		for k, v := range tool {
			cloned[k] = v
		}
		cloned["name"] = suffixed
		clientDeclarations = append(clientDeclarations, cloned)
	}

	allTools := make([]any, 0, len(clientDeclarations)+len(ccDecoyTools))
	allTools = append(allTools, clientDeclarations...)
	for _, dt := range ccDecoyTools {
		allTools = append(allTools, map[string]any{
			"name":         dt.name,
			"description":  dt.description,
			"input_schema": dt.inputSchema,
		})
	}

	var renamedMessages []any
	if rawMsgs, ok := body["messages"].([]any); ok {
		renamedMessages = make([]any, 0, len(rawMsgs))
		for _, rm := range rawMsgs {
			msg, ok := rm.(map[string]any)
			if !ok {
				renamedMessages = append(renamedMessages, rm)
				continue
			}
			content, ok := msg["content"].([]any)
			if !ok {
				renamedMessages = append(renamedMessages, msg)
				continue
			}
			newContent := make([]any, 0, len(content))
			for _, cb := range content {
				block, ok := cb.(map[string]any)
				if !ok {
					newContent = append(newContent, cb)
					continue
				}
				if block["type"] == "tool_use" {
					clonedBlock := make(map[string]any, len(block))
					for k, v := range block {
						clonedBlock[k] = v
					}
					if n, ok := block["name"].(string); ok {
						clonedBlock["name"] = n + claudeToolSuffix
					}
					newContent = append(newContent, clonedBlock)
				} else {
					newContent = append(newContent, block)
				}
			}
			clonedMsg := make(map[string]any, len(msg))
			for k, v := range msg {
				clonedMsg[k] = v
			}
			clonedMsg["content"] = newContent
			renamedMessages = append(renamedMessages, clonedMsg)
		}
	}

	cloaked := make(map[string]any, len(body))
	for k, v := range body {
		cloaked[k] = v
	}
	cloaked["tools"] = allTools
	if renamedMessages != nil {
		cloaked["messages"] = renamedMessages
	}

	if tc, ok := body["tool_choice"].(map[string]any); ok {
		if tc["type"] == "tool" {
			if name, ok := tc["name"].(string); ok {
				if _, isClient := clientToolNames[name]; isClient {
					clonedTC := make(map[string]any, len(tc))
					for k, v := range tc {
						clonedTC[k] = v
					}
					clonedTC["name"] = name + claudeToolSuffix
					cloaked["tool_choice"] = clonedTC
				}
			}
		}
	}

	if len(toolNameMap) == 0 {
		return cloaked, nil, nil
	}
	return cloaked, toolNameMap, nil
}

func decloakToolNames(body map[string]any, toolNameMap map[string]string) map[string]any {
	if len(toolNameMap) == 0 {
		return body
	}
	content, ok := body["content"].([]any)
	if !ok {
		return body
	}
	newContent := make([]any, 0, len(content))
	for _, cb := range content {
		block, ok := cb.(map[string]any)
		if !ok {
			newContent = append(newContent, cb)
			continue
		}
		if block["type"] != "tool_use" {
			newContent = append(newContent, block)
			continue
		}
		name, _ := block["name"].(string)
		if orig, ok := toolNameMap[name]; ok {
			cloned := make(map[string]any, len(block))
			for k, v := range block {
				cloned[k] = v
			}
			cloned["name"] = orig
			newContent = append(newContent, cloned)
		} else {
			newContent = append(newContent, block)
		}
	}
	result := make(map[string]any, len(body))
	for k, v := range body {
		result[k] = v
	}
	result["content"] = newContent
	return result
}

func applyCloaking(body map[string]any, apiKey string, sessionID string) map[string]any {
	if !strings.Contains(apiKey, "sk-ant-oat") {
		return body
	}

	result := make(map[string]any, len(body))
	for k, v := range body {
		result[k] = v
	}

	billingText, err := generateBillingHeader(body)
	if err != nil {
		// On error, return body unchanged to avoid panics.
		return body
	}
	billingBlock := map[string]any{"type": "text", "text": billingText}

	switch sys := result["system"].(type) {
	case []any:
		if len(sys) > 0 {
			if first, ok := sys[0].(map[string]any); ok {
				if text, ok := first["text"].(string); ok && strings.HasPrefix(text, "x-anthropic-billing-header:") {
					// Already injected
					break
				}
			}
		}
		newSys := make([]any, 0, len(sys)+1)
		newSys = append(newSys, billingBlock)
		newSys = append(newSys, sys...)
		result["system"] = newSys
	case string:
		result["system"] = []any{billingBlock, map[string]any{"type": "text", "text": sys}}
	default:
		result["system"] = []any{billingBlock}
	}

	meta, _ := result["metadata"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	if _, ok := meta["user_id"]; !ok {
		fakeID, err := generateFakeUserID(sessionID)
		if err == nil {
			newMeta := make(map[string]any, len(meta)+1)
			for k, v := range meta {
				newMeta[k] = v
			}
			newMeta["user_id"] = fakeID
			result["metadata"] = newMeta
		}
	}

	return result
}
