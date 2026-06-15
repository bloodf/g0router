package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// fakeAudioResolver resolves any model to the embedded fake provider.
type fakeAudioResolver struct {
	prov schemas.Provider
}

func (r *fakeAudioResolver) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	return r.prov, schemas.Key{Provider: "openai"}, nil
}

// fakeAudioProvider records Speech / Transcription (+ stream) calls. It embeds
// fakeMessagesProvider to satisfy the full schemas.Provider interface.
type fakeAudioProvider struct {
	fakeMessagesProvider
	speechCalled        bool
	transcriptionCalled bool
	speechStreamCalled  bool
	capturedKey         schemas.Key
	capturedFile        []byte
	capturedModel       string
	speechResp          *schemas.SpeechResponse
	transcriptionResp   *schemas.TranscriptionResponse
	perr                *schemas.ProviderError
	streamCh            chan *schemas.StreamChunk
}

func (p *fakeAudioProvider) Speech(_ *schemas.GatewayContext, key schemas.Key, _ *schemas.SpeechRequest) (*schemas.SpeechResponse, *schemas.ProviderError) {
	p.speechCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.speechResp, nil
}

func (p *fakeAudioProvider) SpeechStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, key schemas.Key, _ *schemas.SpeechRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	p.speechStreamCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.streamCh, nil
}

func (p *fakeAudioProvider) Transcription(_ *schemas.GatewayContext, key schemas.Key, req *schemas.TranscriptionRequest) (*schemas.TranscriptionResponse, *schemas.ProviderError) {
	p.transcriptionCalled = true
	p.capturedKey = key
	p.capturedFile = req.File
	p.capturedModel = req.Model
	if p.perr != nil {
		return nil, p.perr
	}
	return p.transcriptionResp, nil
}

