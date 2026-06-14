package cursor

import (
	"bytes"
	"testing"
)

// TestEncodeVarint verifies the varint encoding matches the protobuf base-128
// scheme (cursorProtobuf.js encodeVarint) against golden wire bytes.
func TestEncodeVarint(t *testing.T) {
	cases := []struct {
		v    uint64
		want []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{127, []byte{0x7f}},
		{128, []byte{0x80, 0x01}},
		{300, []byte{0xac, 0x02}},
		{16384, []byte{0x80, 0x80, 0x01}},
	}
	for _, c := range cases {
		got := encodeVarint(c.v)
		if !bytes.Equal(got, c.want) {
			t.Errorf("encodeVarint(%d) = % x, want % x", c.v, got, c.want)
		}
	}
}

// TestEncodeFieldVarint verifies a VARINT field encodes tag + value
// (cursorProtobuf.js encodeField). Field 2, wire 0, value 1 → tag (2<<3)|0=0x10.
func TestEncodeFieldVarint(t *testing.T) {
	got := encodeFieldVarint(2, 1)
	want := []byte{0x10, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("encodeFieldVarint(2,1) = % x, want % x", got, want)
	}
}

// TestEncodeFieldLen verifies a LEN field encodes tag + length + bytes
// (cursorProtobuf.js encodeField). Field 1, wire 2, "hi" → tag (1<<3)|2=0x0a,
// length 2, then the bytes.
func TestEncodeFieldLen(t *testing.T) {
	got := encodeFieldLen(1, []byte("hi"))
	want := []byte{0x0a, 0x02, 'h', 'i'}
	if !bytes.Equal(got, want) {
		t.Errorf("encodeFieldLen(1,hi) = % x, want % x", got, want)
	}
}

// TestDecodeVarintRoundTrip verifies decodeVarint reverses encodeVarint.
func TestDecodeVarintRoundTrip(t *testing.T) {
	for _, v := range []uint64{0, 1, 127, 128, 300, 16384, 1 << 20} {
		enc := encodeVarint(v)
		got, n := decodeVarint(enc, 0)
		if got != v {
			t.Errorf("decodeVarint(encode(%d)) = %d", v, got)
		}
		if n != len(enc) {
			t.Errorf("decodeVarint(%d) consumed %d, want %d", v, n, len(enc))
		}
	}
}

// TestWrapConnectFrame verifies the connect-RPC frame envelope: 1-byte flags
// (0x00 uncompressed) + 4-byte big-endian length + payload
// (cursorProtobuf.js wrapConnectRPCFrame).
func TestWrapConnectFrame(t *testing.T) {
	payload := []byte{0xde, 0xad, 0xbe, 0xef}
	frame := wrapConnectFrame(payload, false)
	want := []byte{0x00, 0x00, 0x00, 0x00, 0x04, 0xde, 0xad, 0xbe, 0xef}
	if !bytes.Equal(frame, want) {
		t.Errorf("wrapConnectFrame = % x, want % x", frame, want)
	}
}

// TestParseConnectFrameUncompressed verifies a single uncompressed frame is
// parsed back into its payload (cursorProtobuf.js parseConnectRPCFrame).
func TestParseConnectFrameUncompressed(t *testing.T) {
	payload := []byte("payload-bytes")
	frame := wrapConnectFrame(payload, false)
	flags, got, consumed, ok := parseConnectFrame(frame)
	if !ok {
		t.Fatal("parseConnectFrame ok = false, want true")
	}
	if flags != 0x00 {
		t.Errorf("flags = %#x, want 0x00", flags)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("payload = % x, want % x", got, payload)
	}
	if consumed != len(frame) {
		t.Errorf("consumed = %d, want %d", consumed, len(frame))
	}
}

// TestParseConnectFrameGzip verifies a gzip-flagged frame (0x01) is decompressed
// (cursorProtobuf.js parseConnectRPCFrame gunzip branch).
func TestParseConnectFrameGzip(t *testing.T) {
	payload := []byte("the quick brown fox jumps over the lazy dog")
	frame := wrapConnectFrame(payload, true) // gzip-compressed
	if frame[0] != 0x01 {
		t.Fatalf("gzip frame flags = %#x, want 0x01", frame[0])
	}
	flags, got, _, ok := parseConnectFrame(frame)
	if !ok {
		t.Fatal("parseConnectFrame ok = false, want true")
	}
	if flags != 0x01 {
		t.Errorf("flags = %#x, want 0x01", flags)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("decompressed payload = %q, want %q", got, payload)
	}
}

