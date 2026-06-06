package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"golang.org/x/net/proxy"
)

type fakeDialer struct {
	dialed bool
}

func (f *fakeDialer) Dial(network, addr string) (net.Conn, error) {
	f.dialed = true
	return nil, errors.New("fake dial error")
}

func TestHTTPClientForPoolInvalidProtocol(t *testing.T) {
	pool := &store.ProxyPool{Protocol: "ftp", Host: "host", Port: 8080}
	client := HTTPClientForPool(pool)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Transport != nil {
		t.Fatal("expected nil Transport for invalid protocol")
	}
}

func TestHTTPClientForPoolSocks5DialerFailure(t *testing.T) {
	orig := socks5Constructor
	socks5Constructor = func(network, address string, auth *proxy.Auth, forward proxy.Dialer) (proxy.Dialer, error) {
		return nil, errors.New("socks5 init failed")
	}
	defer func() { socks5Constructor = orig }()

	pool := &store.ProxyPool{Protocol: "socks5", Host: "host", Port: 1080}
	client := HTTPClientForPool(pool)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Transport != nil {
		t.Fatal("expected nil Transport for SOCKS5 dialer failure")
	}
}

func TestStreamHTTPClientForPoolSocks5DialerError(t *testing.T) {
	orig := socks5Constructor
	socks5Constructor = func(network, address string, auth *proxy.Auth, forward proxy.Dialer) (proxy.Dialer, error) {
		return nil, errors.New("socks5 init failed")
	}
	defer func() { socks5Constructor = orig }()

	pool := &store.ProxyPool{Protocol: "socks5", Host: "host", Port: 1080}
	client := StreamHTTPClientForPool(5*time.Second, pool)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if transport.DialContext != nil {
		t.Fatal("expected nil DialContext when SOCKS5 dialer fails")
	}
}

func TestFasthttpClientForPoolInvalidProtocol(t *testing.T) {
	pool := &store.ProxyPool{Protocol: "ftp", Host: "host", Port: 8080}
	client := FasthttpClientForPool(pool)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Dial != nil {
		t.Fatal("expected nil Dial for invalid protocol")
	}
}

func TestBuildProxyURLUsernameOnly(t *testing.T) {
	pool := &store.ProxyPool{Protocol: "http", Host: "host", Port: 8080, Username: "user"}
	u := buildProxyURL(pool)
	if u.User == nil {
		t.Fatal("expected user")
	}
	if u.User.Username() != "user" {
		t.Fatalf("username = %q, want user", u.User.Username())
	}
	if _, hasPassword := u.User.Password(); hasPassword {
		t.Fatal("expected no password")
	}
}

func TestBuildProxyAddrUsernameOnly(t *testing.T) {
	pool := &store.ProxyPool{Protocol: "http", Host: "host", Port: 8080, Username: "user"}
	addr := buildProxyAddr(pool)
	expected := "http://user@host:8080"
	if addr != expected {
		t.Fatalf("addr = %q, want %q", addr, expected)
	}
}

func TestSocks5DialerError(t *testing.T) {
	orig := socks5Constructor
	socks5Constructor = func(network, address string, auth *proxy.Auth, forward proxy.Dialer) (proxy.Dialer, error) {
		return nil, errors.New("socks5 init failed")
	}
	defer func() { socks5Constructor = orig }()

	pool := &store.ProxyPool{Protocol: "socks5", Host: "host", Port: 1080}
	_, err := socks5Dialer(pool)
	if err == nil {
		t.Fatal("expected error for SOCKS5 failure")
	}
}

func TestSocks5DialerWrapper(t *testing.T) {
	orig := socks5Constructor
	socks5Constructor = func(network, address string, auth *proxy.Auth, forward proxy.Dialer) (proxy.Dialer, error) {
		return &fakeDialer{}, nil
	}
	defer func() { socks5Constructor = orig }()

	pool := &store.ProxyPool{Protocol: "socks5", Host: "host", Port: 1080}
	dialer, err := socks5Dialer(pool)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dialer == nil {
		t.Fatal("expected non-nil dialer")
	}

	_, err = dialer.DialContext(context.Background(), "tcp", "127.0.0.1:1")
	if err == nil || err.Error() != "fake dial error" {
		t.Fatalf("expected fake dial error, got %v", err)
	}
}

func TestContextDialerWrapper(t *testing.T) {
	d := &fakeDialer{}
	wrapper := &contextDialerWrapper{Dialer: d}
	_, err := wrapper.DialContext(context.Background(), "tcp", "127.0.0.1:1")
	if err == nil || err.Error() != "fake dial error" {
		t.Fatalf("expected fake dial error, got %v", err)
	}
	if !d.dialed {
		t.Fatal("expected Dial to be called")
	}
}

func TestStreamHTTPClientForPoolNilPool(t *testing.T) {
	client := StreamHTTPClientForPool(5*time.Second, nil)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if transport.Proxy == nil {
		t.Fatal("expected non-nil Proxy for nil pool (ProxyFromEnvironment)")
	}
}

func TestStreamHTTPClientForPoolDefaultTimeout(t *testing.T) {
	client := StreamHTTPClientForPool(0, nil)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if transport.ResponseHeaderTimeout != DefaultStreamResponseHeaderTimeout {
		t.Fatalf("timeout = %v, want %v", transport.ResponseHeaderTimeout, DefaultStreamResponseHeaderTimeout)
	}
}

func TestFasthttpClientForPoolSocks5(t *testing.T) {
	pool := &store.ProxyPool{Protocol: "socks5", Host: "host", Port: 1080}
	client := FasthttpClientForPool(pool)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Dial == nil {
		t.Fatal("expected non-nil Dial for socks5")
	}
}
