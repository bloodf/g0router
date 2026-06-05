package bedrock

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// --- frame builder helpers ---

func encodeExceptionFrame(t *testing.T, exceptionType string, payload []byte) []byte {
	t.Helper()
	headers := encodeEventStreamHeader(":exception-type", exceptionType)
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

// --- bedrockErrorChunk (0.0% → covered by streamConverse bad-read path) ---

func TestBedrockErrorChunkDirect(t *testing.T) {
	chunk := bedrockErrorChunk("my-model", "upstream_stream_error", "something failed")
	if chunk.Model != "my-model" {
		t.Fatalf("model = %q, want my-model", chunk.Model)
	}
	if chunk.Error == nil {
		t.Fatal("expected non-nil Error")
	}
	if chunk.Error.Code != "upstream_stream_error" {
		t.Fatalf("code = %q", chunk.Error.Code)
	}
	if chunk.Error.Message != "something failed" {
		t.Fatalf("message = %q", chunk.Error.Message)
	}
}

func TestStreamConverseReadError(t *testing.T) {
	// Write a valid first frame then corrupt the second prelude so the reader
	// returns a non-EOF error (bad CRC), which triggers bedrockErrorChunk.
	frame1 := encodeEventStreamFrame(t, "messageStart", []byte(`{"role":"assistant"}`))

	// A partial/truncated second frame: valid prelude bytes but wrong prelude CRC.
	badFrame := make([]byte, 12)
	binary.BigEndian.PutUint32(badFrame[0:4], 20) // totalLen=20
	binary.BigEndian.PutUint32(badFrame[4:8], 0)  // headersLen=0
	binary.BigEndian.PutUint32(badFrame[8:12], 0xDEADBEEF) // bad prelude CRC

	var buf bytes.Buffer
	buf.Write(frame1)
	buf.Write(badFrame)

	chunks := make(chan providers.StreamChunk, 8)
	streamConverse("m", &buf, chunks)
	close(chunks)

	var sawError bool
	for c := range chunks {
		if c.Error != nil {
			sawError = true
		}
	}
	if !sawError {
		t.Fatal("expected an error chunk for bad prelude CRC mid-stream")
	}
}

// --- mapConverseEvent branches ---

func TestMapConverseEventExceptionWithFrameMessage(t *testing.T) {
	// frame.message is the parsed wire-level message field (not from JSON payload).
	frame := eventStreamFrame{eventType: "internalServerException", message: "exploded"}
	chunk, stop := mapConverseEvent("m", frame)
	if !stop {
		t.Fatal("exception frame must stop the stream")
	}
	if chunk == nil || chunk.Error == nil {
		t.Fatal("expected error chunk")
	}
	if chunk.Error.Message != "exploded" {
		t.Fatalf("message = %q, want exploded", chunk.Error.Message)
	}
}

func TestMapConverseEventExceptionEmptyFrameMessage(t *testing.T) {
	// When frame.message is empty, the event-type is used as the message.
	frame := eventStreamFrame{eventType: "modelTimeoutException"}
	chunk, stop := mapConverseEvent("m", frame)
	if !stop {
		t.Fatal("exception frame must stop")
	}
	if chunk.Error.Message != "modelTimeoutException" {
		t.Fatalf("message = %q, want modelTimeoutException", chunk.Error.Message)
	}
}

func TestMapConverseEventMalformedPayload(t *testing.T) {
	frame := eventStreamFrame{eventType: "contentBlockDelta", payload: []byte(`not-json`)}
	chunk, stop := mapConverseEvent("m", frame)
	if !stop {
		t.Fatal("malformed payload must stop the stream")
	}
	if chunk == nil || chunk.Error == nil {
		t.Fatal("expected error chunk for malformed payload")
	}
}

func TestMapConverseEventContentBlockStartNoToolUse(t *testing.T) {
	frame := eventStreamFrame{eventType: "contentBlockStart", payload: []byte(`{"start":{}}`)}
	chunk, stop := mapConverseEvent("m", frame)
	if stop {
		t.Fatal("contentBlockStart with no toolUse must not stop")
	}
	if chunk != nil {
		t.Fatal("expected nil chunk for contentBlockStart with no toolUse")
	}
}

func TestMapConverseEventContentBlockDeltaEmpty(t *testing.T) {
	// delta is nil → no chunk.
	frame := eventStreamFrame{eventType: "contentBlockDelta", payload: []byte(`{}`)}
	chunk, stop := mapConverseEvent("m", frame)
	if stop || chunk != nil {
		t.Fatalf("empty delta: stop=%v chunk=%v", stop, chunk)
	}
}

func TestMapConverseEventContentBlockDeltaToolInput(t *testing.T) {
	frame := eventStreamFrame{
		eventType: "contentBlockDelta",
		payload:   []byte(`{"delta":{"toolUse":{"input":"{\"key\":\"val\"}"},"contentBlockIndex":0}}`),
	}
	chunk, stop := mapConverseEvent("m", frame)
	if stop {
		t.Fatal("tool-input delta must not stop")
	}
	if chunk == nil || len(chunk.Choices) == 0 {
		t.Fatal("expected a chunk with tool call delta")
	}
}

func TestMapConverseEventContentBlockDeltaTextEmpty(t *testing.T) {
	// delta present but text == "" and no toolUse.
	frame := eventStreamFrame{
		eventType: "contentBlockDelta",
		payload:   []byte(`{"delta":{"text":""}}`),
	}
	chunk, stop := mapConverseEvent("m", frame)
	if stop || chunk != nil {
		t.Fatalf("empty text delta: stop=%v chunk=%v", stop, chunk)
	}
}

func TestMapConverseEventMetadataNoUsage(t *testing.T) {
	frame := eventStreamFrame{eventType: "metadata", payload: []byte(`{}`)}
	chunk, stop := mapConverseEvent("m", frame)
	if stop || chunk != nil {
		t.Fatalf("metadata with no usage: stop=%v chunk=%v", stop, chunk)
	}
}

func TestMapConverseEventMetadataZeroTotalTokens(t *testing.T) {
	// totalTokens == 0 → computed as prompt + completion.
	frame := eventStreamFrame{
		eventType: "metadata",
		payload:   []byte(`{"usage":{"inputTokens":3,"outputTokens":5,"totalTokens":0}}`),
	}
	chunk, stop := mapConverseEvent("m", frame)
	if stop {
		t.Fatal("metadata must not stop")
	}
	if chunk == nil || chunk.Usage == nil {
		t.Fatal("expected usage chunk")
	}
	if chunk.Usage.TotalTokens != 8 {
		t.Fatalf("totalTokens = %d, want 8", chunk.Usage.TotalTokens)
	}
}

func TestMapConverseEventUnknownType(t *testing.T) {
	frame := eventStreamFrame{eventType: "someFutureEvent", payload: []byte(`{}`)}
	chunk, stop := mapConverseEvent("m", frame)
	if stop || chunk != nil {
		t.Fatalf("unknown event: stop=%v chunk=%v", stop, chunk)
	}
}

func TestMapConverseEventEmptyPayload(t *testing.T) {
	// Empty payload must not cause an unmarshal error path; it just skips.
	frame := eventStreamFrame{eventType: "messageStop", payload: nil}
	chunk, stop := mapConverseEvent("m", frame)
	if stop {
		t.Fatal("messageStop must not stop stream")
	}
	if chunk == nil {
		t.Fatal("expected a chunk for messageStop")
	}
}

// --- frameReader.next error paths ---

func TestFrameReaderBadPreludeCRC(t *testing.T) {
	// Build a valid prelude but corrupt the CRC.
	prelude := make([]byte, 8)
	binary.BigEndian.PutUint32(prelude[0:4], 20) // totalLen=20 (minimal)
	binary.BigEndian.PutUint32(prelude[4:8], 0)  // headersLen=0
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, 0xDEADBEEF) // wrong CRC

	buf := bytes.NewReader(append(prelude, crcBuf...))
	fr := newFrameReader(buf)
	_, err := fr.next()
	if err == nil {
		t.Fatal("expected CRC mismatch error")
	}
}

