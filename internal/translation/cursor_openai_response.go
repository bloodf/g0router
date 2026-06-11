package translation

// cursorToOpenAIResponse converts a Cursor response to OpenAI format.
// Since CursorExecutor already emits OpenAI-shaped chunks, this is a
// passthrough translator (similar to the Kiro pattern).
// Port of open-sse/translator/response/cursor-to-openai.js:13-28.
func cursorToOpenAIResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return nil, nil
	}

	// If chunk is already in OpenAI format (from executor transform), return as-is.
	if obj, ok := chunk["object"].(string); ok && obj == "chat.completion.chunk" {
		if _, ok := chunk["choices"]; ok {
			return []map[string]any{chunk}, nil
		}
	}

	// If chunk is a completion object (non-streaming), return as-is.
	if obj, ok := chunk["object"].(string); ok && obj == "chat.completion" {
		if _, ok := chunk["choices"]; ok {
			return []map[string]any{chunk}, nil
		}
	}

	// Fallback: return chunk as-is.
	return []map[string]any{chunk}, nil
}
