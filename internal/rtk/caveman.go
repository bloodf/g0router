package rtk

import "github.com/bloodf/g0router/internal/providers"

const cavemanSeparator = "\n\n"

func InjectCaveman(req providers.ChatRequest, level CavemanLevel) providers.ChatRequest {
	prompt, ok := cavemanPrompt(level)
	if !ok {
		return req
	}

	messages := make([]providers.Message, len(req.Messages))
	copy(messages, req.Messages)
	req.Messages = messages

	for i := range req.Messages {
		if req.Messages[i].Role != "system" {
			continue
		}
		if content, ok := req.Messages[i].Content.(string); ok && content != "" {
			req.Messages[i].Content = prompt + cavemanSeparator + content
		} else {
			req.Messages = insertMessage(req.Messages, i, providers.Message{Role: "system", Content: prompt})
		}
		return req
	}

	req.Messages = append([]providers.Message{{Role: "system", Content: prompt}}, req.Messages...)
	return req
}

func insertMessage(messages []providers.Message, index int, message providers.Message) []providers.Message {
	messages = append(messages, providers.Message{})
	copy(messages[index+1:], messages[index:])
	messages[index] = message
	return messages
}
