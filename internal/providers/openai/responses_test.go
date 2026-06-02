package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponsesCreatePostsRequest(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotRequest ResponseRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	resp, err := provider.Responses(context.Background(), testKey(), &ResponseRequest{
		Model: "gpt-4o-mini",
		Input: []ResponseInput{
			{
				Role: "user",
				Content: []ResponseContent{
					{Type: "input_text", Text: "Hello"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Responses: %v", err)
	}

	if gotPath != "/v1/responses" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotRequest.Model != "gpt-4o-mini" || len(gotRequest.Input) != 1 {
		t.Fatalf("request = %+v", gotRequest)
	}
	if resp.ID != "resp_123" || resp.OutputText != "Hello back" {
		t.Fatalf("response = %+v", resp)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 7 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestResponsesStreamParsesTypedEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"type":"response.output_text.delta","delta":"Hel","sequence_number":1}`,
			"",
			`data: {"type":"response.output_text.delta","delta":"lo","sequence_number":2}`,
			"",
			`data: {"type":"response.completed","response":{"id":"resp_123","object":"response","created_at":1700000000,"model":"gpt-4o-mini","status":"completed","output_text":"Hello"},"sequence_number":3}`,
			"",
			"data: [DONE]",
			"",
		}, "\n")))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	events, err := provider.ResponsesStream(context.Background(), testKey(), &ResponseRequest{Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("ResponsesStream: %v", err)
	}

	var got []ResponseEvent
	for event := range events {
		got = append(got, event)
	}
	if len(got) != 3 {
		t.Fatalf("events len = %d", len(got))
	}
	if got[0].Type != "response.output_text.delta" || got[0].Delta != "Hel" {
		t.Fatalf("first event = %+v", got[0])
	}
	if got[2].Response == nil || got[2].Response.ID != "resp_123" {
		t.Fatalf("completed event = %+v", got[2])
	}
}

const responseJSON = `{
  "id": "resp_123",
  "object": "response",
  "created_at": 1700000000,
  "model": "gpt-4o-mini",
  "status": "completed",
  "output_text": "Hello back",
  "output": [
    {
      "type": "message",
      "role": "assistant",
      "content": [
        {"type": "output_text", "text": "Hello back"}
      ]
    }
  ],
  "usage": {
    "input_tokens": 4,
    "output_tokens": 3,
    "total_tokens": 7
  }
}`
