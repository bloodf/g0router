package utils

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/http/httpproxy"
)

// ClientPool wraps a fasthttp.Client with common configuration.
type ClientPool struct {
	client        *fasthttp.Client
	proxyFunc     func(*url.URL) (*url.URL, error)
	mu            sync.Mutex
	proxies       map[string]*fasthttp.Client
	proxyOverride *url.URL // per-instance proxy override (PAR-PLAT-009); nil = use env proxyFunc
}

// SetProxyURL sets a per-instance proxy override that takes precedence over the
// environment proxyFunc for all subsequent Do calls. An empty string clears the
// override (restoring the env-proxy behavior). Additive and backward-compatible:
// when no override is set, Do is unchanged.
func (p *ClientPool) SetProxyURL(proxyURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if proxyURL == "" {
		p.proxyOverride = nil
		return nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("parse proxy url %q: %w", proxyURL, err)
	}
	p.proxyOverride = u
	return nil
}

// NewClientPool creates a shared fasthttp client with sensible defaults.
func NewClientPool() *ClientPool {
	return &ClientPool{
		client: &fasthttp.Client{
			MaxConnsPerHost:               100,
			ReadTimeout:                   120000000000, // 120s
			WriteTimeout:                  30000000000,  // 30s
			MaxIdleConnDuration:           60000000000,  // 60s
			DisablePathNormalizing:        true,
			DisableHeaderNamesNormalizing: true,
		},
		proxyFunc: httpproxy.FromEnvironment().ProxyFunc(),
		proxies:   make(map[string]*fasthttp.Client),
	}
}

// Do executes an HTTP request and populates the response.
func (p *ClientPool) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	uri := req.URI()
	scheme := string(uri.Scheme())
	if scheme == "" {
		scheme = "http"
	}
	target := &url.URL{Scheme: scheme, Host: string(uri.Host())}

	// Per-instance proxy override (PAR-PLAT-009) takes precedence over env proxy.
	p.mu.Lock()
	override := p.proxyOverride
	p.mu.Unlock()
	if override != nil {
		client := p.clientForProxy(override)
		if err := client.Do(req, resp); err != nil {
			return fmt.Errorf("do via proxy %s: %w", override, err)
		}
		return nil
	}

	proxyURL, err := p.proxyFunc(target)
	if err != nil {
		return fmt.Errorf("resolve proxy for %s: %w", target, err)
	}
	if proxyURL == nil {
		return p.client.Do(req, resp)
	}

	client := p.clientForProxy(proxyURL)
	if err := client.Do(req, resp); err != nil {
		return fmt.Errorf("do via proxy %s: %w", proxyURL, err)
	}
	return nil
}

func (p *ClientPool) clientForProxy(proxyURL *url.URL) *fasthttp.Client {
	key := proxyURL.String()
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.proxies[key]; ok {
		return c
	}

	c := &fasthttp.Client{
		MaxConnsPerHost:               100,
		ReadTimeout:                   120000000000, // 120s
		WriteTimeout:                  30000000000,  // 30s
		MaxIdleConnDuration:           60000000000,  // 60s
		DisablePathNormalizing:        true,
		DisableHeaderNamesNormalizing: true,
		Dial:                          fasthttpproxy.FasthttpHTTPDialer(proxyURL.Host),
	}
	p.proxies[key] = c
	return c
}

// AcquireRequest returns a pooled fasthttp.Request.
func (p *ClientPool) AcquireRequest() *fasthttp.Request {
	return fasthttp.AcquireRequest()
}

// ReleaseRequest returns a fasthttp.Request to the pool.
func (p *ClientPool) ReleaseRequest(req *fasthttp.Request) {
	fasthttp.ReleaseRequest(req)
}

// AcquireResponse returns a pooled fasthttp.Response.
func (p *ClientPool) AcquireResponse() *fasthttp.Response {
	return fasthttp.AcquireResponse()
}

// ReleaseResponse returns a fasthttp.Response to the pool.
func (p *ClientPool) ReleaseResponse(resp *fasthttp.Response) {
	fasthttp.ReleaseResponse(resp)
}
