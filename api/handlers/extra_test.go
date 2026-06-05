package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
)

// fakeExtraEngine satisfies both handlers.InferenceEngine (so it can be set as
// ServerConfig.InferenceEngine) and handlers.ExtraEngine.
type fakeExtraEngine struct {
	fakeEngine
	embeddings  *providers.EmbeddingsResponse
	images      *providers.ImagesResponse
	audio       *providers.AudioResponse
	speech      []byte
	speechCT    string
	extraErr    error
	gotEmbedReq *providers.EmbeddingsRequest
	gotAudioReq *providers.AudioTranscriptionRequest
}

func (f *fakeExtraEngine) Embeddings(ctx context.Context, req *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error) {
	f.gotEmbedReq = req
	if f.extraErr != nil {
		return nil, f.extraErr
	}
	return f.embeddings, nil
}

func (f *fakeExtraEngine) GenerateImages(ctx context.Context, req *providers.ImagesRequest) (*providers.ImagesResponse, error) {
	if f.extraErr != nil {
		return nil, f.extraErr
	}
	return f.images, nil
}

func (f *fakeExtraEngine) TranscribeAudio(ctx context.Context, req *providers.AudioTranscriptionRequest) (*providers.AudioResponse, error) {
	f.gotAudioReq = req
	if f.extraErr != nil {
		return nil, f.extraErr
	}
	return f.audio, nil
}

func (f *fakeExtraEngine) Speech(ctx context.Context, req *providers.SpeechRequest) ([]byte, string, error) {
	if f.extraErr != nil {
		return nil, "", f.extraErr
	}
	return f.speech, f.speechCT, nil
}

func TestEmbeddingsEndpoint(t *testing.T) {
	engine := &fakeExtraEngine{
		embeddings: &providers.EmbeddingsResponse{
			Object: "list",
			Model:  "text-embedding-3-small",
			Data:   []providers.EmbeddingData{{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2}}},
			Usage:  &providers.EmbeddingUsage{PromptTokens: 4, TotalTokens: 4},
		},
	}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/embeddings", `{"model":"text-embedding-3-small","input":"hello"}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body=%s", resp.StatusCode, body)
	}
	var decoded providers.EmbeddingsResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode: %v; body=%s", err, body)
	}
	if decoded.Object != "list" || len(decoded.Data) != 1 || len(decoded.Data[0].Embedding) != 2 {
		t.Fatalf("decoded = %+v", decoded)
	}
	if engine.gotEmbedReq == nil || engine.gotEmbedReq.Model != "text-embedding-3-small" {
		t.Fatalf("engine got = %+v", engine.gotEmbedReq)
	}
}

func TestEmbeddingsUnsupportedReturns501(t *testing.T) {
	engine := &fakeExtraEngine{extraErr: proxy.ErrCapabilityUnsupported}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/embeddings", `{"model":"m","input":"x"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", resp.StatusCode, body)
	}
}

func TestEmbeddingsRequiresAuth(t *testing.T) {
	engine := &fakeExtraEngine{embeddings: &providers.EmbeddingsResponse{Object: "list"}}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/embeddings", bytes.NewBufferString(`{"model":"m","input":"x"}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header.
	httpResp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", httpResp.StatusCode)
	}
}

func TestEmbeddingsMissingModel(t *testing.T) {
	engine := &fakeExtraEngine{}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})
	resp, body := postJSON(t, baseURL+"/v1/embeddings", `{"input":"x"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}
}

func TestImagesEndpoint(t *testing.T) {
	engine := &fakeExtraEngine{
		images: &providers.ImagesResponse{Created: 1, Data: []providers.ImageData{{URL: "https://img/1.png"}}},
	}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/images/generations", `{"model":"dall-e-3","prompt":"a cat"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body=%s", resp.StatusCode, body)
	}
	var decoded providers.ImagesResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded.Data) != 1 || decoded.Data[0].URL != "https://img/1.png" {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestSpeechEndpointReturnsAudio(t *testing.T) {
	engine := &fakeExtraEngine{speech: []byte("MP3DATA"), speechCT: "audio/mpeg"}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/audio/speech", `{"model":"tts-1","input":"hi","voice":"alloy"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "audio/mpeg" {
		t.Fatalf("content-type = %q", got)
	}
	if string(body) != "MP3DATA" {
		t.Fatalf("body = %q", body)
	}
}

func TestSpeechUnsupportedReturns501(t *testing.T) {
	engine := &fakeExtraEngine{extraErr: proxy.ErrCapabilityUnsupported}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})
	resp, body := postJSON(t, baseURL+"/v1/audio/speech", `{"model":"m","input":"x","voice":"alloy"}`, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", resp.StatusCode, body)
	}
}

func TestAudioTranscriptionEndpoint(t *testing.T) {
	engine := &fakeExtraEngine{audio: &providers.AudioResponse{Text: "hello world"}}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	body, contentType := buildTranscriptionForm(t, "whisper-1", "speech.mp3", []byte("audio-bytes"))
	resp, data := postMultipart(t, baseURL+"/v1/audio/transcriptions", contentType, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body=%s", resp.StatusCode, data)
	}
	var decoded providers.AudioResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Text != "hello world" {
		t.Fatalf("decoded = %+v", decoded)
	}
	if engine.gotAudioReq == nil || engine.gotAudioReq.Model != "whisper-1" || engine.gotAudioReq.Filename != "speech.mp3" {
		t.Fatalf("engine got = %+v", engine.gotAudioReq)
	}
	if string(engine.gotAudioReq.File) != "audio-bytes" {
		t.Fatalf("file = %q", engine.gotAudioReq.File)
	}
}

func TestAudioTranscriptionMissingFile(t *testing.T) {
	engine := &fakeExtraEngine{}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", "whisper-1")
	_ = w.Close()
	resp, data := postMultipart(t, baseURL+"/v1/audio/transcriptions", w.FormDataContentType(), buf.Bytes())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, data)
	}
}

func buildTranscriptionForm(t *testing.T, model, filename string, file []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", model)
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

func postMultipart(t *testing.T, url, contentType string, body []byte) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer g0r_valid")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		t.Fatalf("read body: %v", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, data
}
