package streaming

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestAccumulateChat(t *testing.T) {
	role := "assistant"
	content1 := "Hello"
	content2 := " world"
	chunks := []providers.StreamChunk{
		{
			Choices: []providers.StreamChoice{
				{Delta: providers.StreamDelta{Role: &role, Content: &content1}},
			},
		},
		{
			Choices: []providers.StreamChoice{
				{Delta: providers.StreamDelta{Content: &content2}},
			},
		},
	}
	resp := AccumulateChat(chunks)
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello world" {
		t.Fatalf("content = %q, want 'Hello world'", resp.Choices[0].Message.Content)
	}
}