// buildMultipart constructs an in-memory multipart/form-data body. files maps
// field name -> bytes; values maps field name -> value. Returns body + content-type.
func buildMultipart(t *testing.T, files map[string][]byte, values map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for field, data := range files {
		fw, err := mw.CreateFormFile(field, field)
		if err != nil {
			t.Fatalf("create form file %s: %v", field, err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatalf("write form file %s: %v", field, err)
		}
	}
	for k, v := range values {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return buf.Bytes(), mw.FormDataContentType()
}

// assertNoEnvelope fails if the body carries a top-level data/error wrapper.
func assertNoEnvelope(t *testing.T, body []byte) {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		t.Fatalf("unmarshal top-level: %v", err)
	}
	if _, ok := top["data"]; ok {
		t.Error("response has top-level 'data' wrapper (admin envelope leaked)")
	}
	if _, ok := top["error"]; ok {
		t.Error("response has top-level 'error' key on success")
	}
}

// TestAudioSpeechSuccessReturnsRawBytes verifies /v1/audio/speech writes the
// raw audio bytes with the provider Content-Type and NO JSON envelope.
func TestAudioSpeechSuccessReturnsRawBytes(t *testing.T) {
	audio := []byte{0x49, 0x44, 0x33, 0x07, 0x08}
	prov := &fakeAudioProvider{speechResp: &schemas.SpeechResponse{Audio: audio, ContentType: "audio/mpeg"}}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/speech")
	ctx.Request.SetBody([]byte(`{"model":"tts-1","input":"hi","voice":"alloy"}`))
	h.Speech(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.speechCalled {
		t.Fatal("provider Speech not called")
	}
	if got := ctx.Response.Body(); !bytes.Equal(got, audio) {
		t.Errorf("body = %v, want raw audio %v", got, audio)
	}
	if ct := string(ctx.Response.Header.ContentType()); ct != "audio/mpeg" {
		t.Errorf("content-type = %q, want audio/mpeg", ct)
	}
	// The body must NOT be a JSON object with data/error keys.
	var top map[string]json.RawMessage
	if json.Unmarshal(ctx.Response.Body(), &top) == nil {
		if _, ok := top["data"]; ok {
			t.Error("speech body parsed as JSON with 'data' key — must be raw bytes")
		}
		if _, ok := top["error"]; ok {
			t.Error("speech body parsed as JSON with 'error' key — must be raw bytes")
		}
	}
}

// TestAudioSpeechFallbackContentType verifies an empty provider Content-Type
// falls back to application/octet-stream.
func TestAudioSpeechFallbackContentType(t *testing.T) {
	prov := &fakeAudioProvider{speechResp: &schemas.SpeechResponse{Audio: []byte("x"), ContentType: ""}}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/speech")
	ctx.Request.SetBody([]byte(`{"model":"tts-1","input":"hi","voice":"alloy"}`))
	h.Speech(&ctx)

	if ct := string(ctx.Response.Header.ContentType()); ct != "application/octet-stream" {
		t.Errorf("content-type = %q, want application/octet-stream", ct)
	}
}

// TestAudioSpeechInvalidJSON verifies a malformed body returns 400.
func TestAudioSpeechInvalidJSON(t *testing.T) {
	prov := &fakeAudioProvider{}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/speech")
	ctx.Request.SetBody([]byte(`{not json`))
	h.Speech(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.speechCalled {
		t.Fatal("provider should not be called on invalid JSON")
	}
}

// TestAudioSpeechStream verifies stream:true sets SSE content type and [DONE].
func TestAudioSpeechStream(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "s1"}
	close(ch)
	prov := &fakeAudioProvider{streamCh: ch}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/speech")
	ctx.Request.SetBody([]byte(`{"model":"tts-1","input":"hi","voice":"alloy","stream":true}`))
	h.Speech(&ctx)

	if !prov.speechStreamCalled {
		t.Fatal("provider SpeechStream not called")
	}
	if ct := string(ctx.Response.Header.ContentType()); ct != "text/event-stream" {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}
	if !contains(string(ctx.Response.Body()), "[DONE]") {
		t.Errorf("stream body missing [DONE]: %q", ctx.Response.Body())
	}
}

// TestAudioTranscriptionMultipartSuccess verifies a multipart upload reaches the
// provider and returns the bare TranscriptionResponse (no envelope).
func TestAudioTranscriptionMultipartSuccess(t *testing.T) {
	fileBytes := []byte("RIFF fake audio")
	prov := &fakeAudioProvider{transcriptionResp: &schemas.TranscriptionResponse{Text: "hello"}}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	body, ct := buildMultipart(t, map[string][]byte{"file": fileBytes}, map[string]string{"model": "whisper-1"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/transcriptions")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Transcription(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.transcriptionCalled {
		t.Fatal("provider Transcription not called")
	}
	if !bytes.Equal(prov.capturedFile, fileBytes) {
		t.Errorf("file = %q, want round-trip", prov.capturedFile)
	}
	if prov.capturedModel != "whisper-1" {
		t.Errorf("model = %q, want whisper-1", prov.capturedModel)
	}
	assertNoEnvelope(t, ctx.Response.Body())
	var resp schemas.TranscriptionResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Text != "hello" {
		t.Errorf("Text = %q, want hello", resp.Text)
	}
}

// TestAudioTranscriptionNonMultipart verifies a non-multipart request returns 400.
func TestAudioTranscriptionNonMultipart(t *testing.T) {
	prov := &fakeAudioProvider{}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/transcriptions")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"model":"whisper-1"}`))
	h.Transcription(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.transcriptionCalled {
		t.Fatal("provider should not be called for non-multipart")
	}
}

// TestAudioTranscriptionMissingFile verifies a multipart body without the file
// part returns 400.
func TestAudioTranscriptionMissingFile(t *testing.T) {
	prov := &fakeAudioProvider{}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	body, ct := buildMultipart(t, nil, map[string]string{"model": "whisper-1"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/transcriptions")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Transcription(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.transcriptionCalled {
		t.Fatal("provider should not be called when file missing")
	}
}

// TestAudioSpeechProviderError verifies a provider 501 is passed through.
func TestAudioSpeechProviderError(t *testing.T) {
	prov := &fakeAudioProvider{perr: &schemas.ProviderError{StatusCode: 501, Type: "not_implemented", Message: "speech not implemented"}}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/speech")
	ctx.Request.SetBody([]byte(`{"model":"tts-1","input":"hi","voice":"alloy"}`))
	h.Speech(&ctx)

	if ctx.Response.StatusCode() != 501 {
		t.Fatalf("status = %d, want 501", ctx.Response.StatusCode())
	}
}

// TestAudioSpeechMarshalFailure is N/A for speech (raw bytes), so this exercises
// the transcription JSON marshal-failure fallback to plain 500.
func TestAudioTranscriptionMarshalFailure(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("boom") }

	router := inference.NewRouter(translation.NewRegistry())
	h := NewAudioHandler(router)
	// Force a provider that returns success so the marshal path is reached.
	prov := &fakeAudioProvider{transcriptionResp: &schemas.TranscriptionResponse{Text: "x"}}
	h.router = &fakeAudioResolver{prov: prov}

	body, ct := buildMultipart(t, map[string][]byte{"file": []byte("a")}, map[string]string{"model": "whisper-1"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/transcriptions")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Transcription(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", got)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want 'internal error'", got)
	}
}

// TestAudioSpeechVKDenied verifies the x-g0-vk gate denies before dispatch.
func TestAudioSpeechVKDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-denied", &VKInfo{
		Key:      "vk-denied",
		Configs:  []VKProviderConfig{{Provider: "openai", AllowedModels: []string{"tts-1"}}},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 429, reason: "budget exhausted"})

	prov := &fakeAudioProvider{}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, quota))

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/speech")
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	ctx.Request.SetBody([]byte(`{"model":"tts-1","input":"hi","voice":"alloy"}`))
	h.Speech(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
	if prov.speechCalled {
		t.Fatal("provider Speech should not be called")
	}
}

// TestAudioSpeechVKPinned verifies pinned-key override reaches the provider.
func TestAudioSpeechVKPinned(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-pinned", &VKInfo{
		Key:      "vk-pinned",
		Configs:  []VKProviderConfig{{Provider: "openai", AllowedModels: []string{"tts-1"}, KeyIDs: []string{"conn-2"}}},
		IsActive: true,
	})

	prov := &fakeAudioProvider{speechResp: &schemas.SpeechResponse{Audio: []byte("a"), ContentType: "audio/mpeg"}}
	h := &AudioHandler{router: &fakeAudioResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, newFakeVKQuotaChecker()))
	h.SetVKPinnedResolver(&fakePinnedKeyResolver{connID: "conn-2", credential: "cred-2", ok: true})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/audio/speech")
	ctx.Request.Header.Set("x-g0-vk", "vk-pinned")
	ctx.Request.SetBody([]byte(`{"model":"tts-1","input":"hi","voice":"alloy"}`))
	h.Speech(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if prov.capturedKey.ID != "conn-2" || prov.capturedKey.Value != "cred-2" {
		t.Errorf("key = %+v, want conn-2/cred-2", prov.capturedKey)
	}
}
