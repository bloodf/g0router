package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// ---- Embeddings ----

func TestEmbeddings(t *testing.T) {
	var gotPath string
	var gotBody providers.EmbeddingsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1,0.2,0.3]}],"model":"text-embedding-3-small","usage":{"prompt_tokens":4,"total_tokens":4}}`))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	resp, err := p.Embeddings(context.Background(), testKey(), &providers.EmbeddingsRequest{
		Model: "text-embedding-3-small",
		Input: "hello",
	})
	if err != nil {
		t.Fatalf("Embeddings: %v", err)
	}
	if gotPath != "/v1/embeddings" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody.Model != "text-embedding-3-small" {
		t.Fatalf("model = %q", gotBody.Model)
	}
	if resp.Object != "list" || len(resp.Data) != 1 || len(resp.Data[0].Embedding) != 3 {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Usage == nil || resp.Usage.PromptTokens != 4 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestEmbeddingsNilRequest(t *testing.T) {
	p := New("http://example.com")
	if _, err := p.Embeddings(context.Background(), testKey(), nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestEmbeddingsErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"bad"}}`, nil)
	p := New(server.URL)
	if _, err := p.Embeddings(context.Background(), testKey(), &providers.EmbeddingsRequest{Model: "m"}); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestEmbeddingsBadJSON(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	if _, err := p.Embeddings(context.Background(), testKey(), &providers.EmbeddingsRequest{Model: "m"}); err == nil {
		t.Fatal("expected parse error")
	}
}

// ---- Images ----

func TestGenerateImages(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1710000000,"data":[{"url":"https://img/1.png"}]}`))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	resp, err := p.GenerateImages(context.Background(), testKey(), &providers.ImagesRequest{
		Model:  "dall-e-3",
		Prompt: "a cat",
	})
	if err != nil {
		t.Fatalf("GenerateImages: %v", err)
	}
	if gotPath != "/v1/images/generations" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://img/1.png" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestGenerateImagesNilRequest(t *testing.T) {
	p := New("http://example.com")
	if _, err := p.GenerateImages(context.Background(), testKey(), nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestGenerateImagesErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"down"}}`, nil)
	p := New(server.URL)
	if _, err := p.GenerateImages(context.Background(), testKey(), &providers.ImagesRequest{Model: "m"}); !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

// ---- Audio transcription ----

func TestTranscribeAudio(t *testing.T) {
	var gotPath, gotModel, gotFilename, gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
		}
		gotModel = r.FormValue("model")
		if f, header, err := r.FormFile("file"); err == nil {
			gotFilename = header.Filename
			_ = f.Close()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"hello world"}`))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	resp, err := p.TranscribeAudio(context.Background(), testKey(), &providers.AudioTranscriptionRequest{
		Model:    "whisper-1",
		Filename: "speech.mp3",
		File:     []byte("audio-bytes"),
	})
	if err != nil {
		t.Fatalf("TranscribeAudio: %v", err)
	}
	if gotPath != "/v1/audio/transcriptions" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Fatalf("content-type = %q", gotContentType)
	}
	if gotModel != "whisper-1" || gotFilename != "speech.mp3" {
		t.Fatalf("model=%q filename=%q", gotModel, gotFilename)
	}
	if resp.Text != "hello world" {
		t.Fatalf("text = %q", resp.Text)
	}
}

func TestTranscribeAudioNilRequest(t *testing.T) {
	p := New("http://example.com")
	if _, err := p.TranscribeAudio(context.Background(), testKey(), nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestTranscribeAudioErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusForbidden, `{"error":{"message":"no"}}`, nil)
	p := New(server.URL)
	if _, err := p.TranscribeAudio(context.Background(), testKey(), &providers.AudioTranscriptionRequest{Model: "m", Filename: "a.mp3", File: []byte("x")}); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

// ---- Speech ----

func TestSpeech(t *testing.T) {
	var gotPath string
	var gotBody providers.SpeechRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("MP3DATA"))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	audio, contentType, err := p.Speech(context.Background(), testKey(), &providers.SpeechRequest{
		Model: "tts-1",
		Input: "hi",
		Voice: "alloy",
	})
	if err != nil {
		t.Fatalf("Speech: %v", err)
	}
	if gotPath != "/v1/audio/speech" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody.Voice != "alloy" {
		t.Fatalf("voice = %q", gotBody.Voice)
	}
	if string(audio) != "MP3DATA" {
		t.Fatalf("audio = %q", audio)
	}
	if contentType != "audio/mpeg" {
		t.Fatalf("content-type = %q", contentType)
	}
}

func TestSpeechNilRequest(t *testing.T) {
	p := New("http://example.com")
	if _, _, err := p.Speech(context.Background(), testKey(), nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestSpeechErrorStatus(t *testing.T) {
	server := jsonServer(t, http.StatusTooManyRequests, `{"error":{"message":"slow"}}`, nil)
	p := New(server.URL)
	if _, _, err := p.Speech(context.Background(), testKey(), &providers.SpeechRequest{Model: "m", Input: "x", Voice: "alloy"}); !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
}

// ---- interface conformance ----

func TestOpenAIImplementsCapabilities(t *testing.T) {
	p := New("")
	var _ providers.EmbeddingsProvider = p
	var _ providers.ImagesProvider = p
	var _ providers.AudioTranscriptionProvider = p
	var _ providers.SpeechProvider = p
}
