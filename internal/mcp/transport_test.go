package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// fakeResp is one canned HTTP response served by fakeTransport.
type fakeResp struct {
	status  int               // HTTP status code
	body    string            // response body
	headers map[string]string // response headers (e.g. mcp-session-id, Content-Type)
	err     error             // when non-nil, RoundTrip returns this error
}

// fakeTransport is a test-only http.RoundTripper that returns canned responses in
// order (or, when block is set, blocks until the request context is canceled — the
// probe-timeout test). It captures every request it receives so tests can assert
// headers, method, URL, and body. NO real network is ever opened. Mirrors the
// injectable *http.Client philosophy of internal/auth/oauth.go:128.
type fakeTransport struct {
	mu        sync.Mutex
	responses []fakeResp      // consumed in order
	idx       int             // next response index
	captured  []*http.Request // every request seen
	bodies    []string        // captured request bodies (parallel to captured)
	block     bool            // when true, block until ctx is canceled (timeout test)
}

func (f *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Capture the request (and a copy of its body) before doing anything else.
	var bodyStr string
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		bodyStr = string(b)
		r.Body = io.NopCloser(bytes.NewReader(b))
	}
	f.mu.Lock()
	f.captured = append(f.captured, r)
	f.bodies = append(f.bodies, bodyStr)
	f.mu.Unlock()

	if f.block {
		<-r.Context().Done()
		return nil, r.Context().Err()
	}

	f.mu.Lock()
	if f.idx >= len(f.responses) {
		f.mu.Unlock()
		return nil, io.EOF
	}
	resp := f.responses[f.idx]
	f.idx++
	f.mu.Unlock()

	if resp.err != nil {
		return nil, resp.err
	}
	status := resp.status
	if status == 0 {
		status = http.StatusOK
	}
	header := make(http.Header)
	for k, v := range resp.headers {
		header.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Request:    r,
	}, nil
}

// fakeClient wraps a fakeTransport in an *http.Client for injection.
func fakeClient(t *fakeTransport) *http.Client {
	return &http.Client{Transport: t}
}

// lastBody returns the body of the request at index i (test helper).
func (f *fakeTransport) bodyAt(i int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i < 0 || i >= len(f.bodies) {
		return ""
	}
	return f.bodies[i]
}

// fakeClock is an injectable time source for the registry cache TTL + the OAuth /
// health expiry derivations — advanced explicitly by tests, NEVER a real sleep.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// shortCtx returns a context that is already canceled / about to expire — used to
// drive the probe-timeout path without a real wall-clock wait.
func shortCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Millisecond)
}
