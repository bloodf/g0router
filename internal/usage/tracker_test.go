package usage

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestFromChatResponseExtractsUsage(t *testing.T) {
	resp := providers.ChatResponse{
		Model: "gpt-4o",
		Usage: &providers.Usage{
			PromptTokens:     120,
			CompletionTokens: 45,
			TotalTokens:      165,
			PromptTokensDetails: &providers.PromptTokensDetails{
				CachedTokens: 30,
			},
			CompletionTokensDetails: &providers.CompletionTokensDetails{
				ReasoningTokens: 12,
			},
		},
	}

	got, ok := FromChatResponse(resp)
	if !ok {
		t.Fatal("expected usage")
	}
	if got.InputTokens != 120 {
		t.Fatalf("input tokens = %d, want 120", got.InputTokens)
	}
	if got.OutputTokens != 45 {
		t.Fatalf("output tokens = %d, want 45", got.OutputTokens)
	}
	if got.TotalTokens != 165 {
		t.Fatalf("total tokens = %d, want 165", got.TotalTokens)
	}
	if got.CacheReadTokens != 30 {
		t.Fatalf("cache read tokens = %d, want 30", got.CacheReadTokens)
	}
	if got.ReasoningTokens != 12 {
		t.Fatalf("reasoning tokens = %d, want 12", got.ReasoningTokens)
	}
}

func TestFromChatResponseWithoutUsage(t *testing.T) {
	_, ok := FromChatResponse(providers.ChatResponse{Model: "gpt-4o"})
	if ok {
		t.Fatal("expected no usage")
	}
}

func TestFromStreamChunkExtractsFinalUsage(t *testing.T) {
	chunk := providers.StreamChunk{
		Model: "gpt-4o-mini",
		Usage: &providers.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	got, ok := FromStreamChunk(chunk)
	if !ok {
		t.Fatal("expected usage")
	}
	if got.InputTokens != 10 || got.OutputTokens != 5 || got.TotalTokens != 15 {
		t.Fatalf("usage = %+v, want 10/5/15", got)
	}
}

func TestFromStreamChunksUsesChunkWithUsage(t *testing.T) {
	chunks := []providers.StreamChunk{
		{Model: "gpt-4o-mini"},
		{
			Model: "gpt-4o-mini",
			Usage: &providers.Usage{
				PromptTokens:     20,
				CompletionTokens: 7,
				TotalTokens:      27,
			},
		},
	}

	got, ok := FromStreamChunks(chunks)
	if !ok {
		t.Fatal("expected usage")
	}
	if got.InputTokens != 20 || got.OutputTokens != 7 || got.TotalTokens != 27 {
		t.Fatalf("usage = %+v, want 20/7/27", got)
	}
}
