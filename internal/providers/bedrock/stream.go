package bedrock

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

// ChatCompletionStream invokes the Bedrock ConverseStream API and translates the
// returned vnd.amazon.eventstream frames into a channel of provider stream
// chunks. The event-stream body is a sequence of binary frames; each frame is
// decoded by frameReader and its JSON payload is mapped to a StreamChunk.
func (p *BedrockProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	creds, err := parseCredentials(key.Value)
	if err != nil {
		return nil, fmt.Errorf("parse bedrock credentials: %w", err)
	}

	httpReq, err := p.newConverseStreamRequest(ctx, creds, req)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bedrock converse stream: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read bedrock stream error: %w", readErr)
		}
		return nil, mapError(resp.StatusCode, body)
	}

	chunks := make(chan providers.StreamChunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		streamConverse(req.Model, resp.Body, chunks)
	}()
	return chunks, nil
}

func (p *BedrockProvider) newConverseStreamRequest(ctx context.Context, creds credentials, chatReq *providers.ChatRequest) (*http.Request, error) {
	converseReq, err := toConverseRequest(chatReq)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(converseReq)
	if err != nil {
		return nil, fmt.Errorf("marshal bedrock stream request: %w", err)
	}

	endpoint := p.baseURL + "/model/" + url.PathEscape(chatReq.Model) + "/converse-stream"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create bedrock stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.amazon.eventstream")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(string(body))), nil
	}

	if err := p.sign(req, creds, body); err != nil {
		return nil, fmt.Errorf("sign bedrock stream request: %w", err)
	}
	return req, nil
}

// streamConverse reads event-stream frames from body and emits mapped chunks.
func streamConverse(model string, body io.Reader, chunks chan<- providers.StreamChunk) {
	reader := newFrameReader(body)
	for {
		frame, err := reader.next()
		if err == io.EOF {
			return
		}
		if err != nil {
			chunks <- bedrockErrorChunk(model, "upstream_stream_error", err.Error())
			return
		}
		chunk, stop := mapConverseEvent(model, frame)
		if chunk != nil {
			chunks <- *chunk
		}
		if stop {
			return
		}
	}
}

// converseStreamEvent is the JSON payload carried by a ConverseStream frame. The
// fields present depend on the frame's :event-type header.
type converseStreamEvent struct {
	Role  string `json:"role"`
	Delta *struct {
		Text    string `json:"text"`
		ToolUse *struct {
			Input string `json:"input"`
		} `json:"toolUse"`
	} `json:"delta"`
	Start *struct {
		ToolUse *struct {
			ToolUseID string `json:"toolUseId"`
			Name      string `json:"name"`
		} `json:"toolUse"`
	} `json:"start"`
	StopReason string `json:"stopReason"`
	Usage      *struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
		TotalTokens  int `json:"totalTokens"`
	} `json:"usage"`
	Message string `json:"message"`
}

// mapConverseEvent translates a decoded frame to a StreamChunk. The boolean
// return indicates the stream should terminate after this frame.
func mapConverseEvent(model string, frame eventStreamFrame) (*providers.StreamChunk, bool) {
	if frame.exception() {
		message := frame.message
		if message == "" {
			message = frame.eventType
		}
		return &providers.StreamChunk{
			ID:      streamID(model),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Error: &providers.StreamError{
				Message: message,
				Type:    "server_error",
				Code:    frame.eventType,
			},
		}, true
	}

	var event converseStreamEvent
	if len(frame.payload) > 0 {
		if err := json.Unmarshal(frame.payload, &event); err != nil {
			return &providers.StreamChunk{
				ID:      streamID(model),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   model,
				Error: &providers.StreamError{
					Message: "malformed bedrock stream payload",
					Type:    "server_error",
					Code:    "upstream_stream_malformed",
				},
			}, true
		}
	}

	switch frame.eventType {
	case "messageStart":
		role := event.Role
		if role == "" {
			role = "assistant"
		}
		return newDeltaChunk(model, providers.StreamDelta{Role: &role}, nil), false
	case "contentBlockStart":
		if event.Start != nil && event.Start.ToolUse != nil {
			delta := providers.StreamDelta{ToolCalls: []providers.ToolCall{{
				ID:   event.Start.ToolUse.ToolUseID,
				Type: "function",
				Function: providers.ToolCallFunc{
					Name: event.Start.ToolUse.Name,
				},
			}}}
			return newDeltaChunk(model, delta, nil), false
		}
		return nil, false
	case "contentBlockDelta":
		if event.Delta == nil {
			return nil, false
		}
		if event.Delta.ToolUse != nil {
			delta := providers.StreamDelta{ToolCalls: []providers.ToolCall{{
				Type: "function",
				Function: providers.ToolCallFunc{
					Arguments: event.Delta.ToolUse.Input,
				},
			}}}
			return newDeltaChunk(model, delta, nil), false
		}
		if event.Delta.Text != "" {
			text := event.Delta.Text
			return newDeltaChunk(model, providers.StreamDelta{Content: &text}, nil), false
		}
		return nil, false
	case "messageStop":
		reason := event.StopReason
		return newDeltaChunk(model, providers.StreamDelta{}, &reason), false
	case "metadata":
		if event.Usage == nil {
			return nil, false
		}
		usage := &providers.Usage{
			PromptTokens:     event.Usage.InputTokens,
			CompletionTokens: event.Usage.OutputTokens,
			TotalTokens:      event.Usage.TotalTokens,
		}
		if usage.TotalTokens == 0 {
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		}
		return &providers.StreamChunk{
			ID:      streamID(model),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Usage:   usage,
		}, false
	default:
		return nil, false
	}
}

