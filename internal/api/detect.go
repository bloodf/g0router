package api

import "github.com/bloodf/g0router/internal/translation"

// DetectFormat returns the request format string for the given body.
// Possible values: "openai", "claude", "gemini", "antigravity", "openai-responses".
// Ported from 9router provider.js:49-126 (PAR-ROUTE-033).
func DetectFormat(body map[string]any) string {
	return translation.DetectRequestFormat(body)
}
