package bedrock

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// encodeEventStreamFrame builds a single vnd.amazon.eventstream frame with the
// given event-type header and JSON payload, matching the wire format the
// decoder under test must parse: prelude (total length, headers length,
// prelude CRC), headers, payload, and a trailing message CRC.
func encodeEventStreamFrame(t *testing.T, eventType string, payload []byte) []byte {
	t.Helper()

	headers := encodeEventStreamHeader(":event-type", eventType)
	headers = append(headers, encodeEventStreamHeader(":content-type", "application/json")...)

	totalLen := 16 + len(headers) + len(payload)
	frame := make([]byte, 0, totalLen)

	prelude := make([]byte, 8)
	binary.BigEndian.PutUint32(prelude[0:4], uint32(totalLen))
	binary.BigEndian.PutUint32(prelude[4:8], uint32(len(headers)))
	frame = append(frame, prelude...)

	preludeCRC := crc32.ChecksumIEEE(prelude)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, preludeCRC)
	frame = append(frame, crcBuf...)

	frame = append(frame, headers...)
	frame = append(frame, payload...)

	msgCRC := crc32.ChecksumIEEE(frame)
	binary.BigEndian.PutUint32(crcBuf, msgCRC)
	frame = append(frame, crcBuf...)
	return frame
}

// encodeEventStreamHeader encodes a single string-typed event-stream header.
func encodeEventStreamHeader(name, value string) []byte {
	buf := make([]byte, 0, len(name)+len(value)+4)
	buf = append(buf, byte(len(name)))
	buf = append(buf, []byte(name)...)
	buf = append(buf, 7) // value type 7 = string
	valLen := make([]byte, 2)
	binary.BigEndian.PutUint16(valLen, uint16(len(value)))
	buf = append(buf, valLen...)
	buf = append(buf, []byte(value)...)
	return buf
}

func eventStreamServer(t *testing.T, frames [][]byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		var body bytes.Buffer
		for _, frame := range frames {
			body.Write(frame)
		}
		_, _ = w.Write(body.Bytes())
	}))
	t.Cleanup(server.Close)
	return server
}

func collectStream(t *testing.T, ch <-chan providers.StreamChunk) []providers.StreamChunk {
	t.Helper()
	var chunks []providers.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}
	return chunks
}

func TestChatCompletionStreamHappyMultiChunk(t *testing.T) {
	frames := [][]byte{
		encodeEventStreamFrame(t, "messageStart", []byte(`{"role":"assistant"}`)),
		encodeEventStreamFrame(t, "contentBlockDelta", []byte(`{"delta":{"text":"Hello"},"contentBlockIndex":0}`)),
		encodeEventStreamFrame(t, "contentBlockDelta", []byte(`{"delta":{"text":", world"},"contentBlockIndex":0}`)),
		encodeEventStreamFrame(t, "messageStop", []byte(`{"stopReason":"end_turn"}`)),
		encodeEventStreamFrame(t, "metadata", []byte(`{"usage":{"inputTokens":5,"outputTokens":7,"totalTokens":12}}`)),
	}
	server := eventStreamServer(t, frames)

	provider := New(server.URL)
	ch, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	chunks := collectStream(t, ch)

	var content string
	var sawRole bool
	var finish *string
	var usage *providers.Usage
	for _, chunk := range chunks {
		if chunk.Error != nil {
			t.Fatalf("unexpected error chunk: %+v", chunk.Error)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Role != nil {
				sawRole = true
			}
			if choice.Delta.Content != nil {
				content += *choice.Delta.Content
			}
			if choice.FinishReason != nil {
				finish = choice.FinishReason
			}
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	if !sawRole {
		t.Errorf("expected a role delta from messageStart")
	}
	if content != "Hello, world" {
		t.Errorf("content = %q, want %q", content, "Hello, world")
	}
	if finish == nil || *finish != "end_turn" {
		t.Errorf("finish = %v, want end_turn", finish)
	}
	if usage == nil || usage.PromptTokens != 5 || usage.CompletionTokens != 7 || usage.TotalTokens != 12 {
		t.Errorf("usage = %+v, want 5/7/12", usage)
	}
}

func TestChatCompletionStreamEmitsErrorFrame(t *testing.T) {
	frames := [][]byte{
		encodeEventStreamFrame(t, "contentBlockDelta", []byte(`{"delta":{"text":"partial"},"contentBlockIndex":0}`)),
		encodeEventStreamFrame(t, "internalServerException", []byte(`{"message":"boom"}`)),
	}
	server := eventStreamServer(t, frames)

	provider := New(server.URL)
	ch, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	chunks := collectStream(t, ch)

	var sawError bool
	var content string
	for _, chunk := range chunks {
		if chunk.Error != nil {
			sawError = true
			if chunk.Error.Message == "" {
				t.Errorf("error chunk missing message")
			}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != nil {
				content += *choice.Delta.Content
			}
		}
	}
	if content != "partial" {
		t.Errorf("content = %q, want partial before error", content)
	}
	if !sawError {
		t.Fatalf("expected an error chunk for the exception frame")
	}
}

func TestChatCompletionStreamRejectsBadCredentials(t *testing.T) {
	provider := New("https://bedrock.example")
	_, err := provider.ChatCompletionStream(context.Background(), providers.Key{Value: "bad"}, testChatRequest())
	if err == nil {
		t.Fatal("expected error for malformed credentials")
	}
}

func TestChatCompletionStreamMapsHTTPError(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusUnauthorized, `{"message":"denied"}`)
	provider := New(server.URL)
	_, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
}
