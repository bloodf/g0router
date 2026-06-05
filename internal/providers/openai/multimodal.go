package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

// Embeddings proxies POST /v1/embeddings to the upstream OpenAI API.
func (p *OpenAIProvider) Embeddings(ctx context.Context, key providers.Key, req *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("openai embeddings: nil request")
	}

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/embeddings", key, req)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded providers.EmbeddingsResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse openai embeddings response: %w", err)
	}
	return &decoded, nil
}

// GenerateImages proxies POST /v1/images/generations.
func (p *OpenAIProvider) GenerateImages(ctx context.Context, key providers.Key, req *providers.ImagesRequest) (*providers.ImagesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("openai images: nil request")
	}

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/images/generations", key, req)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai images: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded providers.ImagesResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse openai images response: %w", err)
	}
	return &decoded, nil
}

// TranscribeAudio proxies POST /v1/audio/transcriptions as a multipart upload.
func (p *OpenAIProvider) TranscribeAudio(ctx context.Context, key providers.Key, req *providers.AudioTranscriptionRequest) (*providers.AudioResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("openai transcription: nil request")
	}

	body, contentType, err := buildTranscriptionMultipart(req)
	if err != nil {
		return nil, err
	}

	httpReq := fasthttp.AcquireRequest()
	httpReq.Header.SetMethod(fasthttp.MethodPost)
	httpReq.SetRequestURI(p.baseURL + "/v1/audio/transcriptions")
	httpReq.Header.Set("Authorization", "Bearer "+key.Value)
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.SetBody(body)
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai transcription: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var decoded providers.AudioResponse
	if err := json.Unmarshal(resp.Body(), &decoded); err != nil {
		return nil, fmt.Errorf("parse openai transcription response: %w", err)
	}
	return &decoded, nil
}

// Speech proxies POST /v1/audio/speech and returns the raw audio bytes plus the
// upstream content type.
func (p *OpenAIProvider) Speech(ctx context.Context, key providers.Key, req *providers.SpeechRequest) ([]byte, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("openai speech: nil request")
	}

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/audio/speech", key, req)
	if err != nil {
		return nil, "", err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("openai speech: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, "", mapError(resp)
	}

	audio := append([]byte(nil), resp.Body()...)
	contentType := string(resp.Header.ContentType())
	return audio, contentType, nil
}

func buildTranscriptionMultipart(req *providers.AudioTranscriptionRequest) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	filename := req.Filename
	if filename == "" {
		filename = "audio"
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("create transcription file part: %w", err)
	}
	if _, err := part.Write(req.File); err != nil {
		return nil, "", fmt.Errorf("write transcription file part: %w", err)
	}

	fields := map[string]string{
		"model":           req.Model,
		"language":        req.Language,
		"prompt":          req.Prompt,
		"response_format": req.ResponseFormat,
		"temperature":     req.Temperature,
	}
	for name, value := range fields {
		if value == "" {
			continue
		}
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", fmt.Errorf("write transcription field %s: %w", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close transcription writer: %w", err)
	}
	return buf.Bytes(), writer.FormDataContentType(), nil
}
