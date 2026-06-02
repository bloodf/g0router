package translate

import (
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
)

func OpenAIToAnthropic(req *providers.ChatRequest) (*providers.ChatRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("openai to anthropic: nil request")
	}

	translated := *req
	translated.Messages = make([]providers.Message, 0, len(req.Messages))

	var systemParts []string
	for _, message := range req.Messages {
		if message.Role == "system" {
			systemParts = append(systemParts, systemContent(message.Content))
			continue
		}
		translated.Messages = append(translated.Messages, message)
	}
	if len(systemParts) > 0 {
		translated.System = strings.Join(systemParts, "\n\n")
	}
	return &translated, nil
}

func anthropicToOpenAI(req *providers.ChatRequest) *providers.ChatRequest {
	translated := *req
	translated.Messages = append([]providers.Message(nil), req.Messages...)
	if req.System != nil {
		translated.Messages = append([]providers.Message{
			{Role: "system", Content: req.System},
		}, translated.Messages...)
		translated.System = nil
	}
	return &translated
}

func systemContent(content any) string {
	switch value := content.(type) {
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}
