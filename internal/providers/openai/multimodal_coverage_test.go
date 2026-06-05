package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

// ---- Embeddings: newJSONRequest error via marshal-impossible body ----

func TestEmbeddingsMarshalError(t *testing.T) {
	p := New("http://example.com")
	// channels cannot be marshalled to JSON → newJSONRequest returns error
	type badReq struct {
		Ch chan int `json:"ch"`
	}
	// We can't pass badReq as EmbeddingsRequest, so trigger via do() network error
	// instead: use an expired deadline to hit the do() error branch.
	deadline := time.Now().Add(-time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, err := p.Embeddings(ctx, testKey(), &providers.EmbeddingsRequest{Model: "m", Input: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- Embeddings: do() network error ----

func TestEmbeddingsDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.Embeddings(ctx, testKey(), &providers.EmbeddingsRequest{Model: "m", Input: "x"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- Embeddings: bad JSON response ----

func TestEmbeddingsBadJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.Embeddings(context.Background(), testKey(), &providers.EmbeddingsRequest{Model: "m", Input: "x"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// ---- GenerateImages: newJSONRequest error via expired deadline ----

func TestGenerateImagesExpiredDeadline(t *testing.T) {
	p := New("http://127.0.0.1:1")
	deadline := time.Now().Add(-time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, err := p.GenerateImages(ctx, testKey(), &providers.ImagesRequest{Model: "m", Prompt: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- GenerateImages: do() network error ----

func TestGenerateImagesDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.GenerateImages(ctx, testKey(), &providers.ImagesRequest{Model: "m", Prompt: "x"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- GenerateImages: bad JSON response ----

func TestGenerateImagesBadJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.GenerateImages(context.Background(), testKey(), &providers.ImagesRequest{Model: "m", Prompt: "x"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// ---- TranscribeAudio: buildTranscriptionMultipart error via do() ----

func TestTranscribeAudioDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.TranscribeAudio(ctx, testKey(), &providers.AudioTranscriptionRequest{
		Model: "whisper-1", Filename: "a.mp3", File: []byte("x"),
	})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- TranscribeAudio: bad JSON response ----

func TestTranscribeAudioBadJSONResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, `not json`, nil)
	p := New(server.URL)
	_, err := p.TranscribeAudio(context.Background(), testKey(), &providers.AudioTranscriptionRequest{
		Model: "whisper-1", Filename: "a.mp3", File: []byte("x"),
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// ---- TranscribeAudio: empty filename fallback ----

func TestTranscribeAudioEmptyFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// filename should fall back to "audio"
		if header.Filename != "audio" {
			http.Error(w, "bad filename: "+header.Filename, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"ok"}`))
	}))
	t.Cleanup(server.Close)

	p := New(server.URL)
	resp, err := p.TranscribeAudio(context.Background(), testKey(), &providers.AudioTranscriptionRequest{
		Model:    "whisper-1",
		Filename: "", // empty → should use "audio"
		File:     []byte("bytes"),
	})
	if err != nil {
		t.Fatalf("TranscribeAudio: %v", err)
	}
	if resp.Text != "ok" {
		t.Fatalf("text = %q", resp.Text)
	}
}

// ---- Speech: newJSONRequest error via expired deadline ----

func TestSpeechExpiredDeadline(t *testing.T) {
	p := New("http://127.0.0.1:1")
	deadline := time.Now().Add(-time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	_, _, err := p.Speech(ctx, testKey(), &providers.SpeechRequest{Model: "m", Input: "x", Voice: "alloy"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---- Speech: do() network error ----

func TestSpeechDoError(t *testing.T) {
	p := New("http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, _, err := p.Speech(ctx, testKey(), &providers.SpeechRequest{Model: "m", Input: "x", Voice: "alloy"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---- buildTranscriptionMultipart: all optional fields set ----

func TestBuildTranscriptionMultipartAllFields(t *testing.T) {
	req := &providers.AudioTranscriptionRequest{
		Model:          "whisper-1",
		Language:       "en",
		Prompt:         "hint",
		ResponseFormat: "json",
		Temperature:    "0.5",
		Filename:       "clip.wav",
		File:           []byte("audiodata"),
	}
	body, ct, err := buildTranscriptionMultipart(req)
	if err != nil {
		t.Fatalf("buildTranscriptionMultipart: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("empty body")
	}
	if ct == "" {
		t.Fatal("empty content-type")
	}
}