// TestParseConnectFrameIncomplete verifies a short buffer yields ok=false.
func TestParseConnectFrameIncomplete(t *testing.T) {
	if _, _, _, ok := parseConnectFrame([]byte{0x00, 0x00}); ok {
		t.Error("parseConnectFrame(short) ok = true, want false")
	}
}

// buildResponseText assembles a golden StreamUnifiedChatResponseWithTools frame
// carrying a StreamUnifiedChatResponse (field 2) whose text (field 1) is the
// given string. This is real protobuf wire bytes (field numbers from
// cursorProtobuf.js FIELD: RESPONSE=2, RESPONSE_TEXT=1).
func buildResponseTextPayload(text string) []byte {
	inner := encodeFieldLen(1, []byte(text))     // RESPONSE_TEXT
	return encodeFieldLen(2, inner)              // RESPONSE
}

// TestExtractTextFromResponse verifies the response decoder extracts the text
// from a golden protobuf payload.
func TestExtractTextFromResponse(t *testing.T) {
	payload := buildResponseTextPayload("Hello from cursor")
	res := extractTextFromResponse(payload)
	if res.text != "Hello from cursor" {
		t.Errorf("text = %q, want %q", res.text, "Hello from cursor")
	}
	if res.toolCall != nil {
		t.Errorf("toolCall = %#v, want nil", res.toolCall)
	}
}

// TestExtractToolCallFromResponse verifies the response decoder extracts a tool
// call from a golden ClientSideToolV2Call payload (field 1) carrying tool id
// (field 3), name (field 9), raw args (field 10), and is_last (field 11).
func TestExtractToolCallFromResponse(t *testing.T) {
	call := bytes.Join([][]byte{
		encodeFieldLen(3, []byte("tc_1")),        // TOOL_ID
		encodeFieldLen(9, []byte("search")),      // TOOL_NAME
		encodeFieldLen(10, []byte(`{"q":"go"}`)), // TOOL_RAW_ARGS
		encodeFieldVarint(11, 1),                 // TOOL_IS_LAST
	}, nil)
	payload := encodeFieldLen(1, call) // TOOL_CALL

	res := extractTextFromResponse(payload)
	if res.toolCall == nil {
		t.Fatal("toolCall = nil, want a tool call")
	}
	if res.toolCall.id != "tc_1" {
		t.Errorf("toolCall.id = %q, want tc_1", res.toolCall.id)
	}
	if res.toolCall.name != "search" {
		t.Errorf("toolCall.name = %q, want search", res.toolCall.name)
	}
	if res.toolCall.arguments != `{"q":"go"}` {
		t.Errorf("toolCall.arguments = %q, want {\"q\":\"go\"}", res.toolCall.arguments)
	}
	if !res.toolCall.isLast {
		t.Error("toolCall.isLast = false, want true")
	}
}

// TestGenerateCursorBodyRoundTrip verifies a generated request body is a valid
// connect frame wrapping a protobuf message whose top-level field is REQUEST
// (field 1), and that the encoded model name is present in the wire bytes.
func TestGenerateCursorBodyRoundTrip(t *testing.T) {
	body := generateCursorBody(
		[]map[string]any{{"role": "user", "content": "hello"}},
		"claude-4.5-sonnet",
		nil, "", false,
	)
	flags, payload, _, ok := parseConnectFrame(body)
	if !ok {
		t.Fatal("generated body is not a valid connect frame")
	}
	if flags != 0x00 {
		t.Errorf("request frame flags = %#x, want 0x00 (cursor sends uncompressed)", flags)
	}
	// Top-level field must be REQUEST (field 1, wire LEN) → first tag byte 0x0a.
	if len(payload) == 0 || payload[0] != 0x0a {
		t.Fatalf("top-level field tag = %#x, want 0x0a (field 1 LEN)", payload[0])
	}
	if !bytes.Contains(payload, []byte("claude-4.5-sonnet")) {
		t.Error("model name not found in encoded request")
	}
	if !bytes.Contains(payload, []byte("hello")) {
		t.Error("message content not found in encoded request")
	}
}
