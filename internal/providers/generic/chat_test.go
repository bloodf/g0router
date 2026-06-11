package generic

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

type fakePostHookRunner struct {
	err error
}

func (f *fakePostHookRunner) Run(ctx *schemas.GatewayContext, response any) error {
	return f.err
}

func TestGenericChatURL(t *testing.T) {
	p, err := New("deepseek")
	if err != nil {
		t.Fatalf("New(deepseek) error: %v", err)
	}
	got := p.chatURL()
	want := "https://api.deepseek.com/chat/completions"
	if got != want {
		t.Errorf("chatURL() = %q, want %q", got, want)
	}
}

func TestGenericChatBearerAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","object":"chat.completion","created":1,"model":"m","choices":[]}`))
	}))
	defer srv.Close()

	p, _ := New("groq")
	p.config.BaseURL = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "llama"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want \"Bearer test-key\"", gotAuth)
	}
}

func TestGenericChatCustomHeaders(t *testing.T) {
	var gotReferer, gotTitle string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("HTTP-Referer")
		gotTitle = r.Header.Get("X-Title")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"1","object":"chat.completion","created":1,"model":"m","choices":[]}`))
	}))
	defer srv.Close()

	p, _ := New("openrouter")
	p.config.BaseURL = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "claude"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if gotReferer != "https://endpoint-proxy.local" {
		t.Errorf("HTTP-Referer = %q, want \"https://endpoint-proxy.local\"", gotReferer)
	}
	if gotTitle != "Endpoint Proxy" {
		t.Errorf("X-Title = %q, want \"Endpoint Proxy\"", gotTitle)
	}
}

func TestGenericChatErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"server error","type":"api_error"}}`))
	}))
	defer srv.Close()

	p, _ := New("deepseek")
	p.config.BaseURL = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "deepseek-chat"})
	if perr == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if perr.Meta.Provider != "deepseek" {
		t.Errorf("Meta.Provider = %q, want \"deepseek\"", perr.Meta.Provider)
	}
}

func TestGenericStreamParsesSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"c1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"}}]}\n\n")
		io.WriteString(w, "data: {\"id\":\"c2\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p, _ := New("groq")
	p.config.BaseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "llama"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var content int
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("unexpected error chunk: %v", chunk.Error.Message)
		}
		content++
	}
	if content != 2 {
		t.Errorf("content chunks = %d, want 2", content)
	}
}

func TestGenericStreamMalformedChunkInBandError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"c1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"a\"}}]}\n\n")
		io.WriteString(w, "data: not-json{\n\n")
		io.WriteString(w, "data: {\"id\":\"c2\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"b\"}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p, _ := New("groq")
	p.config.BaseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "llama"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
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
		t.Errorf("content chunks = %d, want 1 (stream must abort at malformed chunk)", content)
	}
	if errChunks != 1 {
		t.Errorf("error chunks = %d, want 1 (abort must be distinguishable from clean EOF)", errChunks)
	}
}

func TestGenericStreamPostHookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"id\":\"c1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p, _ := New("groq")
	p.config.BaseURL = srv.URL

	hook := &fakePostHookRunner{err: errors.New("policy denied")}
	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, hook, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "llama"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var content, errChunks int
	for chunk := range ch {
		if chunk.Error != nil {
			errChunks++
			if chunk.Error.Type != "stream_error" {
				t.Errorf("error type = %q, want \"stream_error\"", chunk.Error.Type)
			}
			if !strings.Contains(chunk.Error.Message, "post hook") {
				t.Errorf("error message = %q, want it to contain \"post hook\"", chunk.Error.Message)
			}
			continue
		}
		content++
	}
	if content != 1 {
		t.Errorf("content chunks = %d, want 1", content)
	}
	if errChunks != 1 {
		t.Errorf("error chunks = %d, want 1", errChunks)
	}
}