func TestFrameReaderBadMessageCRC(t *testing.T) {
	prelude := make([]byte, 8)
	totalLen := uint32(20) // 12 prelude + 0 headers + 0 payload + 4 msg CRC + 4 extra = 20
	binary.BigEndian.PutUint32(prelude[0:4], totalLen)
	binary.BigEndian.PutUint32(prelude[4:8], 0)

	preludeCRC := crc32.ChecksumIEEE(prelude)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, preludeCRC)

	// rest = (totalLen - 12) bytes with a corrupt message CRC at the end.
	rest := make([]byte, totalLen-12) // 8 bytes: 0 payload + 4 bad msg CRC + 4 ???
	// Actually totalLen=20 → rest = 8 bytes; last 4 = msg CRC.
	binary.BigEndian.PutUint32(rest[len(rest)-4:], 0xDEADBEEF)

	var buf bytes.Buffer
	buf.Write(prelude)
	buf.Write(crcBuf)
	buf.Write(rest)

	fr := newFrameReader(&buf)
	_, err := fr.next()
	if err == nil {
		t.Fatal("expected message CRC mismatch error")
	}
}

func TestFrameReaderInvalidPreludeLengths(t *testing.T) {
	// totalLen < 16.
	prelude := make([]byte, 8)
	binary.BigEndian.PutUint32(prelude[0:4], 10) // too small
	binary.BigEndian.PutUint32(prelude[4:8], 0)

	preludeCRC := crc32.ChecksumIEEE(prelude)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, preludeCRC)

	var buf bytes.Buffer
	buf.Write(prelude)
	buf.Write(crcBuf)

	fr := newFrameReader(&buf)
	_, err := fr.next()
	if err == nil {
		t.Fatal("expected invalid prelude lengths error")
	}
}