func newDeltaChunk(model string, delta providers.StreamDelta, finish *string) *providers.StreamChunk {
	return &providers.StreamChunk{
		ID:      streamID(model),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []providers.StreamChoice{{
			Index:        0,
			Delta:        delta,
			FinishReason: finish,
		}},
	}
}

func bedrockErrorChunk(model, code, message string) providers.StreamChunk {
	return providers.StreamChunk{
		ID:      streamID(model),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Error: &providers.StreamError{
			Message: message,
			Type:    "server_error",
			Code:    code,
		},
	}
}

func streamID(model string) string {
	return "bedrock-" + model + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

// eventStreamFrame is a single decoded vnd.amazon.eventstream message.
type eventStreamFrame struct {
	eventType string
	message   string
	payload   []byte
}

// exception reports whether the frame carries a Bedrock error/exception event
// rather than a normal Converse event.
func (f eventStreamFrame) exception() bool {
	return strings.HasSuffix(f.eventType, "Exception") || strings.EqualFold(f.eventType, "exception")
}

// frameReader decodes the minimal subset of the AWS event-stream binary format
// needed to read Converse stream payloads: a 12-byte prelude (4 total length, 4
// headers length, 4 prelude CRC), the headers blob, the payload, and a trailing
// 4-byte message CRC. Prelude and message CRCs are validated.
type frameReader struct {
	r *bufio.Reader
}

func newFrameReader(body io.Reader) *frameReader {
	return &frameReader{r: bufio.NewReader(body)}
}

func (fr *frameReader) next() (eventStreamFrame, error) {
	prelude := make([]byte, 8)
	if _, err := io.ReadFull(fr.r, prelude); err != nil {
		if err == io.ErrUnexpectedEOF {
			return eventStreamFrame{}, io.EOF
		}
		return eventStreamFrame{}, err
	}

	totalLen := binary.BigEndian.Uint32(prelude[0:4])
	headersLen := binary.BigEndian.Uint32(prelude[4:8])
	if totalLen < 16 || headersLen > totalLen-16 {
		return eventStreamFrame{}, fmt.Errorf("invalid event-stream prelude lengths total=%d headers=%d", totalLen, headersLen)
	}

	preludeCRCBuf := make([]byte, 4)
	if _, err := io.ReadFull(fr.r, preludeCRCBuf); err != nil {
		return eventStreamFrame{}, err
	}
	if got := binary.BigEndian.Uint32(preludeCRCBuf); got != crc32.ChecksumIEEE(prelude) {
		return eventStreamFrame{}, fmt.Errorf("event-stream prelude crc mismatch")
	}

	rest := make([]byte, totalLen-12)
	if _, err := io.ReadFull(fr.r, rest); err != nil {
		return eventStreamFrame{}, err
	}

	// Validate the message CRC over the full frame (prelude + preludeCRC + rest
	// minus the trailing 4-byte message CRC).
	full := make([]byte, 0, totalLen)
	full = append(full, prelude...)
	full = append(full, preludeCRCBuf...)
	full = append(full, rest[:len(rest)-4]...)
	msgCRC := binary.BigEndian.Uint32(rest[len(rest)-4:])
	if msgCRC != crc32.ChecksumIEEE(full) {
		return eventStreamFrame{}, fmt.Errorf("event-stream message crc mismatch")
	}

	headers := rest[:headersLen]
	payload := rest[headersLen : len(rest)-4]

	frame := eventStreamFrame{
		eventType: headerValue(headers, ":event-type"),
		payload:   payload,
	}
	if frame.eventType == "" {
		frame.eventType = headerValue(headers, ":exception-type")
	}
	return frame, nil
}

// headerValue extracts a string-typed header value by name from an event-stream
// headers blob. Each header is: name length (1 byte), name, value type (1 byte),
// and for string values a 2-byte length followed by the value bytes. Non-string
// header types in the Converse stream are skipped.
func headerValue(headers []byte, name string) string {
	for offset := 0; offset < len(headers); {
		nameLen := int(headers[offset])
		offset++
		if offset+nameLen > len(headers) {
			return ""
		}
		headerName := string(headers[offset : offset+nameLen])
		offset += nameLen
		if offset >= len(headers) {
			return ""
		}
		valueType := headers[offset]
		offset++
		if valueType != 7 {
			// Only string (7) headers are expected/needed here; anything else
			// means we cannot safely advance, so stop scanning.
			return ""
		}
		if offset+2 > len(headers) {
			return ""
		}
		valueLen := int(binary.BigEndian.Uint16(headers[offset : offset+2]))
		offset += 2
		if offset+valueLen > len(headers) {
			return ""
		}
		value := string(headers[offset : offset+valueLen])
		offset += valueLen
		if headerName == name {
			return value
		}
	}
	return ""
}
