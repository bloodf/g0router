package translation

import (
	"fmt"
	"time"
)

// formatIncompleteResponsesStreamFailure synthesizes a response.failed SSE frame
// for streams that close without reaching a terminal event. Port of
// responsesStreamHelpers.js:33-52.
func formatIncompleteResponsesStreamFailure() []byte {
	return FormatSSE(FormatOpenAIResponses, map[string]any{
		"event": "response.failed",
		"data": map[string]any{
			"type": "response.failed",
			"response": map[string]any{
				"id":     fmt.Sprintf("resp_%d", time.Now().UnixMilli()),
				"status": "failed",
				"error": map[string]any{
					"type":    "stream_error",
					"code":    "stream_disconnected",
					"message": "stream closed before response.completed",
				},
			},
		},
	})
}

// isResponsesTerminalEvent reports whether a chunk signals the end of a
// Responses API stream. Port of responsesStreamHelpers.js:18-23.
// Terminal if: event type in {response.completed, response.failed} OR
// chunk.response.status in {completed, failed}.
func isResponsesTerminalEvent(chunk map[string]any) bool {
	if eventName, ok := chunk["event"].(string); ok {
		if eventName == "response.completed" || eventName == "response.failed" {
			return true
		}
	}
	if typ, ok := chunk["type"].(string); ok {
		if typ == "response.completed" || typ == "response.failed" {
			return true
		}
	}
	if respRaw, ok := chunk["response"]; ok {
		if resp, ok := respRaw.(map[string]any); ok {
			if status, ok := resp["status"].(string); ok {
				return status == "completed" || status == "failed"
			}
		}
	}
	return false
}
