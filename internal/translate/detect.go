package translate

import (
	"encoding/json"
	"fmt"
)

type Format string

const (
	FormatUnknown   Format = "unknown"
	FormatOpenAI    Format = "openai"
	FormatAnthropic Format = "anthropic"
	FormatGemini    Format = "gemini"
)

func (f Format) String() string {
	return string(f)
}

func DetectFormat(body []byte) (Format, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return FormatUnknown, fmt.Errorf("detect format: %w", err)
	}

	if _, ok := fields["contents"]; ok {
		return FormatGemini, nil
	}
	if rawSystem, ok := fields["system"]; ok && isAnthropicSystem(rawSystem) {
		return FormatAnthropic, nil
	}
	if _, ok := fields["messages"]; ok {
		return FormatOpenAI, nil
	}
	return FormatUnknown, nil
}

func isAnthropicSystem(raw json.RawMessage) bool {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return true
	}

	var blocks []any
	return json.Unmarshal(raw, &blocks) == nil
}
