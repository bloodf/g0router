package providers

import "context"

type Provider interface {
	Name() ModelProvider
	ChatCompletion(ctx context.Context, key Key, req *ChatRequest) (*ChatResponse, error)
	ChatCompletionStream(ctx context.Context, key Key, req *ChatRequest) (<-chan StreamChunk, error)
	ListModels(ctx context.Context, key Key) ([]Model, error)
}
