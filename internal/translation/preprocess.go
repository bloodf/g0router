package translation

import (
	"fmt"
	"regexp"

	"github.com/bloodf/g0router/internal/schemas"
)

var toolIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// sanitizeToolID keeps only alphanumeric characters, underscores, and hyphens.
// If the result is empty it returns "" so the caller can regenerate.
func sanitizeToolID(id string) string {
	if id == "" {
		return ""
	}
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(id, "")
	return sanitized
}

// generateToolCallID creates a deterministic tool call ID from message index,
// tool-call index, and tool name (sanitized).
func generateToolCallID(msgIndex, tcIndex int, toolName string) string {
	name := ""
	if toolName != "" {
		sanitized := sanitizeToolID(toolName)
		if sanitized != "" {
			name = "_" + sanitized
		}
	}
	return fmt.Sprintf("call_msg%d_tc%d%s", msgIndex, tcIndex, name)
}

// EnsureToolCallIDs validates and repairs tool call IDs in a ChatRequest.
// It operates on the OpenAI-shaped request body in place.
func EnsureToolCallIDs(req *schemas.ChatRequest) {
	if len(req.Messages) == 0 {
		return
	}

	for i := range req.Messages {
		msg := &req.Messages[i]

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for j := range msg.ToolCalls {
				tc := &msg.ToolCalls[j]
				if tc.ID == "" || !toolIDPattern.MatchString(tc.ID) {
					sanitized := sanitizeToolID(tc.ID)
					if sanitized != "" {
						tc.ID = sanitized
					} else {
						tc.ID = generateToolCallID(i, j, tc.Function.Name)
					}
				}
				if tc.Type == "" {
					tc.Type = "function"
				}
			}
		}

		if msg.Role == "tool" && msg.ToolCallID != nil {
			if *msg.ToolCallID == "" || !toolIDPattern.MatchString(*msg.ToolCallID) {
				sanitized := sanitizeToolID(*msg.ToolCallID)
				if sanitized != "" {
					*msg.ToolCallID = sanitized
				} else {
					*msg.ToolCallID = generateToolCallID(i, 0, "")
				}
			}
		}
	}
}
