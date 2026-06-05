package replicate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

// ChatCompletionStream creates a streaming prediction, resolves its SSE stream
// URL (urls.stream), and translates Replicate's `event: output` / `event: done`
// server-sent events into provider stream chunks. Each `output` token becomes a
// content-delta chunk; `done` ends the stream with a stop finish reason; an
// `error` event surfaces an error chunk.
func (p *Provider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	if req == nil {
		return nil, fmt.Errorf("replicate stream: nil chat request")
	}
	if strings.TrimSpace(key.Value) == "" {
		return nil, fmt.Errorf("replicate stream: missing api key")
	}

	createReq := predictionCreateRequest{
		Model: req.Model,
		Input: map[string]any{"prompt": flattenMessages(req.Messages)},
		Stream: true,
	}
	prediction, err := p.createPrediction(ctx, key, createReq)
	if err != nil {
		return nil, fmt.Errorf("replicate stream: %w", err)
	}

	streamURL := prediction.URLs["stream"]
	if streamURL == "" {
		return nil, fmt.Errorf("replicate stream: prediction missing stream url")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("replicate stream: create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+key.Value)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("replicate stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("replicate stream: status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		streamSSE(req.Model, resp.Body, chunks)
	}()
	return chunks, nil
}

// streamSSE parses Replicate's SSE token stream. Replicate groups each event as
// an `event:` line followed by one or more `data:` lines, terminated by a blank
// line. Multi-line data values are joined with newlines, mirroring the SSE spec.
func streamSSE(model string, body io.Reader, chunks chan<- providers.StreamChunk) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventName string
	var dataLines []string

	dispatch := func() bool {
		defer func() {
			eventName = ""
			dataLines = nil
		}()
		if eventName == "" && len(dataLines) == 0 {
			return false
		}
		data := strings.Join(dataLines, "\n")
		return handleSSEEvent(model, eventName, data, chunks)
	}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if dispatch() {
				return
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if dispatch() {
		return
	}
	if err := scanner.Err(); err != nil {
		chunks <- replicateErrorChunk(model, err.Error())
	}
}

// handleSSEEvent maps one SSE event to a chunk. It returns true when the stream
// should terminate (a done or error event).
func handleSSEEvent(model, event, data string, chunks chan<- providers.StreamChunk) bool {
	switch event {
	case "output":
		if data == "" {
			return false
		}
		text := data
		chunks <- providers.StreamChunk{
			ID:      streamID(model),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []providers.StreamChoice{{
				Index: 0,
				Delta: providers.StreamDelta{Content: &text},
			}},
		}
		return false
	case "error":
		chunks <- replicateErrorChunk(model, strings.TrimSpace(data))
		return true
	case "done":
		finish := "stop"
		chunks <- providers.StreamChunk{
			ID:      streamID(model),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []providers.StreamChoice{{
				Index:        0,
				Delta:        providers.StreamDelta{},
				FinishReason: &finish,
			}},
		}
		return true
	default:
		return false
	}
}

func replicateErrorChunk(model, message string) providers.StreamChunk {
	if message == "" {
		message = "replicate stream error"
	}
	return providers.StreamChunk{
		ID:      streamID(model),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Error: &providers.StreamError{
			Message: message,
			Type:    "server_error",
			Code:    "upstream_stream_error",
		},
	}
}

func streamID(model string) string {
	return "replicate-" + model + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
}
