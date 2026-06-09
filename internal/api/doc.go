// Package api implements the OpenAI-compatible public surface
// (/v1/chat/completions, /v1/embeddings, /v1/models, etc.). Handlers
// here translate external wire formats into the internal inference
// pipeline and stream responses back as SSE.
package api
