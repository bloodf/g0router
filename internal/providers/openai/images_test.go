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

// TestImageGenerationSuccess verifies ImageGeneration posts JSON and decodes
// the bare ImageGenerationResponse.
func TestImageGenerationSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %q, want /v1/images/generations", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("auth = %q, want Bearer k", got)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"created":123,"data":[{"url":"https://img/1.png"}]}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.ImageGeneration(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.ImageGenerationRequest{
		Prompt: "a cat", Model: "dall-e-3",
	})
	if perr != nil {
		t.Fatalf("ImageGeneration error: %v", perr.Message)
	}
	if resp.Created != 123 || len(resp.Data) != 1 || resp.Data[0].URL != "https://img/1.png" {
		t.Errorf("resp = %+v, want created=123 one url", resp)
	}
}

// TestImageGenerationUpstreamError verifies non-200 surfaces a *ProviderError.
func TestImageGenerationUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"boom","type":"api_error"}}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	_, perr := p.ImageGeneration(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.ImageGenerationRequest{Prompt: "x"})
	if perr == nil {
		t.Fatal("expected *ProviderError, got nil")
	}
	if perr.StatusCode != 500 {
		t.Errorf("status = %d, want 500", perr.StatusCode)
	}
}

// TestImageEditSendsMultipart verifies ImageEdit builds a multipart body whose
// image part round-trips and decodes the ImageGenerationResponse (ESC-MULTIPART).
func TestImageEditSendsMultipart(t *testing.T) {
	imgBytes := []byte("\x89PNG\r\n\x1a\n fake-png")
	maskBytes := []byte("\x89PNG mask")
	var gotImage, gotMask []byte
	var gotPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" {
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
		if fhs := form.File["image"]; len(fhs) == 1 {
			f, _ := fhs[0].Open()
			gotImage, _ = io.ReadAll(f)
		}
		if fhs := form.File["mask"]; len(fhs) == 1 {
			f, _ := fhs[0].Open()
			gotMask, _ = io.ReadAll(f)
		}
		if pv := form.Value["prompt"]; len(pv) == 1 {
			gotPrompt = pv[0]
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"created":1,"data":[{"b64_json":"abc"}]}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.ImageEdit(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.ImageEditRequest{
		Image: imgBytes, Mask: maskBytes, Prompt: "make it blue", Model: "dall-e-2",
	})
	if perr != nil {
		t.Fatalf("ImageEdit error: %v", perr.Message)
	}
	if string(gotImage) != string(imgBytes) {
		t.Errorf("image part = %q, want round-trip", gotImage)
	}
	if string(gotMask) != string(maskBytes) {
		t.Errorf("mask part = %q, want round-trip", gotMask)
	}
	if gotPrompt != "make it blue" {
		t.Errorf("prompt = %q, want 'make it blue'", gotPrompt)
	}
	if len(resp.Data) != 1 || resp.Data[0].B64JSON != "abc" {
		t.Errorf("resp = %+v, want one b64 'abc'", resp)
	}
}

// TestImageVariationSendsMultipart verifies ImageVariation builds a multipart
// body whose image part round-trips and decodes the response.
func TestImageVariationSendsMultipart(t *testing.T) {
	imgBytes := []byte("\x89PNG variation-src")
	var gotImage []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/variations" {
			t.Errorf("path = %q", r.URL.Path)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("inbound Content-Type = %q, want multipart/form-data", ct)
		}
		_, params, _ := mime.ParseMediaType(ct)
		mr := multipart.NewReader(r.Body, params["boundary"])
		form, err := mr.ReadForm(1 << 20)
		if err != nil {
			t.Fatalf("read form: %v", err)
		}
		if fhs := form.File["image"]; len(fhs) == 1 {
			f, _ := fhs[0].Open()
			gotImage, _ = io.ReadAll(f)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"created":2,"data":[{"url":"https://img/v.png"}]}`))
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	resp, perr := p.ImageVariation(&schemas.GatewayContext{}, schemas.Key{Value: "k"}, &schemas.ImageVariationRequest{
		Image: imgBytes, Model: "dall-e-2",
	})
	if perr != nil {
		t.Fatalf("ImageVariation error: %v", perr.Message)
	}
	if string(gotImage) != string(imgBytes) {
		t.Errorf("image part = %q, want round-trip", gotImage)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://img/v.png" {
		t.Errorf("resp = %+v, want one url", resp)
	}
}

// TestImageGenerationStreamAbortsOnMalformedChunk verifies AUD-045 for the
// image-generation stream.
func TestImageGenerationStreamAbortsOnMalformedChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"i1\"}\n\n")
		io.WriteString(w, "data: not-json{\n\n")
		io.WriteString(w, "data: {\"id\":\"i2\"}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.ImageGenerationStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "k"}, &schemas.ImageGenerationRequest{Prompt: "x"})
	if perr != nil {
		t.Fatalf("ImageGenerationStream error: %v", perr.Message)
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
