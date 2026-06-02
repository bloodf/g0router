package translate

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
)

func NormalizeOpenAI(body []byte) (*providers.ChatRequest, Format, error) {
	format, err := DetectFormat(body)
	if err != nil {
		return nil, FormatUnknown, err
	}

	switch format {
	case FormatOpenAI:
		req, err := parseChatRequest(body)
		if err != nil {
			return nil, format, err
		}
		return req, format, nil
	case FormatAnthropic:
		req, err := parseChatRequest(body)
		if err != nil {
			return nil, format, err
		}
		return anthropicToOpenAI(req), format, nil
	default:
		return nil, format, fmt.Errorf("normalize openai: unsupported format %q", format)
	}
}

func parseChatRequest(body []byte) (*providers.ChatRequest, error) {
	var req providers.ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse chat request: %w", err)
	}
	return &req, nil
}
