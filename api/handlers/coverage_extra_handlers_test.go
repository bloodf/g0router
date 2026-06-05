package handlers

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

// extraCoverageEngine satisfies ExtraEngine for exercising the embeddings,
// images, and audio handler branches the external server tests do not reach.
type extraCoverageEngine struct {
	embeddings *providers.EmbeddingsResponse
	images     *providers.ImagesResponse
	audio      *providers.AudioResponse
	speech     []byte
	speechCT   string
	err        error
}

func (e *extraCoverageEngine) Embeddings(ctx context.Context, req *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error) {
	return e.embeddings, e.err
}

func (e *extraCoverageEngine) GenerateImages(ctx context.Context, req *providers.ImagesRequest) (*providers.ImagesResponse, error) {
	return e.images, e.err
}

func (e *extraCoverageEngine) TranscribeAudio(ctx context.Context, req *providers.AudioTranscriptionRequest) (*providers.AudioResponse, error) {
	return e.audio, e.err
}

func (e *extraCoverageEngine) Speech(ctx context.Context, req *providers.SpeechRequest) ([]byte, string, error) {
	return e.speech, e.speechCT, e.err
}

// --- Embeddings branches ---

func TestEmbeddingsNilEngineUnavailable(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"m","input":"x"}`, func(ctx *fasthttp.RequestCtx) {
		Embeddings(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestEmbeddingsInvalidJSON(t *testing.T) {
	engine := &extraCoverageEngine{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{bad`, func(ctx *fasthttp.RequestCtx) {
		Embeddings(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

// --- Images branches ---

func TestImagesNilEngineUnavailable(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"m","prompt":"p"}`, func(ctx *fasthttp.RequestCtx) {
		Images(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestImagesInvalidJSON(t *testing.T) {
	engine := &extraCoverageEngine{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{bad`, func(ctx *fasthttp.RequestCtx) {
		Images(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestImagesMissingPrompt(t *testing.T) {
	engine := &extraCoverageEngine{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"dall-e-3"}`, func(ctx *fasthttp.RequestCtx) {
		Images(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestImagesDispatchError(t *testing.T) {
	engine := &extraCoverageEngine{err: errors.New("upstream boom")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"dall-e-3","prompt":"a cat"}`, func(ctx *fasthttp.RequestCtx) {
		Images(ctx, engine)
	})
	if ctx.Response.StatusCode() == fasthttp.StatusOK {
		t.Fatalf("expected error status, got 200")
	}
}

// --- AudioTranscription branches ---

func TestAudioTranscriptionNilEngineUnavailable(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AudioTranscription(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAudioTranscriptionInvalidMultipart(t *testing.T) {
	engine := &extraCoverageEngine{}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.SetContentType("multipart/form-data; boundary=zzz")
	ctx.Request.SetBodyString("not a real multipart body")
	AudioTranscription(&ctx, engine)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAudioTranscriptionMissingModelField(t *testing.T) {
	engine := &extraCoverageEngine{}
	body, ct := buildAudioForm(t, "", "speech.mp3", []byte("bytes"))
	ctx := runMultipartHandler(t, ct, body, func(ctx *fasthttp.RequestCtx) {
		AudioTranscription(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAudioTranscriptionDispatchError(t *testing.T) {
	engine := &extraCoverageEngine{err: errors.New("upstream boom")}
	body, ct := buildAudioForm(t, "whisper-1", "speech.mp3", []byte("bytes"))
	ctx := runMultipartHandler(t, ct, body, func(ctx *fasthttp.RequestCtx) {
		AudioTranscription(ctx, engine)
	})
	if ctx.Response.StatusCode() == fasthttp.StatusOK {
		t.Fatalf("expected error status, got 200")
	}
}

// --- Speech branches ---

func TestSpeechNilEngineUnavailable(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"m","input":"x","voice":"alloy"}`, func(ctx *fasthttp.RequestCtx) {
		Speech(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestSpeechInvalidJSON(t *testing.T) {
	engine := &extraCoverageEngine{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{bad`, func(ctx *fasthttp.RequestCtx) {
		Speech(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestSpeechMissingField(t *testing.T) {
	engine := &extraCoverageEngine{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"tts-1"}`, func(ctx *fasthttp.RequestCtx) {
		Speech(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestSpeechDefaultContentType(t *testing.T) {
	engine := &extraCoverageEngine{speech: []byte("MP3"), speechCT: ""}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"model":"tts-1","input":"hi","voice":"alloy"}`, func(ctx *fasthttp.RequestCtx) {
		Speech(ctx, engine)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if got := string(ctx.Response.Header.ContentType()); got != "audio/mpeg" {
		t.Fatalf("content-type = %q, want audio/mpeg", got)
	}
	if string(body) != "MP3" {
		t.Fatalf("body = %q", body)
	}
}

func buildAudioForm(t *testing.T, model, filename string, file []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if model != "" {
		_ = w.WriteField("model", model)
	}
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(file); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes(), w.FormDataContentType()
}

func runMultipartHandler(t *testing.T, contentType string, body []byte, handler func(*fasthttp.RequestCtx)) *fasthttp.RequestCtx {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.SetContentType(contentType)
	ctx.Request.SetBody(body)
	handler(&ctx)
	return &ctx
}
