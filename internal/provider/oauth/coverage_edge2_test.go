package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func deepseekSession() AuthSession {
	return AuthSession{Provider: deepseekProviderID, SessionID: "user@x"}
}

func TestDeepSeekExchangeErrorPaths(t *testing.T) {
	bad := badStatusServer()
	defer bad.Close()
	flow := NewDeepSeekFlow(DeepSeekConfig{ClientID: "c", TokenURL: bad.URL, HTTPClient: bad.Client()})
	if _, err := flow.Exchange(context.Background(), deepseekSession(), "pw"); err == nil {
		t.Fatal("bad status: want error")
	}

	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"bearer"}`))
	}))
	defer missing.Close()
	flow2 := NewDeepSeekFlow(DeepSeekConfig{ClientID: "c", TokenURL: missing.URL, HTTPClient: missing.Client()})
	if _, err := flow2.Exchange(context.Background(), deepseekSession(), "pw"); err == nil {
		t.Fatal("missing access token: want error")
	}

	decodeErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("notjson"))
	}))
	defer decodeErr.Close()
	flow3 := NewDeepSeekFlow(DeepSeekConfig{ClientID: "c", TokenURL: decodeErr.URL, HTTPClient: decodeErr.Client()})
	if _, err := flow3.Exchange(context.Background(), deepseekSession(), "pw"); err == nil {
		t.Fatal("decode error: want error")
	}
}

func TestCursorPollSuccessAnd404AndBadStatus(t *testing.T) {
	// Success.
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accessToken":"acc","refreshToken":"ref"}`))
	}))
	defer ok.Close()
	flow := NewCursorFlowWithConfig(CursorConfig{PollURL: ok.URL, HTTPClient: ok.Client()})
	res, err := flow.Poll(context.Background(), AuthSession{Provider: cursorProviderID, SessionID: "uuid.verifier"})
	if err != nil {
		t.Fatalf("poll success: %v", err)
	}
	if res.Status != PollStatusComplete || res.Token == nil || res.Token.AccessToken != "acc" {
		t.Fatalf("poll result = %+v", res)
	}

	// 404 -> pending.
	notfound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notfound.Close()
	flow2 := NewCursorFlowWithConfig(CursorConfig{PollURL: notfound.URL, HTTPClient: notfound.Client()})
	res2, err := flow2.Poll(context.Background(), AuthSession{Provider: cursorProviderID, SessionID: "uuid.verifier"})
	if err != nil || res2.Status != PollStatusPending {
		t.Fatalf("poll 404: status=%q err=%v", res2.Status, err)
	}

	// Bad status -> error.
	bad := badStatusServer()
	defer bad.Close()
	flow3 := NewCursorFlowWithConfig(CursorConfig{PollURL: bad.URL, HTTPClient: bad.Client()})
	if _, err := flow3.Poll(context.Background(), AuthSession{Provider: cursorProviderID, SessionID: "uuid.verifier"}); err == nil {
		t.Fatal("poll bad status: want error")
	}

	// Decode error (missing access token).
	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"refreshToken":"r"}`))
	}))
	defer missing.Close()
	flow4 := NewCursorFlowWithConfig(CursorConfig{PollURL: missing.URL, HTTPClient: missing.Client()})
	if _, err := flow4.Poll(context.Background(), AuthSession{Provider: cursorProviderID, SessionID: "uuid.verifier"}); err == nil {
		t.Fatal("poll missing token: want error")
	}
}

func TestCursorPollNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	flow := NewCursorFlowWithConfig(CursorConfig{PollURL: url, HTTPClient: client})
	if _, err := flow.Poll(context.Background(), AuthSession{Provider: cursorProviderID, SessionID: "uuid.verifier"}); err == nil {
		t.Fatal("poll network error: want error")
	}
}

func TestCursorRefreshNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	flow := NewCursorFlowWithConfig(CursorConfig{RefreshURL: url, HTTPClient: client})
	if _, err := flow.Refresh(context.Background(), "tok"); err == nil {
		t.Fatal("refresh network error: want error")
	}
}

func TestStartNetworkErrors(t *testing.T) {
	// Cursor Start parse-login-url error.
	flow := NewCursorFlowWithConfig(CursorConfig{LoginURL: "http://\x7f bad", NewUUID: func() (string, error) { return "uuid", nil }})
	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("cursor start bad login url: want error")
	}

	// Codex Start network error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	codex := NewCodexFlow(CodexFlowConfig{ClientID: "c", DeviceCodeURL: url, HTTPClient: client})
	if _, err := codex.Start(context.Background()); err == nil {
		t.Fatal("codex start network error: want error")
	}
}

func TestExchangeNetworkErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()

	anthropic := NewAnthropicFlowWithConfig(AnthropicConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := anthropic.Exchange(context.Background(), AuthSession{Provider: anthropic.ProviderID(), SessionID: "s.v"}, "code"); err == nil {
		t.Fatal("anthropic exchange network error: want error")
	}

	gemini := NewGeminiFlow(GeminiConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := gemini.Exchange(context.Background(), AuthSession{Provider: gemini.ProviderID(), SessionID: "s.v"}, "code"); err == nil {
		t.Fatal("gemini exchange network error: want error")
	}

	xai := NewXAIFlow(XAIConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := xai.Exchange(context.Background(), AuthSession{Provider: xai.ProviderID(), SessionID: "s.v"}, "code"); err == nil {
		t.Fatal("xai exchange network error: want error")
	}

	deepseek := NewDeepSeekFlow(DeepSeekConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := deepseek.Exchange(context.Background(), deepseekSession(), "pw"); err == nil {
		t.Fatal("deepseek exchange network error: want error")
	}
}

func TestGitHubPollComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a","token_type":"bearer","scope":"x"}`))
	}))
	defer server.Close()
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	res, err := flow.Poll(context.Background(), AuthSession{SessionID: "d"})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if res.Status != PollStatusComplete || res.Token == nil || res.Token.AccessToken != "a" {
		t.Fatalf("res = %+v", res)
	}
}

func TestGitHubPollNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := flow.Poll(context.Background(), AuthSession{SessionID: "d"}); err == nil {
		t.Fatal("github poll network error: want error")
	}
}

func TestGitHubStartNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", DeviceCodeURL: url, HTTPClient: client})
	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("github start network error: want error")
	}
}

func TestPostFormDecodeErrorBranch(t *testing.T) {
	// Non-2xx with non-JSON body -> decode error response branch.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("plain text error"))
	}))
	defer server.Close()
	flow := NewCodexFlow(CodexFlowConfig{ClientID: "c", DeviceCodeURL: server.URL, HTTPClient: server.Client()})
	_, err := flow.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "start codex") {
		t.Fatalf("err = %v, want decode error response", err)
	}
}
