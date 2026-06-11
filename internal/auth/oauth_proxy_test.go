package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestOAuthDefaultClientHonorsEnvProxy(t *testing.T) {
	var mu sync.Mutex
	var proxySeen bool
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		proxySeen = true
		mu.Unlock()

		// Return a valid token response so Refresh succeeds through the proxy.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "proxy-token",
			"expires_in":   3600,
		})
	}))
	defer proxySrv.Close()

	t.Setenv("HTTP_PROXY", proxySrv.URL)
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("NO_PROXY", "")

	st := newTestStore(t)
	cfg := OAuthConfig{
		Provider:     "test",
		ClientID:     "client-id",
		AuthorizeURL: "https://example.com/authorize",
		// Use a non-loopback host so httpproxy does not implicitly bypass the proxy.
		TokenURL:    "http://example.com/token",
		RedirectURI: "http://localhost/cb",
	}
	flow := NewOAuthFlow(cfg, st, nil) // nil -> default client with ProxyFromEnvironment

	tok, err := flow.Refresh("rt-1")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if tok.AccessToken != "proxy-token" {
		t.Errorf("AccessToken = %q, want proxy-token", tok.AccessToken)
	}

	mu.Lock()
	seen := proxySeen
	mu.Unlock()
	if !seen {
		t.Fatal("default OAuth client did not route the token request through HTTP_PROXY")
	}
}
