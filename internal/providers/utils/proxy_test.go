package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/valyala/fasthttp"
)

// proxyLog records inbound proxy requests safely for -race.
type proxyLog struct {
	mu       sync.Mutex
	connects []string
}

func (l *proxyLog) recordConnect(host string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.connects = append(l.connects, host)
}

func (l *proxyLog) connectsTo(host string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, h := range l.connects {
		if h == host {
			return true
		}
	}
	return false
}

func proxyStub(log *proxyLog) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			log.recordConnect(r.Host)
			// Respond 200 and close so the client sees EOF quickly.
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, err := hj.Hijack()
				if err == nil {
					_, _ = conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
					_ = conn.Close()
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func targetStub(t *testing.T) (*httptest.Server, *bool) {
	var seen bool
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = true
		w.WriteHeader(http.StatusOK)
	})), &seen
}

func TestClientPoolUsesEnvProxy(t *testing.T) {
	log := &proxyLog{}
	proxySrv := proxyStub(log)
	defer proxySrv.Close()

	t.Setenv("HTTP_PROXY", proxySrv.URL)
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("NO_PROXY", "")

	pool := NewClientPool()
	req := pool.AcquireRequest()
	defer pool.ReleaseRequest(req)
	resp := pool.AcquireResponse()
	defer pool.ReleaseResponse(resp)

	// Use a non-loopback host so httpproxy does not implicitly bypass the proxy.
	// The proxy stub is not a real tunnel, so the call may error; we only care
	// that the outbound connection was directed at the proxy.
	req.URI().SetScheme("http")
	req.URI().SetHost("example.com:80")
	req.URI().SetPath("/hello")
	req.Header.SetMethod(fasthttp.MethodGet)

	_ = pool.Do(req, resp)

	const targetHost = "example.com:80"
	if !log.connectsTo(targetHost) {
		t.Fatalf("proxy did not receive CONNECT for target %q; connects=%v", targetHost, log.connects)
	}
}

func TestClientPoolNoProxyDirect(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		targetSrv, targetSeen := targetStub(t)
		defer targetSrv.Close()

		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "")
		t.Setenv("NO_PROXY", "")

		pool := NewClientPool()
		req := pool.AcquireRequest()
		defer pool.ReleaseRequest(req)
		resp := pool.AcquireResponse()
		defer pool.ReleaseResponse(resp)

		req.SetRequestURI(targetSrv.URL + "/hello")
		req.Header.SetMethod(fasthttp.MethodGet)

		if err := pool.Do(req, resp); err != nil {
			t.Fatalf("Do: %v", err)
		}
		if !*targetSeen {
			t.Fatal("request did not reach target directly")
		}
	})

	t.Run("no_proxy", func(t *testing.T) {
		log := &proxyLog{}
		proxySrv := proxyStub(log)
		defer proxySrv.Close()

		targetSrv, targetSeen := targetStub(t)
		defer targetSrv.Close()

		targetHost := targetSrv.Listener.Addr().String()
		targetIP := strings.Split(targetHost, ":")[0]

		t.Setenv("HTTP_PROXY", proxySrv.URL)
		t.Setenv("HTTPS_PROXY", "")
		t.Setenv("NO_PROXY", targetIP)

		pool := NewClientPool()
		req := pool.AcquireRequest()
		defer pool.ReleaseRequest(req)
		resp := pool.AcquireResponse()
		defer pool.ReleaseResponse(resp)

		req.SetRequestURI(targetSrv.URL + "/hello")
		req.Header.SetMethod(fasthttp.MethodGet)

		if err := pool.Do(req, resp); err != nil {
			t.Fatalf("Do: %v", err)
		}
		if !*targetSeen {
			t.Fatal("request did not reach target despite NO_PROXY")
		}
		if len(log.connects) != 0 {
			t.Errorf("proxy received %d CONNECT(s), expected 0 because of NO_PROXY", len(log.connects))
		}
	})
}

func TestClientPoolHTTPSProxyPrecedence(t *testing.T) {
	logA := &proxyLog{}
	proxyA := proxyStub(logA)
	defer proxyA.Close()

	logB := &proxyLog{}
	proxyB := proxyStub(logB)
	defer proxyB.Close()

	t.Setenv("HTTP_PROXY", proxyA.URL)
	t.Setenv("HTTPS_PROXY", proxyB.URL)
	t.Setenv("NO_PROXY", "")

	pool := NewClientPool()
	req := pool.AcquireRequest()
	defer pool.ReleaseRequest(req)
	resp := pool.AcquireResponse()
	defer pool.ReleaseResponse(resp)

	// Force an https:// target so the resolver must select HTTPS_PROXY over HTTP_PROXY.
	// Use a non-loopback host so httpproxy does not implicitly bypass the proxy.
	req.URI().SetScheme("https")
	req.URI().SetHost("example.com:443")
	req.URI().SetPath("/hello")
	req.Header.SetMethod(fasthttp.MethodGet)

	_ = pool.Do(req, resp)

	const targetHost = "example.com:443"
	if !logB.connectsTo(targetHost) {
		t.Fatalf("HTTPS_PROXY did not receive CONNECT for target %q; connects=%v", targetHost, logB.connects)
	}
	if logA.connectsTo(targetHost) {
		t.Fatalf("HTTP_PROXY received CONNECT for target %q; expected only HTTPS_PROXY", targetHost)
	}
}
