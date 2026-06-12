package translation

import (
	"regexp"

	"github.com/bloodf/g0router/internal/schemas"
)

const reasoningPlaceholder = " "

var (
	deepseekModelPattern = regexp.MustCompile(`(?i)deepseek`)
	kimiModelPattern     = regexp.MustCompile(`(?i)^kimi-`)
)

func stringPtr(s string) *string { return &s }

func reasoningContentEmpty(rc *string) bool {
	return rc == nil || *rc == ""
}

// InjectReasoningContent injects a non-empty reasoning_content placeholder on
// assistant messages for providers/models that require it. Only model-name
// matching is used because PreprocessChatRequest does not receive a providerID.
func InjectReasoningContent(req *schemas.ChatRequest) {
	if len(req.Messages) == 0 {
		return
	}

	switch {
	case deepseekModelPattern.MatchString(req.Model):
		for i := range req.Messages {
			msg := &req.Messages[i]
			if msg.Role == "assistant" && reasoningContentEmpty(msg.ReasoningContent) {
				msg.ReasoningContent = stringPtr(reasoningPlaceholder)
			}
		}
	case kimiModelPattern.MatchString(req.Model):
		for i := range req.Messages {
			msg := &req.Messages[i]
			if msg.Role == "assistant" && len(msg.ToolCalls) > 0 && reasoningContentEmpty(msg.ReasoningContent) {
				msg.ReasoningContent = stringPtr(reasoningPlaceholder)
			}
		}
	}
}

// ExpandDeepseekModel applies the deepseek-v4-pro alias mappings. The max alias
// enables thinking and reasoning_effort; the none alias disables them.
func ExpandDeepseekModel(req *schemas.ChatRequest) {
	switch req.Model {
	case "deepseek-v4-pro-max":
		req.Model = "deepseek-v4-pro"
		req.Thinking = &schemas.ThinkingConfig{Type: "enabled"}
		req.ReasoningEffort = "max"
	case "deepseek-v4-pro-none":
		req.Model = "deepseek-v4-pro"
		req.Thinking = &schemas.ThinkingConfig{Type: "disabled"}
		req.ReasoningEffort = ""
	}
}
