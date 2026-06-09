// Package providers hosts the per-provider adapter tree. Each
// subdirectory is its own Go package and is responsible for the
// provider-specific request/response translation, streaming, and
// retry semantics:
//
//	openai/      — OpenAI (and any OpenAI-compatible endpoint) reference adapter
//	anthropic/   — Anthropic Claude
//	gemini/      — Google Gemini
//	groq/        — Groq
//	mistral/     — Mistral
//	cohere/      — Cohere
//	fireworks/   — Fireworks AI
//	together/    — Together AI
//	deepseek/    — DeepSeek
//	minimax/     — MiniMax
//	ollama/      — Ollama (local)
//	bedrock/     — AWS Bedrock
//	vertex/      — Google Vertex AI
//	utils/       — Shared fasthttp client, SSE scanner pools, common types
//
// Each provider package implements the adapter interface defined by
// the inference engine; this tree contains the implementations, not
// the contract.
package providers
