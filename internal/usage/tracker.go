package usage

import "github.com/bloodf/g0router/internal/providers"

type Usage struct {
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	CacheReadTokens int
	ReasoningTokens int
}

func FromChatResponse(resp providers.ChatResponse) (Usage, bool) {
	return fromProviderUsage(resp.Usage)
}

func FromStreamChunk(chunk providers.StreamChunk) (Usage, bool) {
	return fromProviderUsage(chunk.Usage)
}

func FromStreamChunks(chunks []providers.StreamChunk) (Usage, bool) {
	for i := len(chunks) - 1; i >= 0; i-- {
		if usage, ok := FromStreamChunk(chunks[i]); ok {
			return usage, true
		}
	}

	return Usage{}, false
}

func fromProviderUsage(providerUsage *providers.Usage) (Usage, bool) {
	if providerUsage == nil {
		return Usage{}, false
	}

	result := Usage{
		InputTokens:  providerUsage.PromptTokens,
		OutputTokens: providerUsage.CompletionTokens,
		TotalTokens:  providerUsage.TotalTokens,
	}
	if providerUsage.PromptTokensDetails != nil {
		result.CacheReadTokens = providerUsage.PromptTokensDetails.CachedTokens
	}
	if providerUsage.CompletionTokensDetails != nil {
		result.ReasoningTokens = providerUsage.CompletionTokensDetails.ReasoningTokens
	}

	return result, true
}
