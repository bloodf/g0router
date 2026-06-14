package cursor

import (
	"encoding/json"
	"runtime"
	"time"
)

// encodeMessage encodes a ConversationMessage (cursorProtobuf.js encodeMessage).
// Tool results are not encoded here (this adapter sends them inline only when
// present in the OpenAI message content; the ref's separate tool-result frame is
// not used by g0router's single-request path).
func encodeMessage(content string, role int, messageID string, isLast, hasTools bool) []byte {
	var out []byte
	out = append(out, encodeFieldStr(fieldMsgContent, content)...)
	out = append(out, encodeFieldVarint(fieldMsgRole, uint64(role))...)
	out = append(out, encodeFieldStr(fieldMsgID, messageID)...)
	if hasTools {
		out = append(out, encodeFieldVarint(fieldMsgIsAgent, 1)...)
	} else {
		out = append(out, encodeFieldVarint(fieldMsgIsAgent, 0)...)
	}
	mode := unifiedModeChat
	if hasTools {
		mode = unifiedModeAgent
	}
	out = append(out, encodeFieldVarint(fieldMsgUnified, uint64(mode))...)
	if isLast && hasTools {
		out = append(out, encodeFieldLen(fieldMsgSupTools, encodeVarint(1))...)
	}
	return out
}

// encodeModel encodes the Model message (cursorProtobuf.js encodeModel).
func encodeModel(modelName string) []byte {
	out := encodeFieldStr(fieldModelName, modelName)
	out = append(out, encodeFieldLen(fieldModelEmpty, nil)...)
	return out
}

// encodeInstruction encodes the Instruction message; empty text yields no bytes
// (cursorProtobuf.js encodeInstruction).
func encodeInstruction(text string) []byte {
	if text == "" {
		return nil
	}
	return encodeFieldStr(fieldInstructionText, text)
}

// encodeCursorSetting encodes the CursorSetting message with the magic constant
// fields the ref sends (cursorProtobuf.js encodeCursorSetting).
func encodeCursorSetting() []byte {
	unknown6 := append(encodeFieldLen(fieldSetting6Field1, nil), encodeFieldLen(fieldSetting6Field2, nil)...)
	var out []byte
	out = append(out, encodeFieldStr(fieldSettingPath, "cursor\\aisettings")...)
	out = append(out, encodeFieldLen(fieldSettingUnknown3, nil)...)
	out = append(out, encodeFieldLen(fieldSettingUnknown6, unknown6)...)
	out = append(out, encodeFieldVarint(fieldSettingUnknown8, 1)...)
	out = append(out, encodeFieldVarint(fieldSettingUnknown9, 1)...)
	return out
}

// encodeMetadata encodes the Metadata message (cursorProtobuf.js encodeMetadata).
func encodeMetadata() []byte {
	var out []byte
	out = append(out, encodeFieldStr(fieldMetaPlatform, runtime.GOOS)...)
	out = append(out, encodeFieldStr(fieldMetaArch, runtime.GOARCH)...)
	out = append(out, encodeFieldStr(fieldMetaVersion, runtime.Version())...)
	out = append(out, encodeFieldStr(fieldMetaCWD, "/")...)
	out = append(out, encodeFieldStr(fieldMetaTimestamp, time.Now().UTC().Format(time.RFC3339))...)
	return out
}

// encodeMessageID encodes a MessageId message (cursorProtobuf.js encodeMessageId).
func encodeMessageID(messageID string, role int) []byte {
	out := encodeFieldStr(fieldMsgIDID, messageID)
	out = append(out, encodeFieldVarint(fieldMsgIDRole, uint64(role))...)
	return out
}

// encodeMCPTool encodes an MCPTool from an OpenAI tool definition
// (cursorProtobuf.js encodeMcpTool).
func encodeMCPTool(tool map[string]any) []byte {
	name := ""
	desc := ""
	var params any
	if fn, ok := tool["function"].(map[string]any); ok {
		name, _ = fn["name"].(string)
		desc, _ = fn["description"].(string)
		params = fn["parameters"]
	} else {
		name, _ = tool["name"].(string)
		desc, _ = tool["description"].(string)
		params = tool["input_schema"]
	}

	var out []byte
	if name != "" {
		out = append(out, encodeFieldStr(fieldMCPToolName, name)...)
	}
	if desc != "" {
		out = append(out, encodeFieldStr(fieldMCPToolDesc, desc)...)
	}
	if pm, ok := params.(map[string]any); ok && len(pm) > 0 {
		if b, err := json.Marshal(pm); err == nil {
			out = append(out, encodeFieldLen(fieldMCPToolParams, b)...)
		}
	}
	out = append(out, encodeFieldStr(fieldMCPToolServer, "custom")...)
	return out
}

