package utils

import (
	"net/http"
	"time"
)

// DefaultStreamResponseHeaderTimeout bounds how long a streaming client waits
// for the upstream to send response headers. It must NOT be applied as a total
// client Timeout: doing so would abort long-lived but legitimate SSE streams.
const DefaultStreamResponseHeaderTimeout = 60 * time.Second

// StreamHTTPClient returns an *http.Client suitable for reading server-sent
// event streams. It bounds the wait for response headers (and the TLS
// handshake) so a stalled upstream cannot hang the SSE reader forever, while
// leaving the body read unbounded so long legitimate streams are not killed.
//
// A zero or negative timeout falls back to DefaultStreamResponseHeaderTimeout.
func StreamHTTPClient(responseHeaderTimeout time.Duration) *http.Client {
	if responseHeaderTimeout <= 0 {
		responseHeaderTimeout = DefaultStreamResponseHeaderTimeout
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: responseHeaderTimeout,
		TLSHandshakeTimeout:   responseHeaderTimeout,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{Transport: transport}
}
