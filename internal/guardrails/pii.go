package guardrails

import (
	"regexp"

	"github.com/bloodf/g0router/internal/providers"
)

var piiRegexes = map[string]*regexp.Regexp{
	"email":       regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
	"phone":       regexp.MustCompile(`(?:\+\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`),
	"ssn":         regexp.MustCompile(`\b\d{3}[-.\s]?\d{2}[-.\s]?\d{4}\b`),
	"credit_card": regexp.MustCompile(`\b(?:\d{4}[-.\s]?){3}\d{4}\b`),
	"ip_address":  regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`),
}

// PIIConfig holds PII redaction configuration.
type PIIConfig struct {
	Enabled bool
	Types   []string
}

// Redact replaces PII spans in the input with [REDACTED:<type>].
func (c PIIConfig) Redact(input string) string {
	if !c.Enabled || input == "" {
		return input
	}

	result := input
	for _, typ := range c.Types {
		re, ok := piiRegexes[typ]
		if !ok {
			continue
		}
		result = re.ReplaceAllString(result, "[REDACTED:"+typ+"]")
	}
	return result
}

// RedactRequest returns a new ChatRequest with PII redacted from all message
// content. The original request is not mutated.
func RedactRequest(cfg PIIConfig, req *providers.ChatRequest) *providers.ChatRequest {
	if req == nil || !cfg.Enabled || len(cfg.Types) == 0 {
		return req
	}

	copied := *req
	copied.Messages = make([]providers.Message, len(req.Messages))
	for i, msg := range req.Messages {
		copied.Messages[i] = msg
		copied.Messages[i].Content = redactContent(cfg, msg.Content)
	}
	return &copied
}

func redactContent(cfg PIIConfig, content any) any {
	switch v := content.(type) {
	case string:
		return cfg.Redact(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			if m, ok := item.(map[string]any); ok {
				mm := make(map[string]any, len(m))
				for k, val := range m {
					mm[k] = val
				}
				if mm["type"] == "text" {
					if text, ok := mm["text"].(string); ok {
						mm["text"] = cfg.Redact(text)
					}
				}
				out[i] = mm
			} else {
				out[i] = item
			}
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(v))
		for i, m := range v {
			mm := make(map[string]any, len(m))
			for k, val := range m {
				mm[k] = val
			}
			if mm["type"] == "text" {
				if text, ok := mm["text"].(string); ok {
					mm["text"] = cfg.Redact(text)
				}
			}
			out[i] = mm
		}
		return out
	default:
		return content
	}
}
