package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchProtectedResourceMetadata(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"authorization_servers":["https://as.example"]}`))
	}))
	defer ok.Close()
	meta, err := fetchProtectedResourceMetadata(context.Background(), ok.Client(), ok.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(meta.AuthorizationServers) != 1 {
		t.Fatalf("servers = %v", meta.AuthorizationServers)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bad.Close()
	if _, err := fetchProtectedResourceMetadata(context.Background(), bad.Client(), bad.URL); err == nil {
		t.Fatal("bad status: want error")
	}

	decodeErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("notjson"))
	}))
	defer decodeErr.Close()
	if _, err := fetchProtectedResourceMetadata(context.Background(), decodeErr.Client(), decodeErr.URL); err == nil {
		t.Fatal("decode error: want error")
	}
}

func TestFetchAuthorizationServerMetadata(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token_endpoint":"https://as.example/token"}`))
	}))
	defer ok.Close()
	meta, err := fetchAuthorizationServerMetadata(context.Background(), ok.Client(), ok.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if meta.TokenEndpoint != "https://as.example/token" {
		t.Fatalf("token endpoint = %q", meta.TokenEndpoint)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	if _, err := fetchAuthorizationServerMetadata(context.Background(), bad.Client(), bad.URL); err == nil {
		t.Fatal("bad status: want error")
	}

	decodeErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("notjson"))
	}))
	defer decodeErr.Close()
	if _, err := fetchAuthorizationServerMetadata(context.Background(), decodeErr.Client(), decodeErr.URL); err == nil {
		t.Fatal("decode error: want error")
	}
}

func TestDiscoverTokenEndpoint(t *testing.T) {
	engineWith := func(client *http.Client) *OAuthEngine {
		return NewOAuthEngine(newFakeOAuthStore(), client)
	}

	// Non-http scheme rejected.
	if _, err := engineWith(http.DefaultClient).discoverTokenEndpoint(context.Background(), "ftp://x"); err == nil {
		t.Fatal("non-http scheme: want error")
	}
	// Unparseable URL.
	if _, err := engineWith(http.DefaultClient).discoverTokenEndpoint(context.Background(), "http://\x7f bad"); err == nil {
		t.Fatal("bad url: want error")
	}

	// Success.
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token_endpoint":"https://as.example/token"}`))
	}))
	defer ok.Close()
	got, err := engineWith(ok.Client()).discoverTokenEndpoint(context.Background(), ok.URL)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if got != "https://as.example/token" {
		t.Fatalf("token endpoint = %q", got)
	}

	// Missing token endpoint -> unavailable.
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer empty.Close()
	if _, err := engineWith(empty.Client()).discoverTokenEndpoint(context.Background(), empty.URL); err == nil {
		t.Fatal("missing token endpoint: want error")
	}

	// Bad status.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer bad.Close()
	if _, err := engineWith(bad.Client()).discoverTokenEndpoint(context.Background(), bad.URL); err == nil {
		t.Fatal("bad status: want error")
	}
}
