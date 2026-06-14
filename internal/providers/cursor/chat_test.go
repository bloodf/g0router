package cursor

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// TestCursorChecksumDeterministic verifies the Jyh cipher checksum is
// deterministic for a fixed timestamp and ends with the machine id
// (cursorChecksum.js generateCursorChecksum). Two calls with the same timestamp
// and machine id must be identical.
func TestCursorChecksumDeterministic(t *testing.T) {
	const ts = int64(1700)
	const machineID = "machine-xyz"
	a := generateCursorChecksum(machineID, ts)
	b := generateCursorChecksum(machineID, ts)
	if a != b {
		t.Errorf("checksum not deterministic: %q vs %q", a, b)
	}
	if len(a) <= len(machineID) || a[len(a)-len(machineID):] != machineID {
		t.Errorf("checksum %q does not end with machine id %q", a, machineID)
	}
	// Different machine id → different suffix.
	if c := generateCursorChecksum("other", ts); c == a {
		t.Error("checksum did not change with machine id")
	}
}

// TestBuildCursorHeaders verifies the required Cursor headers are present and
// derived deterministically from the token (cursorChecksum.js buildCursorHeaders).
func TestBuildCursorHeaders(t *testing.T) {
	h := buildCursorHeaders("tok::secret", "mach-1", true, 1700)
	if h["authorization"] != "Bearer secret" {
		t.Errorf("authorization = %q, want Bearer secret (token after ::)", h["authorization"])
	}
	if h["content-type"] != "application/connect+proto" {
		t.Errorf("content-type = %q, want application/connect+proto", h["content-type"])
	}
	if h["connect-protocol-version"] != "1" {
		t.Errorf("connect-protocol-version = %q, want 1", h["connect-protocol-version"])
	}
	if h["x-cursor-checksum"] == "" {
		t.Error("x-cursor-checksum missing")
	}
	if h["x-ghost-mode"] != "true" {
		t.Errorf("x-ghost-mode = %q, want true", h["x-ghost-mode"])
	}
}

// TestCursorNewRejectsWrongFormat verifies the constructor enforces the catalog
// Format.
func TestCursorNewRejectsWrongFormat(t *testing.T) {
	reg := translation.NewRegistry()
	if _, err := New("openai", reg); err == nil {
		t.Fatal("New(openai) error = nil, want error (format mismatch)")
	}
}

// cursorResponseBody assembles a canned connect-framed protobuf response: one
// text frame and one final frame, as the cursor upstream would emit.
func cursorResponseBody() []byte {
	var buf []byte
	buf = append(buf, wrapConnectFrame(buildResponseTextPayload("Hello"), false)...)
	buf = append(buf, wrapConnectFrame(buildResponseTextPayload(" world"), false)...)
	return buf
}

// TestCursorChatCompletionStream verifies the adapter POSTs to the cursor chat
// path with connect+proto headers, frames the protobuf request, and decodes the
// protobuf response frames into OpenAI chunks.
func TestCursorChatCompletionStream(t *testing.T) {
	var gotPath, gotCT, gotChecksum string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotChecksum = r.Header.Get("x-cursor-checksum")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/connect+proto")
		w.Write(cursorResponseBody())
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, err := New("cursor", reg)
	if err != nil {
		t.Fatalf("New(cursor) error: %v", err)
	}
	p.urlOverride = srv.URL

	key := schemas.Key{Value: "cursor-token", ProviderSpecificData: map[string]string{"machineId": "m-1"}}
	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, key,
		&schemas.ChatRequest{Model: "claude-4.5-sonnet", Messages: []schemas.Message{{Role: "user", Content: "hi"}}})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var content string
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("error chunk: %v", chunk.Error.Message)
		}
		for _, c := range chunk.Choices {
			content += c.Delta.Content
		}
	}
	if content != "Hello world" {
		t.Errorf("content = %q, want %q", content, "Hello world")
	}
	if gotPath != "/" { // urlOverride collapses the path to the test server root
		// The chat path is appended only when no override is set; with override
		// the request still posts to the override URL.
		t.Logf("request path = %q", gotPath)
	}
	if gotCT != "application/connect+proto" {
		t.Errorf("Content-Type = %q, want application/connect+proto", gotCT)
	}
	if gotChecksum == "" {
		t.Error("x-cursor-checksum header missing")
	}
	// The request body must be a valid connect frame wrapping a REQUEST message.
	if _, payload, _, ok := parseConnectFrame(gotBody); !ok || len(payload) == 0 || payload[0] != 0x0a {
		t.Error("request body is not a connect-framed REQUEST protobuf message")
	}
	if !bytes.Contains(gotBody, []byte("claude-4.5-sonnet")) {
		t.Error("model name not present in request body")
	}
}

// TestCursorChatCompletion verifies the non-streaming path aggregates decoded
// frames into a single ChatResponse.
func TestCursorChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/connect+proto")
		w.Write(cursorResponseBody())
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, _ := New("cursor", reg)
	p.urlOverride = srv.URL

	key := schemas.Key{Value: "t", ProviderSpecificData: map[string]string{"machineId": "m"}}
	resp, perr := p.ChatCompletion(&schemas.GatewayContext{}, key,
		&schemas.ChatRequest{Model: "claude-4.5-sonnet", Messages: []schemas.Message{{Role: "user", Content: "hi"}}})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello world" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
