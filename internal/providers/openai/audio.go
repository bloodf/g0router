package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// Speech sends a text-to-speech request to POST /v1/audio/speech and returns
// the raw synthesized audio bytes with the upstream Content-Type. Unlike the
// other endpoints the success body is binary, not JSON, so it is copied
// verbatim rather than decoded (ESC-SPEECH-BYTES).
func (p *Provider) Speech(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.SpeechRequest) (*schemas.SpeechResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/audio/speech")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if err := utils.SetJSONBody(req, request); err != nil {
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("build request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.provider),
				ModelRequested: request.Model,
				RequestType:    "speech",
				StatusCode:     0,
			},
		}
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "speech",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "speech",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	// Copy the binary body and Content-Type; the bytes must outlive the pooled
	// response, so clone rather than alias resp.Body().
	audio := append([]byte(nil), resp.Body()...)
	return &schemas.SpeechResponse{
		Audio:       audio,
		ContentType: string(resp.Header.ContentType()),
	}, nil
}

// SpeechStream sends a streaming text-to-speech request and returns a channel
// of SSE chunks (ESC-SPEECH-STREAM: SSE-drain template; if the upstream emits
// SSE frames they pass through, mirroring TextCompletionStream).
func (p *Provider) SpeechStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.SpeechRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.baseURL + "/v1/audio/speech")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	streamReq := *request
	if err := utils.SetJSONBody(req, &streamReq); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, &schemas.ProviderError{
			Message:    fmt.Sprintf("build request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.provider),
				ModelRequested: request.Model,
				RequestType:    "speech_stream",
				StatusCode:     0,
			},
		}
	}

	return p.streamSSE(ctx, postHookRunner, req, resp, request.Model, "speech_stream")
}

// Transcription sends an audio transcription request to
// POST /v1/audio/transcriptions as multipart/form-data (ESC-MULTIPART) and
// decodes the JSON TranscriptionResponse.
func (p *Provider) Transcription(ctx *schemas.GatewayContext, key schemas.Key, request *schemas.TranscriptionRequest) (*schemas.TranscriptionResponse, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	defer p.client.ReleaseRequest(req)
	resp := p.client.AcquireResponse()
	defer p.client.ReleaseResponse(resp)

	req.SetRequestURI(p.baseURL + "/v1/audio/transcriptions")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if perr := p.setTranscriptionBody(req, request, false); perr != nil {
		return nil, perr
	}

	if err := p.client.Do(req, resp); err != nil {
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "transcription",
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "transcription",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	var result schemas.TranscriptionResponse
	if err := utils.ReadJSONBody(resp, &result); err != nil {
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: request.Model,
			RequestType:    "transcription",
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}
	return &result, nil
}

// TranscriptionStream sends a streaming transcription request (multipart body
// with stream=true) and returns a channel of SSE chunks.
func (p *Provider) TranscriptionStream(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, key schemas.Key, request *schemas.TranscriptionRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	req := p.client.AcquireRequest()
	resp := p.client.AcquireResponse()

	req.SetRequestURI(p.baseURL + "/v1/audio/transcriptions")
	req.Header.SetMethod(fasthttp.MethodPost)
	utils.SetAuthHeader(req, key.Value)

	if perr := p.setTranscriptionBody(req, request, true); perr != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, perr
	}

	return p.streamSSE(ctx, postHookRunner, req, resp, request.Model, "transcription_stream")
}

// setTranscriptionBody builds the outbound multipart/form-data body for a
// transcription request from the already-parsed schema fields, with an
// explicit field whitelist (ESC-MULTIPART). When stream is true a stream=true
// form value is added.
func (p *Provider) setTranscriptionBody(req *fasthttp.Request, request *schemas.TranscriptionRequest, stream bool) *schemas.ProviderError {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fail := func(err error) *schemas.ProviderError {
		return &schemas.ProviderError{
			Message:    fmt.Sprintf("build multipart request: %v", err),
			Type:       "invalid_request_error",
			StatusCode: 0,
			Meta: schemas.ErrorMeta{
				Provider:       string(p.provider),
				ModelRequested: request.Model,
				RequestType:    "transcription",
				StatusCode:     0,
			},
		}
	}

	fw, err := mw.CreateFormFile("file", "audio")
	if err != nil {
		return fail(err)
	}
	if _, err := fw.Write(request.File); err != nil {
		return fail(err)
	}
	if err := mw.WriteField("model", request.Model); err != nil {
		return fail(err)
	}
	if request.Language != nil {
		if err := mw.WriteField("language", *request.Language); err != nil {
			return fail(err)
		}
	}
	if request.Prompt != nil {
		if err := mw.WriteField("prompt", *request.Prompt); err != nil {
			return fail(err)
		}
	}
	if request.ResponseFormat != nil {
		if err := mw.WriteField("response_format", *request.ResponseFormat); err != nil {
			return fail(err)
		}
	}
	if request.Temperature != nil {
		if err := mw.WriteField("temperature", fmt.Sprintf("%v", *request.Temperature)); err != nil {
			return fail(err)
		}
	}
	for _, g := range request.TimestampGranularities {
		if err := mw.WriteField("timestamp_granularities[]", g); err != nil {
			return fail(err)
		}
	}
	if stream {
		if err := mw.WriteField("stream", "true"); err != nil {
			return fail(err)
		}
	}
	if err := mw.Close(); err != nil {
		return fail(err)
	}

	req.Header.SetContentType(mw.FormDataContentType())
	req.SetBody(buf.Bytes())
	return nil
}

// streamSSE issues req, validates the response status, then drains the upstream
// body as SSE frames onto a channel. It mirrors TextCompletionStream so all the
// streaming endpoints share one terminator/error/post-hook contract
// (AUD-045/046/047). It takes ownership of req and resp.
func (p *Provider) streamSSE(ctx *schemas.GatewayContext, postHookRunner schemas.PostHookRunner, req *fasthttp.Request, resp *fasthttp.Response, model, requestType string) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	if err := p.client.Do(req, resp); err != nil {
		p.client.ReleaseRequest(req)
		p.client.ReleaseResponse(resp)
		return nil, p.errorConverter.Convert(0, []byte(err.Error()), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: model,
			RequestType:    requestType,
			StatusCode:     0,
			RawBody:        []byte(err.Error()),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		defer p.client.ReleaseRequest(req)
		defer p.client.ReleaseResponse(resp)
		return nil, p.errorConverter.Convert(resp.StatusCode(), resp.Body(), schemas.ErrorMeta{
			Provider:       string(p.provider),
			ModelRequested: model,
			RequestType:    requestType,
			StatusCode:     resp.StatusCode(),
			RawBody:        resp.Body(),
		})
	}

	p.client.ReleaseRequest(req)

	ch := make(chan *schemas.StreamChunk, 16)
	go func() {
		defer close(ch)
		defer p.client.ReleaseResponse(resp)

		body := bytes.NewReader(resp.Body())
		scanner := utils.NewSSEScanner(body)
		for {
			line, err := scanner.Scan()
			if err != nil {
				if err == io.EOF {
					return
				}
				ch <- streamError(fmt.Sprintf("read stream: %v", err))
				return
			}
			if line == "[DONE]" {
				return
			}
			var chunk schemas.StreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				ch <- streamError(fmt.Sprintf("decode stream chunk: %v", err))
				return
			}
			ch <- &chunk
			if postHookRunner != nil {
				if err := postHookRunner.Run(ctx, &chunk); err != nil {
					ch <- streamError(fmt.Sprintf("post hook: %v", err))
					return
				}
			}
		}
	}()

	return ch, nil
}
