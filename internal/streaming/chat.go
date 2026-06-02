package streaming

import "github.com/bloodf/g0router/internal/providers"

func AccumulateChat(chunks []providers.StreamChunk) providers.ChatResponse {
	acc := NewAccumulator()
	for _, chunk := range chunks {
		acc.AddChunk(chunk)
	}
	return acc.Response()
}
