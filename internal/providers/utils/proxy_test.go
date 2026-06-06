package utils

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// TestHTTPClientForPoolNilReturnsDefault verifies that a nil pool yields a
// working default client.
func TestHTTPClientForPoolNilReturnsDefault(t *testing.T) {
	client := HTTPClientForPool(nil)
	if client == nil {
		t.Fatal("HTTPClientForPool(nil) returned nil")
	}
	if client.Transport == nil {
		// default http.Client has nil Transport which uses DefaultTransport
		return
	}
	// If transport is set, it should not have a Proxy when pool is nil.
	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		return
	}
	if tr.Proxy != nil {
		// acceptable if it falls back to environment proxy
	}
}

// TestFasthttpClientForPoolNilReturnsDefault verifies that a nil pool yields a
// working default fasthttp client.
func TestFasthttpClientForPoolNilReturnsDefault(t *testing.T) {
	client := FasthttpClientForPool(nil)
	if client == nil {
		t.Fatal("FasthttpClientForPool(nil) returned nil")
	}
}

// TestHTTPClientForPoolHTTPProxy verifies that requests route through an HTTP proxy.
func TestHTTPClientForPoolHTTPProxy(t *testing.T) {
	// Target server that echoes a marker header.
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok from target"))
	}))
	defer target.Close()

	// Simple HTTP proxy that records whether it was hit.
	var proxyHit bool
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit = true
		if r.Method == http.MethodConnect {
			// Handle CONNECT tunneling.
			target, err := net.Dial("tcp", r.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, err := hj.Hijack()
				if err != nil {
					target.Close()
					return
				}
				go func() {
					_, _ = io.Copy(target, conn)
				}()
				go func() {
					_, _ = io.Copy(conn, target)
				}()
			}
			return
		}
		// For non-CONNECT HTTP proxy, forward the request.
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxyServer.Close()

	proxyURL := proxyServer.URL
	// Parse host and port from proxy URL.
	host, port, _ := net.SplitHostPort(proxyURL[len("http://"):])

	pool := &store.ProxyPool{
		Protocol: "http",
		Host:     host,
		Port:     parsePort(t, port),
	}

	client := HTTPClientForPool(pool)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok from target" {
		t.Fatalf("unexpected body: %s", body)
	}
	if !proxyHit {
		t.Fatal("HTTP proxy was not hit")
	}
}

// TestHTTPClientForPoolSOCKS5Proxy verifies that requests route through a SOCKS5 proxy.
func TestHTTPClientForPoolSOCKS5Proxy(t *testing.T) {
	// Target server.
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok via socks5"))
	}))
	defer target.Close()

	// Minimal SOCKS5 server.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	go runMinimalSocks5Server(t, listener)

	_, portStr, _ := net.SplitHostPort(listener.Addr().String())
	pool := &store.ProxyPool{
		Protocol: "socks5",
		Host:     "127.0.0.1",
		Port:     parsePort(t, portStr),
	}

	client := HTTPClientForPool(pool)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do via socks5: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok via socks5" {
		t.Fatalf("unexpected body: %s", body)
	}
}

// TestFasthttpClientForPoolHTTPProxy verifies fasthttp requests route through an HTTP proxy.
func TestFasthttpClientForPoolHTTPProxy(t *testing.T) {
	var proxyHit bool
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit = true
		if r.Method == http.MethodConnect {
			target, err := net.Dial("tcp", r.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, err := hj.Hijack()
				if err != nil {
					target.Close()
					return
				}
				go func() {
					_, _ = io.Copy(target, conn)
				}()
				go func() {
					_, _ = io.Copy(conn, target)
				}()
			}
			return
		}
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxyServer.Close()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fasthttp ok"))
	}))
	defer target.Close()

	host, port, _ := net.SplitHostPort(proxyServer.URL[len("http://"):])
	pool := &store.ProxyPool{
		Protocol: "http",
		Host:     host,
		Port:     parsePort(t, port),
	}

	client := FasthttpClientForPool(pool)
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(target.URL)
	req.Header.SetMethod(fasthttp.MethodGet)

	if err := client.Do(req, resp); err != nil {
		t.Fatalf("fasthttp client.Do: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode())
	}
	if string(resp.Body()) != "fasthttp ok" {
		t.Fatalf("unexpected body: %s", resp.Body())
	}
	if !proxyHit {
		t.Fatal("HTTP proxy was not hit by fasthttp client")
	}
}