func TestFrameReaderEmptyBodyReturnsEOF(t *testing.T) {
	fr := newFrameReader(bytes.NewReader(nil))
	_, err := fr.next()
	if err != io.EOF {
		t.Fatalf("empty body: err = %v, want io.EOF", err)
	}
}

// --- headerValue edge cases ---

func TestHeaderValueNonStringType(t *testing.T) {
	// A header with value-type != 7 should cause the scanner to stop.
	// name "foo" (3 bytes), type 5 (boolean), then more bytes.
	headers := []byte{
		3, 'f', 'o', 'o',
		5, // non-string type → stop
		0, 0, 4, 't', 'e', 's', 't',
	}
	if got := headerValue(headers, "foo"); got != "" {
		t.Fatalf("non-string type should return empty, got %q", got)
	}
}

func TestHeaderValueTruncatedAfterName(t *testing.T) {
	// Name fits but no bytes remain for value type.
	headers := []byte{3, 'f', 'o', 'o'} // nameLen=3, name="foo", then nothing
	if got := headerValue(headers, "foo"); got != "" {
		t.Fatalf("truncated header should return empty, got %q", got)
	}
}

func TestHeaderValueTruncatedValueLen(t *testing.T) {
	// Value type 7 present but only 1 byte for the 2-byte length.
	headers := []byte{3, 'f', 'o', 'o', 7, 0} // only 1 byte of the 2-byte length
	if got := headerValue(headers, "foo"); got != "" {
		t.Fatalf("truncated value length should return empty, got %q", got)
	}
}

func TestHeaderValueTruncatedValue(t *testing.T) {
	// Value length says 5 but only 2 bytes present.
	headers := []byte{3, 'f', 'o', 'o', 7, 0, 5, 'a', 'b'}
	if got := headerValue(headers, "foo"); got != "" {
		t.Fatalf("truncated value bytes should return empty, got %q", got)
	}
}

func TestHeaderValueNotFound(t *testing.T) {
	// Valid header for "bar" but asking for "foo".
	headers := encodeEventStreamHeader("bar", "baz")
	if got := headerValue(headers, "foo"); got != "" {
		t.Fatalf("missing key should return empty, got %q", got)
	}
}

func TestStreamConverseExceptionFrameViaWire(t *testing.T) {
	// Encode an exception frame (uses :exception-type header) and feed through the
	// full decode path to cover the headerValue ":exception-type" branch.
	frame := encodeExceptionFrame(t, "throttlingException", []byte(`{"message":"slow down"}`))
	var buf bytes.Buffer
	buf.Write(frame)

	chunks := make(chan providers.StreamChunk, 8)
	streamConverse("m", &buf, chunks)
	close(chunks)

	var sawError bool
	for c := range chunks {
		if c.Error != nil {
			sawError = true
		}
	}
	if !sawError {
		t.Fatal("expected error chunk from exception frame")
	}
}
