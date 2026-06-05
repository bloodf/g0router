package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestStreamHTTPClientReturnsWithinResponseHeaderTimeout(t *testing.T) {
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never send headers until the test releases us: simulates a stalled
		// upstream that accepts the connection but never responds.
		<-release
	}))
	// Cleanups run LIFO: unblock the handler before srv.Close so the server's
	// graceful shutdown does not wait forever on the stalled request.
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	client := StreamHTTPClient(150 * time.Millisecond)

	start := time.Now()
	resp, err := client.Get(srv.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatalf("expected timeout error, got response")
	}
	elapsed := time.Since(start)

	var urlErr *url.Error
	if !errors.As(err, &urlErr) || !urlErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("reader did not return promptly: %v", elapsed)
	}
}

func TestStreamHTTPClientDefaultsAndHasNoTotalTimeout(t *testing.T) {
	client := StreamHTTPClient(0)
	if client.Timeout != 0 {
		t.Fatalf("client.Timeout = %v, want 0 (no total timeout to avoid killing long streams)", client.Timeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.ResponseHeaderTimeout != DefaultStreamResponseHeaderTimeout {
		t.Fatalf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, DefaultStreamResponseHeaderTimeout)
	}
}
