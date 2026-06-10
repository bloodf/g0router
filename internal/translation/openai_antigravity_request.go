package translation

import (
	"fmt"
	"strings"
)

func isClaudeModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "claude")
}

func openaiToAntigravityRequest(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error) {
	if isClaudeModel(model) {
		claudeRequest, err := openaiToClaudeRequestForAntigravity(model, body, stream, credentials)
		if err != nil {
			return nil, fmt.Errorf("openaiToAntigravityRequest: %w", err)
		}
		env, err := wrapInCloudCodeEnvelopeForClaude(model, claudeRequest, credentials)
		if err != nil {
			return nil, fmt.Errorf("openaiToAntigravityRequest: %w", err)
		}
		return env, nil
	}

	geminiCLI, err := openaiToGeminiCLIRequest(model, body, stream, credentials)
	if err != nil {
		return nil, fmt.Errorf("openaiToAntigravityRequest: %w", err)
	}
	env, err := wrapInCloudCodeEnvelope(model, geminiCLI, credentials, true)
	if err != nil {
		return nil, fmt.Errorf("openaiToAntigravityRequest: %w", err)
	}
	return env, nil
}
