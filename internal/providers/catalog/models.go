package catalog

// ModelEntry is the Go port of a single entry in the reference
// PROVIDER_MODELS map (open-sse/config/providerModels.js).
// Type is stored verbatim from the ref (empty string when the ref has no
// type field) and is NOT defaulted to "llm" here — that normalization
// happens at read sites in later waves.
type ModelEntry struct {
	ID              string
	Name            string
	UpstreamModelID string
	Type            string
	Params          []string
}

// Models maps provider id → static model catalog. Each block is ported
// verbatim from providerModels.js (including Type and Params).
var Models = map[string][]ModelEntry{
	"deepseek": {
		{ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro", UpstreamModelID: "deepseek-v4-pro"},
		{ID: "deepseek-v4-pro-max", Name: "DeepSeek V4 Pro Max", UpstreamModelID: "deepseek-v4-pro"},
		{ID: "deepseek-v4-pro-none", Name: "DeepSeek V4 Pro No Thinking", UpstreamModelID: "deepseek-v4-pro"},
		{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash", UpstreamModelID: "deepseek-v4-flash"},
		{ID: "deepseek-chat", Name: "DeepSeek V3.2 Chat", UpstreamModelID: "deepseek-chat"},
		{ID: "deepseek-reasoner", Name: "DeepSeek V3.2 Reasoner", UpstreamModelID: "deepseek-reasoner"},
	},
	"groq": {
		{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B", UpstreamModelID: "llama-3.3-70b-versatile"},
		{ID: "meta-llama/llama-4-maverick-17b-128e-instruct", Name: "Llama 4 Maverick", UpstreamModelID: "meta-llama/llama-4-maverick-17b-128e-instruct"},
		{ID: "qwen/qwen3-32b", Name: "Qwen3 32B", UpstreamModelID: "qwen/qwen3-32b"},
		{ID: "openai/gpt-oss-120b", Name: "GPT-OSS 120B", UpstreamModelID: "openai/gpt-oss-120b"},
		// STT models
		{ID: "whisper-large-v3", Name: "Whisper Large v3", Type: "stt", Params: []string{"language", "response_format", "temperature", "prompt"}, UpstreamModelID: "whisper-large-v3"},
		{ID: "whisper-large-v3-turbo", Name: "Whisper Large v3 Turbo", Type: "stt", Params: []string{"language", "response_format", "temperature", "prompt"}, UpstreamModelID: "whisper-large-v3-turbo"},
		{ID: "distil-whisper-large-v3-en", Name: "Distil Whisper Large v3 EN", Type: "stt", Params: []string{"language", "response_format", "temperature", "prompt"}, UpstreamModelID: "distil-whisper-large-v3-en"},
	},
	"xai": {
		{ID: "grok-4", Name: "Grok 4", UpstreamModelID: "grok-4"},
		{ID: "grok-4-fast-reasoning", Name: "Grok 4 Fast Reasoning", UpstreamModelID: "grok-4-fast-reasoning"},
		{ID: "grok-code-fast-1", Name: "Grok Code Fast", UpstreamModelID: "grok-code-fast-1"},
		{ID: "grok-3", Name: "Grok 3", UpstreamModelID: "grok-3"},
		{ID: "grok-2-image-1212", Name: "Grok 2 Image", Type: "image", Params: []string{"n", "response_format"}, UpstreamModelID: "grok-2-image-1212"},
	},
	"mistral": {
		{ID: "mistral-large-latest", Name: "Mistral Large 3", UpstreamModelID: "mistral-large-latest"},
		{ID: "codestral-latest", Name: "Codestral", UpstreamModelID: "codestral-latest"},
		{ID: "mistral-medium-latest", Name: "Mistral Medium 3", UpstreamModelID: "mistral-medium-latest"},
		{ID: "mistral-embed", Name: "Mistral Embed", Type: "embedding", UpstreamModelID: "mistral-embed"},
	},
	"perplexity": {
		{ID: "sonar-pro", Name: "Sonar Pro", UpstreamModelID: "sonar-pro"},
		{ID: "sonar", Name: "Sonar", UpstreamModelID: "sonar"},
	},
	"together": {
		{ID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", Name: "Llama 3.3 70B Turbo", UpstreamModelID: "meta-llama/Llama-3.3-70B-Instruct-Turbo"},
		{ID: "deepseek-ai/DeepSeek-R1", Name: "DeepSeek R1", UpstreamModelID: "deepseek-ai/DeepSeek-R1"},
		{ID: "Qwen/Qwen3-235B-A22B", Name: "Qwen3 235B", UpstreamModelID: "Qwen/Qwen3-235B-A22B"},
		{ID: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8", Name: "Llama 4 Maverick", UpstreamModelID: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8"},
		{ID: "BAAI/bge-large-en-v1.5", Name: "BGE Large EN v1.5", Type: "embedding", UpstreamModelID: "BAAI/bge-large-en-v1.5"},
		{ID: "togethercomputer/m2-bert-80M-8k-retrieval", Name: "M2 BERT 80M 8K", Type: "embedding", UpstreamModelID: "togethercomputer/m2-bert-80M-8k-retrieval"},
	},
	"fireworks": {
		{ID: "accounts/fireworks/models/deepseek-v3p1", Name: "DeepSeek V3.1", UpstreamModelID: "accounts/fireworks/models/deepseek-v3p1"},
		{ID: "accounts/fireworks/models/llama-v3p3-70b-instruct", Name: "Llama 3.3 70B", UpstreamModelID: "accounts/fireworks/models/llama-v3p3-70b-instruct"},
		{ID: "accounts/fireworks/models/qwen3-235b-a22b", Name: "Qwen3 235B", UpstreamModelID: "accounts/fireworks/models/qwen3-235b-a22b"},
		{ID: "nomic-ai/nomic-embed-text-v1.5", Name: "Nomic Embed Text v1.5", Type: "embedding", UpstreamModelID: "nomic-ai/nomic-embed-text-v1.5"},
	},
	"cohere": {
		{ID: "command-r-plus-08-2024", Name: "Command R+ (Aug 2024)", UpstreamModelID: "command-r-plus-08-2024"},
		{ID: "command-r-08-2024", Name: "Command R (Aug 2024)", UpstreamModelID: "command-r-08-2024"},
		{ID: "command-a-03-2025", Name: "Command A (Mar 2025)", UpstreamModelID: "command-a-03-2025"},
	},
	"openrouter": {
		// Embedding models
		{ID: "openai/text-embedding-3-large", Name: "OpenAI Text Embedding 3 Large", Type: "embedding", UpstreamModelID: "openai/text-embedding-3-large"},
		{ID: "openai/text-embedding-3-small", Name: "OpenAI Text Embedding 3 Small", Type: "embedding", UpstreamModelID: "openai/text-embedding-3-small"},
		{ID: "openai/text-embedding-ada-002", Name: "OpenAI Text Embedding Ada 002", Type: "embedding", UpstreamModelID: "openai/text-embedding-ada-002"},
		{ID: "qwen/qwen3-embedding-8b", Name: "Qwen3 Embedding 8B", Type: "embedding", UpstreamModelID: "qwen/qwen3-embedding-8b"},
		{ID: "perplexity/pplx-embed-v1-4b", Name: "Perplexity Embed V1 4B", Type: "embedding", UpstreamModelID: "perplexity/pplx-embed-v1-4b"},
		{ID: "perplexity/pplx-embed-v1-0.6b", Name: "Perplexity Embed V1 0.6B", Type: "embedding", UpstreamModelID: "perplexity/pplx-embed-v1-0.6b"},
		{ID: "nvidia/llama-nemotron-embed-vl-1b-v2:free", Name: "NVIDIA Nemotron Embed VL 1B V2 (Free)", Type: "embedding", UpstreamModelID: "nvidia/llama-nemotron-embed-vl-1b-v2:free"},
		// TTS models
		{ID: "openai/gpt-4o-mini-tts", Name: "GPT-4o Mini TTS", Type: "tts", UpstreamModelID: "openai/gpt-4o-mini-tts"},
		{ID: "openai/tts-1-hd", Name: "TTS-1 HD", Type: "tts", UpstreamModelID: "openai/tts-1-hd"},
		{ID: "openai/tts-1", Name: "TTS-1", Type: "tts", UpstreamModelID: "openai/tts-1"},
		// Image models
		{ID: "openai/dall-e-3", Name: "DALL-E 3 (via OpenRouter)", Type: "image", Params: []string{"size", "quality", "style", "response_format"}, UpstreamModelID: "openai/dall-e-3"},
		{ID: "openai/gpt-image-1", Name: "GPT Image 1 (via OpenRouter)", Type: "image", Params: []string{"n", "size", "quality", "response_format"}, UpstreamModelID: "openai/gpt-image-1"},
		{ID: "google/imagen-3.0-generate-002", Name: "Imagen 3 (via OpenRouter)", Type: "image", Params: []string{"n", "size"}, UpstreamModelID: "google/imagen-3.0-generate-002"},
		{ID: "black-forest-labs/FLUX.1-schnell", Name: "FLUX.1 Schnell (via OpenRouter)", Type: "image", Params: []string{"n", "size"}, UpstreamModelID: "black-forest-labs/FLUX.1-schnell"},
	},
	"ollama": {
		{ID: "gpt-oss:120b", Name: "GPT OSS 120B", UpstreamModelID: "gpt-oss:120b"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5", UpstreamModelID: "kimi-k2.5"},
		{ID: "glm-5", Name: "GLM 5", UpstreamModelID: "glm-5"},
		{ID: "minimax-m2.5", Name: "MiniMax M2.5", UpstreamModelID: "minimax-m2.5"},
		{ID: "glm-4.7-flash", Name: "GLM 4.7 Flash", UpstreamModelID: "glm-4.7-flash"},
		{ID: "qwen3.5", Name: "Qwen3.5", UpstreamModelID: "qwen3.5"},
	},
}

// ModelsFor returns the static model catalog for the given provider.
// The returned slice is a copy to prevent external mutation.
func ModelsFor(provider string) []ModelEntry {
	models, ok := Models[provider]
	if !ok {
		return nil
	}
	out := make([]ModelEntry, len(models))
	copy(out, models)
	return out
}

// ResolveModel returns the ModelEntry for the given provider and model id.
func ResolveModel(provider, id string) (ModelEntry, bool) {
	for _, m := range Models[provider] {
		if m.ID == id {
			return m, true
		}
	}
	return ModelEntry{}, false
}
