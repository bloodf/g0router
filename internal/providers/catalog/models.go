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
	// --- w7-prov-openai: Free-tier openai model blocks (PAR-PROV-067) ---
	"aimlapi": {
		{ID: "gpt-4o", Name: "GPT-4o", UpstreamModelID: "gpt-4o"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", UpstreamModelID: "gpt-4o-mini"},
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", UpstreamModelID: "claude-3-5-sonnet-20241022"},
		{ID: "gemini-2.0-flash-exp", Name: "Gemini 2.0 Flash", UpstreamModelID: "gemini-2.0-flash-exp"},
		{ID: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", Name: "Llama 3.1 70B", UpstreamModelID: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"},
	},
	"novita": {
		{ID: "deepseek/deepseek-r1", Name: "DeepSeek R1", UpstreamModelID: "deepseek/deepseek-r1"},
		{ID: "deepseek/deepseek-v3", Name: "DeepSeek V3", UpstreamModelID: "deepseek/deepseek-v3"},
		{ID: "meta-llama/llama-3.3-70b-instruct", Name: "Llama 3.3 70B", UpstreamModelID: "meta-llama/llama-3.3-70b-instruct"},
		{ID: "qwen/qwen-2.5-72b-instruct", Name: "Qwen 2.5 72B", UpstreamModelID: "qwen/qwen-2.5-72b-instruct"},
	},
	"modal": {
		{ID: "auto", Name: "Auto (User-hosted)", UpstreamModelID: "auto"},
	},
	"reka": {
		{ID: "reka-flash-3", Name: "Reka Flash 3", UpstreamModelID: "reka-flash-3"},
		{ID: "reka-edge-2603", Name: "Reka Edge 2603", UpstreamModelID: "reka-edge-2603"},
	},
	"nlpcloud": {
		{ID: "chatdolphin", Name: "ChatDolphin", UpstreamModelID: "chatdolphin"},
		{ID: "dolphin", Name: "Dolphin", UpstreamModelID: "dolphin"},
		{ID: "finetuned-llama-3-70b", Name: "Llama 3 70B (Finetuned)", UpstreamModelID: "finetuned-llama-3-70b"},
	},
	"bazaarlink": {
		{ID: "auto:free", Name: "Auto Free (Zero Cost)", UpstreamModelID: "auto:free"},
		{ID: "auto", Name: "Auto (Best Model)", UpstreamModelID: "auto"},
	},
	"completions": {
		{ID: "claude-opus-4", Name: "Claude Opus 4", UpstreamModelID: "claude-opus-4"},
		{ID: "claude-sonnet-4", Name: "Claude Sonnet 4", UpstreamModelID: "claude-sonnet-4"},
		{ID: "gpt-4o", Name: "GPT-4o", UpstreamModelID: "gpt-4o"},
		{ID: "gemini-2.0-flash", Name: "Gemini 2.0 Flash", UpstreamModelID: "gemini-2.0-flash"},
	},
	"enally": {
		{ID: "gpt-4o", Name: "GPT-4o", UpstreamModelID: "gpt-4o"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", UpstreamModelID: "gpt-4o-mini"},
		{ID: "claude-3-5-sonnet", Name: "Claude 3.5 Sonnet", UpstreamModelID: "claude-3-5-sonnet"},
	},
	"freetheai": {
		{ID: "gpt-4o", Name: "GPT-4o", UpstreamModelID: "gpt-4o"},
		{ID: "claude-3-5-sonnet", Name: "Claude 3.5 Sonnet", UpstreamModelID: "claude-3-5-sonnet"},
		{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", UpstreamModelID: "gemini-1.5-pro"},
		{ID: "deepseek-chat", Name: "DeepSeek Chat", UpstreamModelID: "deepseek-chat"},
	},
	"llm7": {
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", UpstreamModelID: "gpt-4o-mini"},
		{ID: "gpt-4.1-mini", Name: "GPT-4.1 Mini", UpstreamModelID: "gpt-4.1-mini"},
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", UpstreamModelID: "gemini-1.5-flash"},
	},
	"lepton": {
		{ID: "llama3-1-405b", Name: "Llama 3.1 405B", UpstreamModelID: "llama3-1-405b"},
		{ID: "llama3-1-70b", Name: "Llama 3.1 70B", UpstreamModelID: "llama3-1-70b"},
		{ID: "llama3-1-8b", Name: "Llama 3.1 8B", UpstreamModelID: "llama3-1-8b"},
		{ID: "mixtral-8x7b", Name: "Mixtral 8x7B", UpstreamModelID: "mixtral-8x7b"},
	},
	"kluster": {
		{ID: "deepseek-ai/DeepSeek-R1", Name: "DeepSeek R1", UpstreamModelID: "deepseek-ai/DeepSeek-R1"},
		{ID: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8", Name: "Llama 4 Maverick", UpstreamModelID: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8"},
		{ID: "meta-llama/Llama-4-Scout-17B-16E-Instruct", Name: "Llama 4 Scout", UpstreamModelID: "meta-llama/Llama-4-Scout-17B-16E-Instruct"},
		{ID: "Qwen/Qwen3-235B-A22B-Instruct", Name: "Qwen3 235B", UpstreamModelID: "Qwen/Qwen3-235B-A22B-Instruct"},
	},
	"ai21": {
		{ID: "jamba-large", Name: "Jamba 1.5 Large", UpstreamModelID: "jamba-large"},
		{ID: "jamba-mini", Name: "Jamba 1.5 Mini", UpstreamModelID: "jamba-mini"},
	},
	"inference-net": {
		{ID: "meta-llama/llama-3.3-70b-instruct/fp-16", Name: "Llama 3.3 70B", UpstreamModelID: "meta-llama/llama-3.3-70b-instruct/fp-16"},
		{ID: "deepseek/deepseek-v3-0324", Name: "DeepSeek V3", UpstreamModelID: "deepseek/deepseek-v3-0324"},
		{ID: "mistralai/mistral-nemo-12b-instruct/fp-16", Name: "Mistral Nemo 12B", UpstreamModelID: "mistralai/mistral-nemo-12b-instruct/fp-16"},
	},
	"predibase": {
		{ID: "llama-3-2-3b-instruct", Name: "Llama 3.2 3B", UpstreamModelID: "llama-3-2-3b-instruct"},
		{ID: "llama-3-1-8b-instruct", Name: "Llama 3.1 8B", UpstreamModelID: "llama-3-1-8b-instruct"},
		{ID: "qwen2-5-7b-instruct", Name: "Qwen 2.5 7B", UpstreamModelID: "qwen2-5-7b-instruct"},
	},
	"bytez": {
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B", UpstreamModelID: "meta-llama/Llama-3.3-70B-Instruct"},
		{ID: "mistralai/Mistral-7B-Instruct-v0.3", Name: "Mistral 7B v0.3", UpstreamModelID: "mistralai/Mistral-7B-Instruct-v0.3"},
		{ID: "Qwen/Qwen2.5-72B-Instruct", Name: "Qwen 2.5 72B", UpstreamModelID: "Qwen/Qwen2.5-72B-Instruct"},
	},
	"morph": {
		{ID: "morph-v3-large", Name: "Morph V3 Large", UpstreamModelID: "morph-v3-large"},
		{ID: "morph-v3-fast", Name: "Morph V3 Fast", UpstreamModelID: "morph-v3-fast"},
	},
	"longcat": {
		{ID: "LongCat-Flash-Chat", Name: "LongCat Flash Chat", UpstreamModelID: "LongCat-Flash-Chat"},
		{ID: "LongCat-Flash-Thinking", Name: "LongCat Flash Thinking", UpstreamModelID: "LongCat-Flash-Thinking"},
		{ID: "LongCat-Flash-Lite", Name: "LongCat Flash Lite", UpstreamModelID: "LongCat-Flash-Lite"},
	},
	"puter": {
		{ID: "gpt-5", Name: "GPT-5", UpstreamModelID: "gpt-5"},
		{ID: "claude-opus-4", Name: "Claude Opus 4", UpstreamModelID: "claude-opus-4"},
		{ID: "gemini-3-pro-preview", Name: "Gemini 3 Pro", UpstreamModelID: "gemini-3-pro-preview"},
		{ID: "grok-4", Name: "Grok 4", UpstreamModelID: "grok-4"},
		{ID: "deepseek-chat", Name: "DeepSeek V3", UpstreamModelID: "deepseek-chat"},
	},
	"uncloseai": {
		{ID: "auto", Name: "Auto (Free)", UpstreamModelID: "auto"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", UpstreamModelID: "gpt-4o-mini"},
	},
	"scaleway": {
		{ID: "qwen3-235b-a22b-instruct-2507", Name: "Qwen3 235B", UpstreamModelID: "qwen3-235b-a22b-instruct-2507"},
		{ID: "llama-3.3-70b-instruct", Name: "Llama 3.3 70B", UpstreamModelID: "llama-3.3-70b-instruct"},
		{ID: "mistral-small-3.1-24b-instruct-2503", Name: "Mistral Small 3.1", UpstreamModelID: "mistral-small-3.1-24b-instruct-2503"},
	},
	"deepinfra": {
		{ID: "meta-llama/Meta-Llama-3.1-70B-Instruct", Name: "Llama 3.1 70B", UpstreamModelID: "meta-llama/Meta-Llama-3.1-70B-Instruct"},
		{ID: "deepseek-ai/DeepSeek-V3", Name: "DeepSeek V3", UpstreamModelID: "deepseek-ai/DeepSeek-V3"},
		{ID: "Qwen/Qwen2.5-72B-Instruct", Name: "Qwen 2.5 72B", UpstreamModelID: "Qwen/Qwen2.5-72B-Instruct"},
	},
	"sambanova": {
		{ID: "Meta-Llama-3.1-405B-Instruct", Name: "Llama 3.1 405B", UpstreamModelID: "Meta-Llama-3.1-405B-Instruct"},
		{ID: "Meta-Llama-3.1-70B-Instruct", Name: "Llama 3.1 70B", UpstreamModelID: "Meta-Llama-3.1-70B-Instruct"},
		{ID: "Meta-Llama-3.1-8B-Instruct", Name: "Llama 3.1 8B", UpstreamModelID: "Meta-Llama-3.1-8B-Instruct"},
	},
	"nscale": {
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B", UpstreamModelID: "meta-llama/Llama-3.3-70B-Instruct"},
		{ID: "Qwen/Qwen2.5-Coder-32B-Instruct", Name: "Qwen 2.5 Coder 32B", UpstreamModelID: "Qwen/Qwen2.5-Coder-32B-Instruct"},
	},
	"baseten": {
		{ID: "deepseek-ai/DeepSeek-R1", Name: "DeepSeek R1", UpstreamModelID: "deepseek-ai/DeepSeek-R1"},
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B", UpstreamModelID: "meta-llama/Llama-3.3-70B-Instruct"},
	},
	"publicai": {
		{ID: "auto", Name: "Auto (Community)", UpstreamModelID: "auto"},
	},
	"nous-research": {
		{ID: "Hermes-4-405B", Name: "Hermes 4 405B", UpstreamModelID: "Hermes-4-405B"},
		{ID: "Hermes-4-70B", Name: "Hermes 4 70B", UpstreamModelID: "Hermes-4-70B"},
	},
	"glhf": {
		{ID: "hf:meta-llama/Meta-Llama-3.1-405B-Instruct", Name: "Llama 3.1 405B", UpstreamModelID: "hf:meta-llama/Meta-Llama-3.1-405B-Instruct"},
		{ID: "hf:meta-llama/Meta-Llama-3.1-70B-Instruct", Name: "Llama 3.1 70B", UpstreamModelID: "hf:meta-llama/Meta-Llama-3.1-70B-Instruct"},
		{ID: "hf:Qwen/Qwen2.5-72B-Instruct", Name: "Qwen 2.5 72B", UpstreamModelID: "hf:Qwen/Qwen2.5-72B-Instruct"},
	},
	// --- w7-prov-openai: Western openai-format model blocks ---
	"nvidia": {
		{ID: "minimaxai/minimax-m2.7", Name: "Minimax M2.7", UpstreamModelID: "minimaxai/minimax-m2.7"},
		{ID: "z-ai/glm4.7", Name: "GLM 4.7", UpstreamModelID: "z-ai/glm4.7"},
		{ID: "nvidia/nv-embedqa-e5-v5", Name: "NV EmbedQA E5 v5", Type: "embedding", UpstreamModelID: "nvidia/nv-embedqa-e5-v5"},
		{ID: "nvidia/parakeet-ctc-1.1b-asr", Name: "Parakeet CTC 1.1B", Type: "stt", Params: []string{"language"}, UpstreamModelID: "nvidia/parakeet-ctc-1.1b-asr"},
	},
	"cerebras": {
		{ID: "gpt-oss-120b", Name: "GPT OSS 120B", UpstreamModelID: "gpt-oss-120b"},
		{ID: "zai-glm-4.7", Name: "ZAI GLM 4.7", UpstreamModelID: "zai-glm-4.7"},
		{ID: "llama-3.3-70b", Name: "Llama 3.3 70B", UpstreamModelID: "llama-3.3-70b"},
		{ID: "llama-4-scout-17b-16e-instruct", Name: "Llama 4 Scout", UpstreamModelID: "llama-4-scout-17b-16e-instruct"},
		{ID: "qwen-3-235b-a22b-instruct-2507", Name: "Qwen3 235B A22B", UpstreamModelID: "qwen-3-235b-a22b-instruct-2507"},
		{ID: "qwen-3-32b", Name: "Qwen3 32B", UpstreamModelID: "qwen-3-32b"},
	},
	"nebius": {
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B Instruct", UpstreamModelID: "meta-llama/Llama-3.3-70B-Instruct"},
		{ID: "Qwen/Qwen3-Embedding-8B", Name: "Qwen3 Embedding 8B", Type: "embedding", UpstreamModelID: "Qwen/Qwen3-Embedding-8B"},
	},
	"siliconflow": {
		{ID: "deepseek-ai/DeepSeek-V3.2", Name: "DeepSeek V3.2", UpstreamModelID: "deepseek-ai/DeepSeek-V3.2"},
		{ID: "deepseek-ai/DeepSeek-V3.1", Name: "DeepSeek V3.1", UpstreamModelID: "deepseek-ai/DeepSeek-V3.1"},
		{ID: "deepseek-ai/DeepSeek-R1", Name: "DeepSeek R1", UpstreamModelID: "deepseek-ai/DeepSeek-R1"},
		{ID: "Qwen/Qwen3-235B-A22B-Instruct-2507", Name: "Qwen3 235B", UpstreamModelID: "Qwen/Qwen3-235B-A22B-Instruct-2507"},
		{ID: "Qwen/Qwen3-Coder-480B-A35B-Instruct", Name: "Qwen3 Coder 480B", UpstreamModelID: "Qwen/Qwen3-Coder-480B-A35B-Instruct"},
		{ID: "Qwen/Qwen3-32B", Name: "Qwen3 32B", UpstreamModelID: "Qwen/Qwen3-32B"},
		{ID: "moonshotai/Kimi-K2.5", Name: "Kimi K2.5", UpstreamModelID: "moonshotai/Kimi-K2.5"},
		{ID: "zai-org/GLM-4.7", Name: "GLM 4.7", UpstreamModelID: "zai-org/GLM-4.7"},
		{ID: "openai/gpt-oss-120b", Name: "GPT OSS 120B", UpstreamModelID: "openai/gpt-oss-120b"},
		{ID: "baidu/ERNIE-4.5-300B-A47B", Name: "ERNIE 4.5 300B", UpstreamModelID: "baidu/ERNIE-4.5-300B-A47B"},
	},
	"hyperbolic": {
		{ID: "Qwen/QwQ-32B", Name: "QwQ 32B", UpstreamModelID: "Qwen/QwQ-32B"},
		{ID: "deepseek-ai/DeepSeek-R1", Name: "DeepSeek R1", UpstreamModelID: "deepseek-ai/DeepSeek-R1"},
		{ID: "deepseek-ai/DeepSeek-V3", Name: "DeepSeek V3", UpstreamModelID: "deepseek-ai/DeepSeek-V3"},
		{ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B", UpstreamModelID: "meta-llama/Llama-3.3-70B-Instruct"},
		{ID: "meta-llama/Llama-3.2-3B-Instruct", Name: "Llama 3.2 3B", UpstreamModelID: "meta-llama/Llama-3.2-3B-Instruct"},
		{ID: "Qwen/Qwen2.5-72B-Instruct", Name: "Qwen 2.5 72B", UpstreamModelID: "Qwen/Qwen2.5-72B-Instruct"},
		{ID: "Qwen/Qwen2.5-Coder-32B-Instruct", Name: "Qwen 2.5 Coder 32B", UpstreamModelID: "Qwen/Qwen2.5-Coder-32B-Instruct"},
		{ID: "NousResearch/Hermes-3-Llama-3.1-70B", Name: "Hermes 3 70B", UpstreamModelID: "NousResearch/Hermes-3-Llama-3.1-70B"},
	},
	"blackbox": {
		{ID: "gpt-4o", Name: "GPT-4o", UpstreamModelID: "gpt-4o"},
		{ID: "gpt-4o-mini", Name: "GPT-4o mini", UpstreamModelID: "gpt-4o-mini"},
		{ID: "claude-sonnet-4.6", Name: "Claude Sonnet 4.6", UpstreamModelID: "claude-sonnet-4.6"},
		{ID: "claude-sonnet-4.5", Name: "Claude Sonnet 4.5", UpstreamModelID: "claude-sonnet-4.5"},
		{ID: "claude-opus-4.6", Name: "Claude Opus 4.6", UpstreamModelID: "claude-opus-4.6"},
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6 (Legacy)", UpstreamModelID: "claude-sonnet-4-6"},
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6 (Legacy)", UpstreamModelID: "claude-opus-4-6"},
		{ID: "deepseek-chat", Name: "DeepSeek Chat", UpstreamModelID: "deepseek-chat"},
		{ID: "deepseek-v3-671b", Name: "DeepSeek V3 671B", UpstreamModelID: "deepseek-v3-671b"},
		{ID: "deepseek-r1", Name: "DeepSeek R1", UpstreamModelID: "deepseek-r1"},
		{ID: "o1", Name: "OpenAI o1", UpstreamModelID: "o1"},
		{ID: "o3-mini", Name: "OpenAI o3-mini", UpstreamModelID: "o3-mini"},
		{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", UpstreamModelID: "gemini-2.5-flash"},
		{ID: "gemini-3-flash-preview", Name: "Gemini 3 Flash Preview", UpstreamModelID: "gemini-3-flash-preview"},
		{ID: "qwen3-coder-plus", Name: "Qwen3 Coder Plus", UpstreamModelID: "qwen3-coder-plus"},
		{ID: "qwen3-max", Name: "Qwen3 Max", UpstreamModelID: "qwen3-max"},
		{ID: "qwen3-vl-plus", Name: "Qwen3 VL Plus", UpstreamModelID: "qwen3-vl-plus"},
	},
	// gitlab, codebuddy, vercel-ai-gateway, chutes: no static model block in ref (ESC-6).
	// --- w7-prov-openai: Chinese openai-format model blocks ---
	"glm-cn": {
		{ID: "glm-5.1", Name: "GLM 5.1", UpstreamModelID: "glm-5.1"},
		{ID: "glm-5", Name: "GLM 5", UpstreamModelID: "glm-5"},
		{ID: "glm-4.7", Name: "GLM-4.7", UpstreamModelID: "glm-4.7"},
		{ID: "glm-4.6", Name: "GLM-4.6", UpstreamModelID: "glm-4.6"},
		{ID: "glm-4.5-air", Name: "GLM-4.5-Air", UpstreamModelID: "glm-4.5-air"},
	},
	"alicode": {
		{ID: "qwen3.5-plus", Name: "Qwen3.5 Plus", UpstreamModelID: "qwen3.5-plus"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5", UpstreamModelID: "kimi-k2.5"},
		{ID: "glm-5", Name: "GLM 5", UpstreamModelID: "glm-5"},
		{ID: "MiniMax-M2.5", Name: "MiniMax M2.5", UpstreamModelID: "MiniMax-M2.5"},
		{ID: "qwen3-max-2026-01-23", Name: "Qwen3 Max", UpstreamModelID: "qwen3-max-2026-01-23"},
		{ID: "qwen3-coder-next", Name: "Qwen3 Coder Next", UpstreamModelID: "qwen3-coder-next"},
		{ID: "qwen3-coder-plus", Name: "Qwen3 Coder Plus", UpstreamModelID: "qwen3-coder-plus"},
		{ID: "glm-4.7", Name: "GLM 4.7", UpstreamModelID: "glm-4.7"},
	},
	"alicode-intl": {
		{ID: "qwen3.5-plus", Name: "Qwen3.5 Plus", UpstreamModelID: "qwen3.5-plus"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5", UpstreamModelID: "kimi-k2.5"},
		{ID: "glm-5", Name: "GLM 5", UpstreamModelID: "glm-5"},
		{ID: "MiniMax-M2.5", Name: "MiniMax M2.5", UpstreamModelID: "MiniMax-M2.5"},
		{ID: "qwen3-coder-next", Name: "Qwen3 Coder Next", UpstreamModelID: "qwen3-coder-next"},
		{ID: "qwen3-coder-plus", Name: "Qwen3 Coder Plus", UpstreamModelID: "qwen3-coder-plus"},
		{ID: "glm-4.7", Name: "GLM 4.7", UpstreamModelID: "glm-4.7"},
	},
	"volcengine-ark": {
		{ID: "Doubao-Seed-2.0-Code", Name: "Doubao-Seed-2.0-Code", UpstreamModelID: "Doubao-Seed-2.0-Code"},
		{ID: "Doubao-Seed-2.0-pro", Name: "Doubao-Seed-2.0-pro", UpstreamModelID: "Doubao-Seed-2.0-pro"},
		{ID: "Doubao-Seed-2.0-lite", Name: "Doubao-Seed-2.0-lite", UpstreamModelID: "Doubao-Seed-2.0-lite"},
		{ID: "Doubao-Seed-Code", Name: "Doubao-Seed-Code", UpstreamModelID: "Doubao-Seed-Code"},
		{ID: "DeepSeek-V4-Flash", Name: "DeepSeek-V4-Flash", UpstreamModelID: "DeepSeek-V4-Flash"},
		{ID: "DeepSeek-V4-Pro", Name: "DeepSeek-V4-Pro", UpstreamModelID: "DeepSeek-V4-Pro"},
		{ID: "GLM-5.1", Name: "GLM-5.1", UpstreamModelID: "GLM-5.1"},
		{ID: "MiniMax-M2.7", Name: "MiniMax-M2.7", UpstreamModelID: "MiniMax-M2.7"},
		{ID: "Kimi-K2.6", Name: "Kimi-K2.6", UpstreamModelID: "Kimi-K2.6"},
	},
	"byteplus": {
		{ID: "seed-2-0-pro-260328", Name: "Seed 2.0 Pro", UpstreamModelID: "seed-2-0-pro-260328"},
		{ID: "seed-2-0-code-preview-260328", Name: "Seed 2.0 Code Preview", UpstreamModelID: "seed-2-0-code-preview-260328"},
		{ID: "seed-2-0-mini-260215", Name: "Seed 2.0 Mini", UpstreamModelID: "seed-2-0-mini-260215"},
		{ID: "seed-2-0-lite-260228", Name: "Seed 2.0 Lite", UpstreamModelID: "seed-2-0-lite-260228"},
		{ID: "kimi-k2-thinking-251104", Name: "Kimi K2 Thinking", UpstreamModelID: "kimi-k2-thinking-251104"},
		{ID: "glm-4-7-251222", Name: "GLM 4.7", UpstreamModelID: "glm-4-7-251222"},
		{ID: "gpt-oss-120b-250805", Name: "GPT-OSS-120B", UpstreamModelID: "gpt-oss-120b-250805"},
	},
	"xiaomi-mimo": {
		{ID: "mimo-v2.5-pro", Name: "MiMo V2.5 Pro", UpstreamModelID: "mimo-v2.5-pro"},
		{ID: "mimo-v2.5", Name: "MiMo V2.5", UpstreamModelID: "mimo-v2.5"},
		{ID: "mimo-v2-omni", Name: "MiMo V2 Omni", UpstreamModelID: "mimo-v2-omni"},
		{ID: "mimo-v2-flash", Name: "MiMo V2 Flash", UpstreamModelID: "mimo-v2-flash"},
	},
	"opencode-go": {
		{ID: "kimi-k2.6", Name: "Kimi K2.6", UpstreamModelID: "kimi-k2.6"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5", UpstreamModelID: "kimi-k2.5"},
		{ID: "glm-5.1", Name: "GLM 5.1", UpstreamModelID: "glm-5.1"},
		{ID: "glm-5", Name: "GLM 5", UpstreamModelID: "glm-5"},
		{ID: "qwen3.5-plus", Name: "Qwen 3.5 Plus", UpstreamModelID: "qwen3.5-plus"},
		{ID: "qwen3.6-plus", Name: "Qwen 3.6 Plus", UpstreamModelID: "qwen3.6-plus"},
		{ID: "mimo-v2-pro", Name: "MiMo V2 Pro", UpstreamModelID: "mimo-v2-pro"},
		{ID: "mimo-v2-omni", Name: "MiMo V2 Omni", UpstreamModelID: "mimo-v2-omni"},
		// minimax-m2.7/m2.5 carry targetFormat:"claude" in the ref; ModelEntry has
		// no TargetFormat field (ESC-5, no struct change) — ID/Name ported verbatim.
		{ID: "minimax-m2.7", Name: "MiniMax M2.7", UpstreamModelID: "minimax-m2.7"},
		{ID: "minimax-m2.5", Name: "MiniMax M2.5", UpstreamModelID: "minimax-m2.5"},
	},
	"ollama": {
		{ID: "gpt-oss:120b", Name: "GPT OSS 120B", UpstreamModelID: "gpt-oss:120b"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5", UpstreamModelID: "kimi-k2.5"},
		{ID: "glm-5", Name: "GLM 5", UpstreamModelID: "glm-5"},
		{ID: "minimax-m2.5", Name: "MiniMax M2.5", UpstreamModelID: "minimax-m2.5"},
		{ID: "glm-4.7-flash", Name: "GLM 4.7 Flash", UpstreamModelID: "glm-4.7-flash"},
		{ID: "qwen3.5", Name: "Qwen3.5", UpstreamModelID: "qwen3.5"},
	},
	// --- w7-prov-special-a: claude-format provider models (providerModels.js
	// @827e5c3). minimax/minimax-cn share the minimax block. The ref
	// targetFormat:"claude" hint on MiniMax-M3 has no ModelEntry field (the
	// adapter is already claude-wire) and is not ported.
	"glm": {
		{ID: "glm-5.1", Name: "GLM 5.1"},
		{ID: "glm-5", Name: "GLM 5"},
		{ID: "glm-4.7", Name: "GLM 4.7"},
		{ID: "glm-4.6v", Name: "GLM 4.6V (Vision)"},
	},
	"kimi": {
		{ID: "kimi-k2.6", Name: "Kimi K2.6"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5"},
		{ID: "kimi-k2.5-thinking", Name: "Kimi K2.5 Thinking"},
		{ID: "kimi-latest", Name: "Kimi Latest"},
	},
	"minimax": {
		{ID: "MiniMax-M3", Name: "MiniMax M3"},
		{ID: "MiniMax-M2.7", Name: "MiniMax M2.7"},
		{ID: "MiniMax-M2.5", Name: "MiniMax M2.5"},
		{ID: "MiniMax-M2.1", Name: "MiniMax M2.1"},
		{ID: "minimax-image-01", Name: "MiniMax Image 01", Type: "image", Params: []string{"n", "size", "response_format"}},
	},
	// cloudflare-ai model block (providerModels.js @827e5c3). image models port
	// Type/Params; the ref `capabilities` hint has no ModelEntry field.
	"cloudflare-ai": {
		{ID: "@cf/meta/llama-3.2-1b-instruct", Name: "Llama 3.2 1B Instruct"},
		{ID: "@cf/meta/llama-3.2-3b-instruct", Name: "Llama 3.2 3B Instruct"},
		{ID: "@cf/meta/llama-3.1-8b-instruct-fp8-fast", Name: "Llama 3.1 8B Instruct FP8 Fast"},
		{ID: "@cf/meta/llama-3.1-8b-instruct-awq", Name: "Llama 3.1 8B Instruct AWQ"},
		{ID: "@cf/mistralai/mistral-small-3.1-24b-instruct", Name: "Mistral Small 3.1 24B Instruct"},
		{ID: "@cf/meta/llama-3.1-70b-instruct-fp8-fast", Name: "Llama 3.1 70B Instruct FP8 Fast"},
		{ID: "@cf/meta/llama-3.3-70b-instruct-fp8-fast", Name: "Llama 3.3 70B Instruct FP8 Fast"},
		{ID: "@cf/deepseek-ai/deepseek-r1-distill-qwen-32b", Name: "DeepSeek R1 Distill Qwen 32B"},
		{ID: "@cf/moonshotai/kimi-k2.5", Name: "Kimi K2.5"},
		{ID: "@cf/moonshotai/kimi-k2.6", Name: "Kimi K2.6"},
		{ID: "@cf/zai-org/glm-4.7-flash", Name: "GLM 4.7 Flash"},
		{ID: "@cf/qwen/qwq-32b", Name: "QwQ 32B"},
		{ID: "@cf/qwen/qwen2.5-coder-32b-instruct", Name: "Qwen 2.5 Coder 32B Instruct"},
		{ID: "@cf/black-forest-labs/flux-2-klein-9b", Name: "FLUX.2 Klein 9B", Type: "image", Params: []string{"size"}},
		{ID: "@cf/black-forest-labs/flux-2-klein-4b", Name: "FLUX.2 Klein 4B", Type: "image", Params: []string{"size"}},
		{ID: "@cf/black-forest-labs/flux-2-dev", Name: "FLUX.2 Dev", Type: "image", Params: []string{"size"}},
		{ID: "@cf/leonardo/lucid-origin", Name: "Lucid Origin", Type: "image", Params: []string{"size"}},
		{ID: "@cf/leonardo/phoenix-1.0", Name: "Phoenix 1.0", Type: "image", Params: []string{"size"}},
		{ID: "@cf/black-forest-labs/flux-1-schnell", Name: "FLUX.1 Schnell", Type: "image", Params: []string{"size"}},
		{ID: "@cf/bytedance/stable-diffusion-xl-lightning", Name: "SDXL Lightning", Type: "image", Params: []string{"size"}},
		{ID: "@cf/lykon/dreamshaper-8-lcm", Name: "DreamShaper 8 LCM", Type: "image", Params: []string{"size"}},
		{ID: "@cf/runwayml/stable-diffusion-v1-5-img2img", Name: "Stable Diffusion v1.5 Img2Img", Type: "image", Params: []string{"size"}},
		{ID: "@cf/runwayml/stable-diffusion-v1-5-inpainting", Name: "Stable Diffusion v1.5 Inpainting", Type: "image", Params: []string{"size"}},
		{ID: "@cf/stabilityai/stable-diffusion-xl-base-1.0", Name: "SDXL Base 1.0", Type: "image", Params: []string{"size"}},
	},
	// xiaomi-tokenplan model block (providerModels.js @827e5c3). The
	// mimo-v2.5-pro-claude targetFormat:"claude" + tts/voice variants are noted
	// in ESC-A2; they port here as openai/media entries (Type from ref empty).
	"xiaomi-tokenplan": {
		{ID: "mimo-v2.5-pro", Name: "MiMo V2.5 Pro"},
		{ID: "mimo-v2.5-pro-claude", Name: "MiMo V2.5 Pro (Claude Native)", UpstreamModelID: "mimo-v2.5-pro"},
		{ID: "mimo-v2.5", Name: "MiMo V2.5"},
		{ID: "mimo-v2-pro", Name: "MiMo V2 Pro"},
		{ID: "mimo-v2-omni", Name: "MiMo V2 Omni"},
		{ID: "mimo-v2-tts", Name: "MiMo V2 TTS"},
		{ID: "mimo-v2.5-tts", Name: "MiMo V2.5 TTS"},
		{ID: "mimo-v2.5-tts-voiceclone", Name: "MiMo V2.5 TTS Voice Clone"},
		{ID: "mimo-v2.5-tts-voicedesign", Name: "MiMo V2.5 TTS Voice Design"},
	},
	// vertex partner model block (providerModels.js "vertex-partner" @827e5c3).
	// Registered under "vertex" because this plan ships the partner-openai path
	// (native gemini-on-vertex deferred — ESC-A1).
	"vertex": {
		{ID: "deepseek-ai/deepseek-v3.2-maas", Name: "DeepSeek V3.2 (Vertex)"},
		{ID: "qwen/qwen3-next-80b-a3b-thinking-maas", Name: "Qwen3 Next 80B Thinking (Vertex)"},
		{ID: "qwen/qwen3-next-80b-a3b-instruct-maas", Name: "Qwen3 Next 80B Instruct (Vertex)"},
		{ID: "zai-org/glm-5-maas", Name: "GLM-5 (Vertex)"},
	},
	// commandcode model block (providerModels.js @827e5c3).
	"commandcode": {
		{ID: "deepseek/deepseek-v4-pro", Name: "DeepSeek V4 Pro"},
		{ID: "deepseek/deepseek-v4-flash", Name: "DeepSeek V4 Flash"},
		{ID: "moonshotai/Kimi-K2.6", Name: "Kimi K2.6"},
		{ID: "moonshotai/Kimi-K2.5", Name: "Kimi K2.5"},
		{ID: "zai-org/GLM-5.1", Name: "GLM 5.1"},
		{ID: "zai-org/GLM-5", Name: "GLM 5"},
		{ID: "MiniMaxAI/MiniMax-M2.7", Name: "MiniMax M2.7"},
		{ID: "MiniMaxAI/MiniMax-M2.5", Name: "MiniMax M2.5"},
		{ID: "Qwen/Qwen3.6-Max-Preview", Name: "Qwen 3.6 Max Preview"},
		{ID: "Qwen/Qwen3.6-Plus", Name: "Qwen 3.6 Plus"},
		{ID: "stepfun/Step-3.5-Flash", Name: "Step 3.5 Flash"},
	},
	// w7-prov-special-b: kiro AWS-eventstream provider (PAR-PROV-022).
	// Verbatim from providerModels.js:127-146. The commented-out claude-opus-4.5
	// entry is excluded (it is commented out in the ref). The ref `strip` lists
	// are a read-site concern and are not part of the static catalog (ModelEntry
	// has no Strip field).
	"kiro": {
		{ID: "claude-sonnet-4.5", Name: "Claude Sonnet 4.5"},
		{ID: "claude-haiku-4.5", Name: "Claude Haiku 4.5"},
		{ID: "deepseek-3.2", Name: "DeepSeek 3.2"},
		{ID: "qwen3-coder-next", Name: "Qwen3 Coder Next"},
		{ID: "glm-5", Name: "GLM 5"},
		{ID: "MiniMax-M2.5", Name: "MiniMax M2.5"},
		{ID: "claude-sonnet-4.5-thinking", Name: "Claude Sonnet 4.5 (Thinking)"},
		{ID: "claude-haiku-4.5-thinking", Name: "Claude Haiku 4.5 (Thinking)"},
		{ID: "claude-sonnet-4.5-agentic", Name: "Claude Sonnet 4.5 (Agentic)"},
		{ID: "claude-haiku-4.5-agentic", Name: "Claude Haiku 4.5 (Agentic)"},
		{ID: "claude-sonnet-4.5-thinking-agentic", Name: "Claude Sonnet 4.5 (Thinking + Agentic)"},
		{ID: "claude-haiku-4.5-thinking-agentic", Name: "Claude Haiku 4.5 (Thinking + Agentic)"},
	},
	// w7-prov-special-b: antigravity multi-backend provider (PAR-PROV-020).
	// Verbatim from providerModels.js:84-94 — one provider id fronting the
	// gemini / claude / gpt-oss backends (selected per model by the executor).
	"antigravity": {
		{ID: "gemini-3-flash-agent", Name: "Gemini 3.5 Flash (High)"},
		{ID: "gemini-3.5-flash-low", Name: "Gemini 3.5 Flash (Medium)"},
		{ID: "gemini-3.5-flash-extra-low", Name: "Gemini 3.5 Flash (Low)"},
		{ID: "gemini-pro-agent", Name: "Gemini 3.1 Pro (High)"},
		{ID: "gemini-3.1-pro-low", Name: "Gemini 3.1 Pro (Low)"},
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6 (Thinking)"},
		{ID: "claude-opus-4-6-thinking", Name: "Claude Opus 4.6 (Thinking)"},
		{ID: "gpt-oss-120b-medium", Name: "GPT-OSS 120B (Medium)"},
		{ID: "gemini-3-flash", Name: "Gemini 3 Flash"},
	},
	// w7-prov-special-b: cursor connect+protobuf provider (PAR-PROV-023).
	// Verbatim from providerModels.js:163-178.
	"cursor": {
		{ID: "default", Name: "Auto (Server Picks)"},
		{ID: "claude-4.5-opus-high-thinking", Name: "Claude 4.5 Opus High Thinking"},
		{ID: "claude-4.5-opus-high", Name: "Claude 4.5 Opus High"},
		{ID: "claude-4.5-sonnet-thinking", Name: "Claude 4.5 Sonnet Thinking"},
		{ID: "claude-4.5-sonnet", Name: "Claude 4.5 Sonnet"},
		{ID: "claude-4.5-haiku", Name: "Claude 4.5 Haiku"},
		{ID: "claude-4.5-opus", Name: "Claude 4.5 Opus"},
		{ID: "gpt-5.2-codex", Name: "GPT 5.2 Codex"},
		{ID: "claude-4.6-opus-max", Name: "Claude 4.6 Opus Max"},
		{ID: "claude-4.6-sonnet-medium-thinking", Name: "Claude 4.6 Sonnet Medium Thinking"},
		{ID: "kimi-k2.5", Name: "Kimi K2.5"},
		{ID: "gemini-3-flash-preview", Name: "Gemini 3 Flash Preview"},
		{ID: "gpt-5.2", Name: "GPT 5.2"},
		{ID: "gpt-5.3-codex", Name: "GPT 5.3 Codex"},
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
