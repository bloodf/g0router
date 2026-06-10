package translation

import (
	"fmt"
	"regexp"

	"github.com/bloodf/g0router/internal/schemas"
)

var (
	toolIDPattern    = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	toolIDInvalidRun = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

// sanitizeToolID keeps only alphanumeric characters, underscores, and hyphens.
// If the result is empty it returns "" so the caller can regenerate.
func sanitizeToolID(id string) string {
	if id == "" {
		return ""
	}
	return toolIDInvalidRun.ReplaceAllString(id, "")
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

// PreprocessChatRequest runs the full preprocessing pipeline on an OpenAI-
// shaped chat request in the order used by 9router.
func PreprocessChatRequest(req *schemas.ChatRequest) {
	NormalizeThinkingConfig(req)
	EnsureToolCallIDs(req)
	FixMissingToolResponses(req)
}

// NormalizeThinkingConfig removes Thinking and ReasoningEffort when the
// last message in the request is not from the user.
func NormalizeThinkingConfig(req *schemas.ChatRequest) {
	if len(req.Messages) == 0 {
		return
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role == "user" {
		return
	}
	req.Thinking = nil
	req.ReasoningEffort = ""
}

// FixMissingToolResponses inserts empty role:tool messages after assistant
// tool_calls when the next message does not contain results for those IDs.
// It operates on the OpenAI-shaped request body in place.
func FixMissingToolResponses(req *schemas.ChatRequest) {
	if len(req.Messages) == 0 {
		return
	}

	var newMessages []schemas.Message

	for i := 0; i < len(req.Messages); i++ {
		msg := req.Messages[i]
		newMessages = append(newMessages, msg)

		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			continue
		}

		var ids []string
		for _, tc := range msg.ToolCalls {
			if tc.ID != "" {
				ids = append(ids, tc.ID)
			}
		}
		if len(ids) == 0 {
			continue
		}

		nextMsg := func() *schemas.Message {
			if i+1 < len(req.Messages) {
				return &req.Messages[i+1]
			}
			return nil
		}()

		if nextMsg != nil && hasToolResults(*nextMsg, ids) {
			continue
		}

		if nextMsg == nil {
			continue
		}

		for _, id := range ids {
			idCopy := id
			newMessages = append(newMessages, schemas.Message{
				Role:       "tool",
				ToolCallID: &idCopy,
				Content:    "",
			})
		}
	}

	req.Messages = newMessages
}

// hasToolResults reports whether msg contains a tool result for any of the
// given toolCallIds. It checks the OpenAI role:tool format.
func hasToolResults(msg schemas.Message, toolCallIds []string) bool {
	if msg.Role == "tool" && msg.ToolCallID != nil {
		for _, id := range toolCallIds {
			if id == *msg.ToolCallID {
				return true
			}
		}
	}
	return false
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
