package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/streaming"
	"github.com/valyala/fasthttp"
)

type Response = streaming.Response
type ResponseEvent = streaming.ResponseEvent

type ResponseRequest struct {
	Model           string          `json:"model"`
	Input           []ResponseInput `json:"input,omitempty"`
	Instructions    *string         `json:"instructions,omitempty"`
	Stream          *bool           `json:"stream,omitempty"`
	Temperature     *float64        `json:"temperature,omitempty"`
	TopP            *float64        `json:"top_p,omitempty"`
	MaxOutputTokens *int            `json:"max_output_tokens,omitempty"`
	Tools           []ResponseTool  `json:"tools,omitempty"`
	Text            any             `json:"text,omitempty"`
	ToolChoice      any             `json:"tool_choice,omitempty"`
}

type ResponseInput struct {
	Role    string            `json:"role"`
	Content []ResponseContent `json:"content"`
}

type ResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResponseTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

func (p *OpenAIProvider) Responses(ctx context.Context, key providers.Key, req *ResponseRequest) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("openai responses: nil request")
	}

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/responses", key, req)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai responses: %w", err)
	}
	defer fasthttp.ReleaseResponse(resp)

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, mapError(resp)
	}

	var response Response
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return nil, fmt.Errorf("parse openai responses response: %w", err)
	}
	return &response, nil
}

func (p *OpenAIProvider) ResponsesStream(ctx context.Context, key providers.Key, req *ResponseRequest) (<-chan ResponseEvent, error) {
	if req == nil {
		return nil, fmt.Errorf("openai responses stream: nil request")
	}

	streamReq := *req
	stream := true
	streamReq.Stream = &stream

	httpReq, err := p.newJSONRequest(fasthttp.MethodPost, "/v1/responses", key, &streamReq)
	if err != nil {
		return nil, err
	}
	defer fasthttp.ReleaseRequest(httpReq)

	resp, err := p.do(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai responses stream: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		defer fasthttp.ReleaseResponse(resp)
		return nil, mapError(resp)
	}

	events := make(chan ResponseEvent)
	body := append([]byte(nil), resp.Body()...)
	fasthttp.ReleaseResponse(resp)
	go func() {
		defer close(events)
		parseResponsesSSE(bytes.NewReader(body), events)
	}()
	return events, nil
}

func parseResponsesSSE(body io.Reader, events chan<- ResponseEvent) {
	parseSSEData(body, func(data string) bool {
		var event ResponseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return false
		}
		events <- event
		return false
	})
}

func parseSSEData(body io.Reader, handle func(string) bool) {
	scanner := bufio.NewScanner(body)
	var dataLines []string

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if handleResponsesData(dataLines, handle) {
				return
			}
			dataLines = nil
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if len(dataLines) > 0 {
		handleResponsesData(dataLines, handle)
	}
}

func handleResponsesData(dataLines []string, handle func(string) bool) bool {
	if len(dataLines) == 0 {
		return false
	}

	data := strings.Join(dataLines, "\n")
	if data == "[DONE]" {
		return true
	}
	return handle(data)
}
