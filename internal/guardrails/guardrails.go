package guardrails

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
)

// ErrBlocklistMatch is returned when a prompt matches the guardrails blocklist.
var ErrBlocklistMatch = errors.New("prompt blocked by guardrails")

// Config holds guardrails configuration.
type Config struct {
	Enabled   bool
	Blocklist []string
}

// CheckBlocklist returns whether the prompt contains any blocklist term
// (case-insensitive). When disabled or the blocklist is empty, it always
// returns false.
func (c Config) CheckBlocklist(prompt string) (bool, []string) {
	if !c.Enabled || len(c.Blocklist) == 0 || prompt == "" {
		return false, nil
	}

	lowerPrompt := strings.ToLower(prompt)
	var matches []string
	seen := make(map[string]struct{})

	for _, term := range c.Blocklist {
		if term == "" {
			continue
		}
		lowerTerm := strings.ToLower(term)
		if strings.Contains(lowerPrompt, lowerTerm) {
			if _, ok := seen[lowerTerm]; !ok {
				seen[lowerTerm] = struct{}{}
				matches = append(matches, lowerTerm)
			}
		}
	}

	return len(matches) > 0, matches
}

// CheckRequest evaluates the guardrails blocklist against all message content
// in the chat request. It returns true, matched terms, and ErrBlocklistMatch
// when blocked.
func CheckRequest(cfg Config, req *providers.ChatRequest) (bool, []string, error) {
	if req == nil || !cfg.Enabled || len(cfg.Blocklist) == 0 {
		return false, nil, nil
	}

	var allMatches []string
	seen := make(map[string]struct{})

	for _, msg := range req.Messages {
		text := messageText(msg.Content)
		blocked, matches := cfg.CheckBlocklist(text)
		if blocked {
			for _, m := range matches {
				if _, ok := seen[m]; !ok {
					seen[m] = struct{}{}
					allMatches = append(allMatches, m)
				}
			}
		}
	}

	if len(allMatches) > 0 {
		return true, allMatches, fmt.Errorf("%w: matched %v", ErrBlocklistMatch, allMatches)
	}
	return false, nil, nil
}

func messageText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if t, ok := m["text"].(string); ok && m["type"] == "text" {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, " ")
	case []map[string]any:
		var parts []string
		for _, m := range v {
			if t, ok := m["text"].(string); ok && m["type"] == "text" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, " ")
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}