// encodeRequest builds the StreamUnifiedChatRequest body (cursorProtobuf.js
// encodeRequest). messages are {role, content} maps in OpenAI shape.
func encodeRequest(messages []map[string]any, modelName string, tools []map[string]any, reasoningEffort string, forceAgentMode bool) []byte {
	hasTools := len(tools) > 0
	isAgentic := hasTools || forceAgentMode

	type fmtMsg struct {
		content   string
		role      int
		messageID string
		isLast    bool
	}
	var formatted []fmtMsg
	var messageIDs []fmtMsg
	for i, msg := range messages {
		role := roleAssistant
		if r, _ := msg["role"].(string); r == "user" {
			role = roleUser
		}
		content, _ := msg["content"].(string)
		id := newUUID()
		fm := fmtMsg{content: content, role: role, messageID: id, isLast: i == len(messages)-1}
		formatted = append(formatted, fm)
		messageIDs = append(messageIDs, fm)
	}

	thinkingLevel := thinkingUnspecified
	switch reasoningEffort {
	case "medium":
		thinkingLevel = thinkingMedium
	case "high":
		thinkingLevel = thinkingHigh
	}

	var out []byte
	// Messages.
	for _, fm := range formatted {
		out = append(out, encodeFieldLen(fieldMessages, encodeMessage(fm.content, fm.role, fm.messageID, fm.isLast, hasTools))...)
	}
	// Static fields.
	out = append(out, encodeFieldVarint(fieldUnknown2, 1)...)
	out = append(out, encodeFieldLen(fieldInstruction, encodeInstruction(""))...)
	out = append(out, encodeFieldVarint(fieldUnknown4, 1)...)
	out = append(out, encodeFieldLen(fieldModel, encodeModel(modelName))...)
	out = append(out, encodeFieldStr(fieldWebTool, "")...)
	out = append(out, encodeFieldVarint(fieldUnknown13, 1)...)
	out = append(out, encodeFieldLen(fieldCursorSetting, encodeCursorSetting())...)
	out = append(out, encodeFieldVarint(fieldUnknown19, 1)...)
	out = append(out, encodeFieldStr(fieldConversationID, newUUID())...)
	out = append(out, encodeFieldLen(fieldMetadata, encodeMetadata())...)
	// Tool-related fields.
	out = append(out, encodeFieldVarint(fieldIsAgentic, boolToVarint(isAgentic))...)
	if isAgentic {
		out = append(out, encodeFieldLen(fieldSupportedTools, encodeVarint(1))...)
	}
	// Message IDs.
	for _, mid := range messageIDs {
		out = append(out, encodeFieldLen(fieldMessageIDs, encodeMessageID(mid.messageID, mid.role))...)
	}
	// MCP tools.
	for _, tool := range tools {
		out = append(out, encodeFieldLen(fieldMCPTools, encodeMCPTool(tool))...)
	}
	// Mode fields.
	out = append(out, encodeFieldVarint(fieldLargeContext, 0)...)
	out = append(out, encodeFieldVarint(fieldUnknown38, 0)...)
	mode := unifiedModeChat
	if isAgentic {
		mode = unifiedModeAgent
	}
	out = append(out, encodeFieldVarint(fieldUnifiedMode, uint64(mode))...)
	out = append(out, encodeFieldStr(fieldUnknown47, "")...)
	out = append(out, encodeFieldVarint(fieldShouldDisable, boolToVarint(!isAgentic))...)
	out = append(out, encodeFieldVarint(fieldThinkingLevel, uint64(thinkingLevel))...)
	out = append(out, encodeFieldVarint(fieldUnknown51, 0)...)
	out = append(out, encodeFieldVarint(fieldUnknown53, 1)...)
	modeName := "Ask"
	if isAgentic {
		modeName = "Agent"
	}
	out = append(out, encodeFieldStr(fieldUnifiedModeName, modeName)...)
	return out
}

// buildChatRequest wraps the request in the top-level REQUEST field
// (cursorProtobuf.js buildChatRequest).
func buildChatRequest(messages []map[string]any, modelName string, tools []map[string]any, reasoningEffort string, forceAgentMode bool) []byte {
	return encodeFieldLen(fieldRequest, encodeRequest(messages, modelName, tools, reasoningEffort, forceAgentMode))
}

// generateCursorBody produces the connect-framed protobuf request body
// (cursorProtobuf.js generateCursorBody). Cursor does not accept compressed
// requests, so the frame is uncompressed.
func generateCursorBody(messages []map[string]any, modelName string, tools []map[string]any, reasoningEffort string, forceAgentMode bool) []byte {
	pb := buildChatRequest(messages, modelName, tools, reasoningEffort, forceAgentMode)
	return wrapConnectFrame(pb, false)
}

func boolToVarint(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}
