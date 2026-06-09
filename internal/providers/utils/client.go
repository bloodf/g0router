package utils

import (
	"github.com/valyala/fasthttp"
)

// ClientPool wraps a fasthttp.Client with common configuration.
type ClientPool struct {
	client *fasthttp.Client
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
	}
}

// Do executes an HTTP request and populates the response.
func (p *ClientPool) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	return p.client.Do(req, resp)
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
