package openai

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestSpeechSuccessReturnsRawBytes verifies Speech copies the upstream audio
// body verbatim into SpeechResponse.Audio and the upstream Content-Type into
// SpeechResponse.ContentType (ESC-SPEECH-BYTES: non-JSON binary body).
func TestSpeechSuccessReturnsRawBytes(t *testing.T) {
	audio := []byte{0x49, 0x44, 0x33, 0x04, 0x00, 0x01, 0x02, 0x03} // "ID3" + bytes
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			t.Errorf("path = %q, want /v1/audio/speech", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth = %q, want Bearer test-key", got)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audio)
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.Speech(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.SpeechRequest{
		Model: "tts-1", Input: "hello", Voice: "alloy",
	})
	if perr != nil {
		t.Fatalf("Speech error: %v", perr.Message)
	}
	if string(resp.Audio) != string(audio) {
		t.Errorf("Audio = %v, want %v", resp.Audio, audio)
	}
	if resp.ContentType != "audio/mpeg" {
		t.Errorf("ContentType = %q, want audio/mpeg", resp.ContentType)
	}
}

// TestSpeechUpstreamErrorSurfacesProviderError verifies an upstream non-200
// becomes a *ProviderError carrying the status code.
func TestSpeechUpstreamErrorSurfacesProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"boom","type":"api_error"}}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.Speech(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.SpeechRequest{Model: "tts-1"})
	if perr == nil {
		t.Fatal("expected *ProviderError, got nil")
	}
	if perr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", perr.StatusCode)
	}
}

// TestTranscriptionSendsMultipartAndReturnsResponse verifies Transcription
// builds a multipart/form-data outbound body whose file part round-trips, and
// parses the JSON TranscriptionResponse (ESC-MULTIPART).
func TestTranscriptionSendsMultipartAndReturnsResponse(t *testing.T) {
	fileBytes := []byte("RIFF....WAVEfmt fake-audio")
	var gotFile []byte
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("inbound Content-Type = %q, want multipart/form-data", ct)
		}
		_, params, err := mime.ParseMediaType(ct)
		if err != nil {
			t.Fatalf("parse media type: %v", err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		form, err := mr.ReadForm(1 << 20)
		if err != nil {
			t.Fatalf("read form: %v", err)
		}
		if fhs := form.File["file"]; len(fhs) == 1 {
			f, _ := fhs[0].Open()
			gotFile, _ = io.ReadAll(f)
		}
		if mv := form.Value["model"]; len(mv) == 1 {
			gotModel = mv[0]
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":"hello world"}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.Transcription(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.TranscriptionRequest{
		File: fileBytes, Model: "whisper-1",
	})
	if perr != nil {
		t.Fatalf("Transcription error: %v", perr.Message)
	}
	if string(gotFile) != string(fileBytes) {
		t.Errorf("file part = %q, want round-trip of input", gotFile)
	}
	if gotModel != "whisper-1" {
		t.Errorf("model field = %q, want whisper-1", gotModel)
	}
	if resp.Text != "hello world" {
		t.Errorf("Text = %q, want hello world", resp.Text)
	}
}

// TestTranscriptionUpstreamError verifies non-200 surfaces a *ProviderError.
func TestTranscriptionUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"bad","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.Transcription(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.TranscriptionRequest{File: []byte("x"), Model: "whisper-1"})
	if perr == nil {
		t.Fatal("expected *ProviderError, got nil")
	}
	if perr.StatusCode != 400 {
		t.Errorf("status = %d, want 400", perr.StatusCode)
	}
}

// TestSpeechStreamForwardsChunksThenDone verifies SpeechStream drains SSE
// frames and stops at [DONE].
func TestSpeechStreamForwardsChunksThenDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"s1\"}\n\n")
		io.WriteString(w, "data: {\"id\":\"s2\"}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.SpeechStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "k"}, &schemas.SpeechRequest{Model: "tts-1"})
	if perr != nil {
		t.Fatalf("SpeechStream error: %v", perr.Message)
	}
	var n int
	for chunk := range ch {
		if chunk.Error != nil {
			t.Errorf("unexpected error chunk: %v", chunk.Error.Message)
			continue
		}
		n++
	}
	if n != 2 {
		t.Errorf("chunks = %d, want 2", n)
	}
}

// TestTranscriptionStreamAbortsOnMalformedChunk verifies AUD-045 for the
// transcription stream: a malformed chunk aborts with one error chunk.
func TestTranscriptionStreamAbortsOnMalformedChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("inbound Content-Type = %q, want multipart/form-data", ct)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"t1\"}\n\n")
		io.WriteString(w, "data: not-json{\n\n")
		io.WriteString(w, "data: {\"id\":\"t2\"}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.TranscriptionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "k"}, &schemas.TranscriptionRequest{File: []byte("x"), Model: "whisper-1"})
	if perr != nil {
		t.Fatalf("TranscriptionStream error: %v", perr.Message)
	}
	var content, errChunks int
	for chunk := range ch {
		if chunk.Error != nil {
			errChunks++
			continue
		}
		content++
	}
	if content != 1 {
		t.Errorf("content chunks = %d, want 1 (abort at malformed)", content)
	}
	if errChunks != 1 {
		t.Errorf("error chunks = %d, want 1", errChunks)
	}
}
