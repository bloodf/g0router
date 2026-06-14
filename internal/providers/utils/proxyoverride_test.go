package utils

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestClientPoolProxyOverride(t *testing.T) {
	// Env proxy points at proxyEnv; the per-instance override points at proxyOverride.
	// With the override set, the request must be directed at the override proxy.
	logEnv := &proxyLog{}
	proxyEnv := proxyStub(logEnv)
	defer proxyEnv.Close()

	logOverride := &proxyLog{}
	proxyOverride := proxyStub(logOverride)
	defer proxyOverride.Close()

	t.Setenv("HTTP_PROXY", proxyEnv.URL)
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("NO_PROXY", "")

	pool := NewClientPool()
	if err := pool.SetProxyURL(proxyOverride.URL); err != nil {
		t.Fatalf("SetProxyURL: %v", err)
	}

	req := pool.AcquireRequest()
	defer pool.ReleaseRequest(req)
	resp := pool.AcquireResponse()
	defer pool.ReleaseResponse(resp)

	req.URI().SetScheme("http")
	req.URI().SetHost("example.com:80")
	req.URI().SetPath("/hello")
	req.Header.SetMethod(fasthttp.MethodGet)

	_ = pool.Do(req, resp)

	const targetHost = "example.com:80"
	if !logOverride.connectsTo(targetHost) {
		t.Fatalf("override proxy did not receive CONNECT for %q; connects=%v", targetHost, logOverride.connects)
	}
	if logEnv.connectsTo(targetHost) {
		t.Fatalf("env proxy received CONNECT for %q; the override must take precedence", targetHost)
	}
}

func TestClientPoolProxyOverrideClearedFallsBackToEnv(t *testing.T) {
	logEnv := &proxyLog{}
	proxyEnv := proxyStub(logEnv)
	defer proxyEnv.Close()

	t.Setenv("HTTP_PROXY", proxyEnv.URL)
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("NO_PROXY", "")

	pool := NewClientPool()
	// Empty override clears it; env path remains in effect.
	if err := pool.SetProxyURL(""); err != nil {
		t.Fatalf("SetProxyURL(\"\"): %v", err)
	}

	req := pool.AcquireRequest()
	defer pool.ReleaseRequest(req)
	resp := pool.AcquireResponse()
	defer pool.ReleaseResponse(resp)

	req.URI().SetScheme("http")
	req.URI().SetHost("example.com:80")
	req.URI().SetPath("/hello")
	req.Header.SetMethod(fasthttp.MethodGet)

	_ = pool.Do(req, resp)

	if !logEnv.connectsTo("example.com:80") {
		t.Fatalf("with no override, env proxy should receive the CONNECT; connects=%v", logEnv.connects)
	}
}
