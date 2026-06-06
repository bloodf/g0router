package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/proxy"
)

// HTTPClientForPool returns an *http.Client that routes through the given proxy pool.
// If pool is nil, returns a default client.
func HTTPClientForPool(pool *store.ProxyPool) *http.Client {
	if pool == nil {
		return &http.Client{}
	}
	transport := &http.Transport{}
	switch pool.Protocol {
	case "http", "https":
		proxyURL := buildProxyURL(pool)
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks5":
		dialer, err := socks5Dialer(pool)
		if err != nil {
			return &http.Client{}
		}
		transport.DialContext = dialer.DialContext
	default:
		return &http.Client{}
	}
	return &http.Client{Transport: transport}
}

// StreamHTTPClientForPool returns an *http.Client suitable for SSE streams with
// optional proxy routing. A zero or negative timeout falls back to
// DefaultStreamResponseHeaderTimeout.
func StreamHTTPClientForPool(responseHeaderTimeout time.Duration, pool *store.ProxyPool) *http.Client {
	if responseHeaderTimeout <= 0 {
		responseHeaderTimeout = DefaultStreamResponseHeaderTimeout
	}
	transport := &http.Transport{
		ResponseHeaderTimeout: responseHeaderTimeout,
		TLSHandshakeTimeout:   responseHeaderTimeout,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     true,
	}
	if pool != nil {
		switch pool.Protocol {
		case "http", "https":
			transport.Proxy = http.ProxyURL(buildProxyURL(pool))
		case "socks5":
			if dialer, err := socks5Dialer(pool); err == nil {
				transport.DialContext = dialer.DialContext
			}
		}
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Transport: transport}
}

// FasthttpClientForPool returns a fasthttp.Client configured for the given proxy pool.
func FasthttpClientForPool(pool *store.ProxyPool) *fasthttp.Client {
	if pool == nil {
		return &fasthttp.Client{}
	}
	client := &fasthttp.Client{}
	switch pool.Protocol {
	case "http", "https":
		proxyAddr := buildProxyAddr(pool)
		client.Dial = fasthttpproxy.FasthttpHTTPDialer(proxyAddr)
	case "socks5":
		proxyAddr := buildProxyAddr(pool)
		client.Dial = fasthttpproxy.FasthttpSocksDialer(proxyAddr)
	default:
		return &fasthttp.Client{}
	}
	return client
}

func buildProxyURL(pool *store.ProxyPool) *url.URL {
	u := &url.URL{
		Scheme: pool.Protocol,
		Host:   net.JoinHostPort(pool.Host, fmt.Sprintf("%d", pool.Port)),
	}
	if pool.Username != "" {
		if pool.Password != "" {
			u.User = url.UserPassword(pool.Username, pool.Password)
		} else {
			u.User = url.User(pool.Username)
		}
	}
	return u
}

func buildProxyAddr(pool *store.ProxyPool) string {
	if pool.Username != "" && pool.Password != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d", pool.Protocol, pool.Username, pool.Password, pool.Host, pool.Port)
	}
	if pool.Username != "" {
		return fmt.Sprintf("%s://%s@%s:%d", pool.Protocol, pool.Username, pool.Host, pool.Port)
	}
	return fmt.Sprintf("%s://%s:%d", pool.Protocol, pool.Host, pool.Port)
}

func socks5Dialer(pool *store.ProxyPool) (proxy.ContextDialer, error) {
	addr := net.JoinHostPort(pool.Host, fmt.Sprintf("%d", pool.Port))
	var auth *proxy.Auth
	if pool.Username != "" {
		auth = &proxy.Auth{
			User:     pool.Username,
			Password: pool.Password,
		}
	}
	d, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
	if err != nil {
		return nil, err
	}
	if cd, ok := d.(proxy.ContextDialer); ok {
		return cd, nil
	}
	// Wrap if the dialer does not implement ContextDialer.
	return &contextDialerWrapper{Dialer: d}, nil
}

type contextDialerWrapper struct {
	proxy.Dialer
}

func (w *contextDialerWrapper) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return w.Dialer.Dial(network, address)
}
