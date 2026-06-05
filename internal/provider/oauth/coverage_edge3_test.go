package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiStartBadAuthURL(t *testing.T) {
	flow := NewGeminiFlow(GeminiConfig{ClientID: "c", AuthURL: "http://\x7f bad"})
	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("gemini start bad auth url: want error")
	}
}

func TestXAIStartBadAuthURL(t *testing.T) {
	flow := NewXAIFlow(XAIConfig{ClientID: "c", AuthURL: "http://\x7f bad"})
	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("xai start bad auth url: want error")
	}
}

func TestGeminiAndXAIExchangeBadStatusAndDecode(t *testing.T) {
	bad := badStatusServer()
	defer bad.Close()
	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"bearer"}`))
	}))
	defer missing.Close()
	decodeErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("notjson"))
	}))
	defer decodeErr.Close()

	type builder func(tokenURL string, client *http.Client) refreshFlow
	builders := map[string]builder{
		"gemini": func(u string, c *http.Client) refreshFlow {
			return NewGeminiFlow(GeminiConfig{ClientID: "c", TokenURL: u, HTTPClient: c})
		},
		"xai": func(u string, c *http.Client) refreshFlow {
			return NewXAIFlow(XAIConfig{ClientID: "c", TokenURL: u, HTTPClient: c})
		},
	}
	type exchanger interface {
		Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error)
		ProviderID() ProviderID
	}
	for name, b := range builders {
		for label, server := range map[string]*httptest.Server{"badstatus": bad, "missing": missing, "decode": decodeErr} {
			flow := b(server.URL, server.Client()).(exchanger)
			_, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID(), SessionID: "s.v"}, "code")
			if err == nil {
				t.Fatalf("%s %s: want error", name, label)
			}
		}
	}
}

func TestKimiPollSuccessAndNetworkError(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a","refresh_token":"r","token_type":"bearer","expires_in":3600,"scope":"x"}`))
	}))
	defer ok.Close()
	flow := NewKimiFlow(KimiFlowConfig{ClientID: "c", TokenURL: ok.URL, HTTPClient: ok.Client()})
	res, err := flow.Poll(context.Background(), AuthSession{SessionID: "d"})
	if err != nil {
		t.Fatalf("kimi poll: %v", err)
	}
	if res.Status != PollStatusComplete || res.Token == nil || res.Token.AccessToken != "a" {
		t.Fatalf("res = %+v", res)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	flow2 := NewKimiFlow(KimiFlowConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := flow2.Poll(context.Background(), AuthSession{SessionID: "d"}); err == nil {
		t.Fatal("kimi poll network error: want error")
	}
}

func TestKimiPollMissingAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"bearer"}`))
	}))
	defer server.Close()
	flow := NewKimiFlow(KimiFlowConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	if _, err := flow.Poll(context.Background(), AuthSession{SessionID: "d"}); err == nil {
		t.Fatal("kimi poll missing token: want error")
	}
}

func TestDecodeCursorTokenDecodeError(t *testing.T) {
	if _, err := decodeCursorToken(badReader{}); err == nil {
		t.Fatal("decode error: want error")
	}
}

type badReader struct{}

func (badReader) Read(p []byte) (int, error) {
	return 0, http.ErrBodyNotAllowed
}