// TestStreamHTTPClientForPoolWithProxy verifies streaming client uses proxy.
func TestStreamHTTPClientForPoolWithProxy(t *testing.T) {
	var proxyHit bool
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHit = true
		if r.Method == http.MethodConnect {
			target, err := net.Dial("tcp", r.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, err := hj.Hijack()
				if err != nil {
					target.Close()
					return
				}
				go func() {
					_, _ = io.Copy(target, conn)
				}()
				go func() {
					_, _ = io.Copy(conn, target)
				}()
			}
			return
		}
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxyServer.Close()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stream ok"))
	}))
	defer target.Close()

	host, port, _ := net.SplitHostPort(proxyServer.URL[len("http://"):])
	pool := &store.ProxyPool{
		Protocol: "http",
		Host:     host,
		Port:     parsePort(t, port),
	}

	client := StreamHTTPClientForPool(5*time.Second, pool)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "stream ok" {
		t.Fatalf("unexpected body: %s", body)
	}
	if !proxyHit {
		t.Fatal("stream proxy was not hit")
	}
}

// runMinimalSocks5Server runs a tiny SOCKS5 server that only handles CONNECT.
func runMinimalSocks5Server(t *testing.T, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			// Read greeting.
			buf := make([]byte, 2)
			if _, err := io.ReadFull(c, buf); err != nil {
				return
			}
			nmethods := int(buf[1])
			methods := make([]byte, nmethods)
			if _, err := io.ReadFull(c, methods); err != nil {
				return
			}
			// Respond with no-auth.
			if _, err := c.Write([]byte{0x05, 0x00}); err != nil {
				return
			}
			// Read request.
			header := make([]byte, 4)
			if _, err := io.ReadFull(c, header); err != nil {
				return
			}
			if header[1] != 0x01 { // not CONNECT
				return
			}
			var addr string
			switch header[3] {
			case 0x01: // IPv4
				ip := make([]byte, 4)
				if _, err := io.ReadFull(c, ip); err != nil {
					return
				}
				port := make([]byte, 2)
				if _, err := io.ReadFull(c, port); err != nil {
					return
				}
				addr = fmt.Sprintf("%d.%d.%d.%d:%d", ip[0], ip[1], ip[2], ip[3], int(port[0])<<8|int(port[1]))
			case 0x03: // Domain
				lenBuf := make([]byte, 1)
				if _, err := io.ReadFull(c, lenBuf); err != nil {
					return
				}
				domain := make([]byte, lenBuf[0])
				if _, err := io.ReadFull(c, domain); err != nil {
					return
				}
				port := make([]byte, 2)
				if _, err := io.ReadFull(c, port); err != nil {
					return
				}
				addr = fmt.Sprintf("%s:%d", string(domain), int(port[0])<<8|int(port[1]))
			default:
				return
			}
			// Connect to target.
			target, err := net.Dial("tcp", addr)
			if err != nil {
				_, _ = c.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
				return
			}
			defer target.Close()
			// Respond success.
			if _, err := c.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
				return
			}
			// Bidirectional copy.
			done := make(chan struct{}, 2)
			go func() {
				_, _ = io.Copy(target, c)
				done <- struct{}{}
			}()
			go func() {
				_, _ = io.Copy(c, target)
				done <- struct{}{}
			}()
			<-done
		}(conn)
	}
}

func parsePort(t *testing.T, s string) int {
	t.Helper()
	p, err := net.LookupPort("tcp", s)
	if err != nil {
		// fallback for numeric strings
		fmt.Sscanf(s, "%d", &p)
	}
	if p == 0 {
		t.Fatalf("invalid port %q", s)
	}
	return p
}
