package translation

import "github.com/bloodf/g0router/internal/schemas"

// StripContentTypes removes image and audio content parts from messages.
// In Stage 1, Message.Content is always a plain string (no content-part arrays),
// so this function is a no-op. It exists so the call site is wired for Stage 2.
func StripContentTypes(req *schemas.ChatRequest, stripImages, stripAudio bool) {
	// Content-part arrays are not supported in Stage-1 schema; nothing to strip.
}
