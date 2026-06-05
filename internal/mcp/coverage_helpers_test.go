package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountLabelFromToken(t *testing.T) {
	// Selected wins.
	if got := accountLabelFromToken(tokenResponse{AccountLabel: "x", Email: "e"}, "sel"); got != "sel" {
		t.Fatalf("selected = %q", got)
	}
	// AccountLabel.
	if got := accountLabelFromToken(tokenResponse{AccountLabel: "lbl", Email: "e"}, ""); got != "lbl" {
		t.Fatalf("label = %q", got)
	}
	// Email.
	if got := accountLabelFromToken(tokenResponse{Email: "user@x"}, ""); got != "user@x" {
		t.Fatalf("email = %q", got)
	}
	// Subject.
	if got := accountLabelFromToken(tokenResponse{Subject: "subj"}, ""); got != "subj" {
		t.Fatalf("subject = %q", got)
	}
	// Sub.
	if got := accountLabelFromToken(tokenResponse{Sub: "sub-id"}, ""); got != "sub-id" {
		t.Fatalf("sub = %q", got)
	}
	// Default.
	if got := accountLabelFromToken(tokenResponse{}, ""); got != "default" {
		t.Fatalf("default = %q", got)
	}
}

func TestSelectedAccountLabel(t *testing.T) {
	// Store without label support -> empty.
	nonLabel := struct{ OAuthStore }{}
	if got := selectedAccountLabel(nonLabel, "inst"); got != "" {
		t.Fatalf("non-label store = %q", got)
	}
	// Store with label support.
	store := newFakeOAuthStore()
	store.accountLabels["inst"] = "  team  "
	if got := selectedAccountLabel(store, "inst"); got != "team" {
		t.Fatalf("label = %q", got)
	}
}

func TestSplitScopesAndFirstNonEmpty(t *testing.T) {
	if splitScopes("") != nil {
		t.Fatal("empty scope -> nil")
	}
	if got := splitScopes("a b c"); len(got) != 3 {
		t.Fatalf("scopes = %v", got)
	}
	if firstNonEmpty("", "", "x", "y") != "x" {
		t.Fatal("firstNonEmpty mismatch")
	}
	if firstNonEmpty("", "") != "" {
		t.Fatal("firstNonEmpty all empty")
	}
}

func TestApplyHTTPHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
	applyHTTPHeaders(req, map[string]string{"X-A": "1", "X-B": "2"})
	if req.Header.Get("X-A") != "1" || req.Header.Get("X-B") != "2" {
		t.Fatalf("headers = %v", req.Header)
	}
	// Nil map is a no-op.
	applyHTTPHeaders(req, nil)
}

func TestLauncherHTTPClient(t *testing.T) {
	var nilLauncher *Launcher
	if nilLauncher.HTTPClient() != nil {
		t.Fatal("nil launcher should return nil client")
	}
	client := &http.Client{}
	l := NewLauncher(nil, client)
	if l.HTTPClient() != client {
		t.Fatal("HTTPClient should return configured doer")
	}
}

func TestWithAllowedTools(t *testing.T) {
	// Nil context defaults to Background and stores allowed set.
	ctx := WithAllowedTools(nil, "a", " b ", "")
	allowed, ok := allowedToolsFromContext(ctx)
	if !ok {
		t.Fatal("allowed tools not stored")
	}
	if _, has := allowed["a"]; !has {
		t.Fatal("missing tool a")
	}
	if _, has := allowed["b"]; !has {
		t.Fatal("trimmed tool b not stored")
	}
	if len(allowed) != 2 {
		t.Fatalf("allowed = %v", allowed)
	}
	// All-empty names returns ctx unchanged (no value).
	base := context.Background()
	if got := WithAllowedTools(base, "", "  "); got != base {
		t.Fatal("empty names should return original context")
	}
	// Nil context lookup.
	if _, ok := allowedToolsFromContext(nil); ok {
		t.Fatal("nil context should not have allowed tools")
	}
}

func TestProtectedResourceMetadataURLFallback(t *testing.T) {
	// Server without WWW-Authenticate header -> fallback well-known path.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	got, err := protectedResourceMetadataURL(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("protectedResourceMetadataURL: %v", err)
	}
	if got != server.URL+"/.well-known/oauth-protected-resource" {
		t.Fatalf("fallback url = %q", got)
	}
}

func TestProtectedResourceMetadataURLFromHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="https://meta.example/.well-known/x"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	got, err := protectedResourceMetadataURL(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("protectedResourceMetadataURL: %v", err)
	}
	if got != "https://meta.example/.well-known/x" {
		t.Fatalf("header url = %q", got)
	}
}

func TestProtectedResourceMetadataURLNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	if _, err := protectedResourceMetadataURL(context.Background(), client, url); err == nil {
		t.Fatal("network error: want error")
	}
}
