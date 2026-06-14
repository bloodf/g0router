package cursor

import "strings"

// toolCall is a decoded ClientSideToolV2Call from the response.
type toolCall struct {
	id        string
	name      string
	arguments string
	isLast    bool
}

// extractResult is the decoded result of a single response frame
// (cursorProtobuf.js extractTextFromResponse return shape).
type extractResult struct {
	text     string
	thinking string
	toolCall *toolCall
}

// extractTextFromResponse decodes a StreamUnifiedChatResponseWithTools payload,
// extracting a tool call (field 1) or text+thinking (field 2)
// (cursorProtobuf.js extractTextFromResponse).
func extractTextFromResponse(payload []byte) extractResult {
	fields := decodeMessage(payload)

	// Field 1: ClientSideToolV2Call.
	if tcData, ok := firstLen(fields, fieldToolCall); ok {
		if tc := extractToolCall(tcData); tc != nil {
			return extractResult{toolCall: tc}
		}
	}

	// Field 2: StreamUnifiedChatResponse.
	if respData, ok := firstLen(fields, fieldResponse); ok {
		text, thinking := extractTextAndThinking(respData)
		if text != "" || thinking != "" {
			return extractResult{text: text, thinking: thinking}
		}
	}

	return extractResult{}
}

// extractToolCall decodes a ClientSideToolV2Call message (cursorProtobuf.js
// extractToolCall).
func extractToolCall(data []byte) *toolCall {
	tc := decodeMessage(data)
	out := &toolCall{}

	if idData, ok := firstLen(tc, fieldToolID); ok {
		// Cursor returns a multi-line ID; take the first line.
		full := string(idData)
		if i := strings.IndexByte(full, '\n'); i >= 0 {
			full = full[:i]
		}
		out.id = full
	}
	if nameData, ok := firstLen(tc, fieldToolName); ok {
		out.name = string(nameData)
	}
	if v, ok := firstVarint(tc, fieldToolIsLast); ok {
		out.isLast = v != 0
	}

	// MCP params carry the real tool name + raw args.
	if mcpData, ok := firstLen(tc, fieldToolMCPParams); ok {
		mcp := decodeMessage(mcpData)
		if toolData, ok := firstLen(mcp, fieldMCPToolsList); ok {
			nested := decodeMessage(toolData)
			if nameData, ok := firstLen(nested, fieldMCPNestedName); ok {
				out.name = string(nameData)
			}
			if argData, ok := firstLen(nested, fieldMCPNestedParam); ok {
				out.arguments = string(argData)
			}
		}
	}

	// Fallback to raw_args.
	if out.arguments == "" {
		if argData, ok := firstLen(tc, fieldToolRawArgs); ok {
			out.arguments = string(argData)
		}
	}

	if out.id != "" && out.name != "" {
		if out.arguments == "" {
			out.arguments = "{}"
		}
		return out
	}
	return nil
}

// extractTextAndThinking decodes a StreamUnifiedChatResponse message
// (cursorProtobuf.js extractTextAndThinking).
func extractTextAndThinking(data []byte) (text, thinking string) {
	nested := decodeMessage(data)
	if textData, ok := firstLen(nested, fieldResponseText); ok {
		text = string(textData)
	}
	if thinkData, ok := firstLen(nested, fieldThinking); ok {
		thinkingMsg := decodeMessage(thinkData)
		if tt, ok := firstLen(thinkingMsg, fieldThinkingText); ok {
			thinking = string(tt)
		}
	}
	return text, thinking
}
