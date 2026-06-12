package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// UsageEntry is the api-side data passed to the wired UsageRecorder when a
// request completes. It mirrors the request_log schema the persistence layer
// stores but lives in the transport package so the api layer can stay free
// of repository imports (AGENTS.md layered DDD).
type UsageEntry struct {
	Provider         string
	Model            string
	ConnectionID     string
	APIKey           string
	Endpoint         string
	PromptTokens     int64
	CompletionTokens int64
	Cost             float64
	Status           string
	Tokens           map[string]int64
}

// RequestDetailCapture is the api-side payload for the wired DetailCapture
// when a request completes (success or error). It mirrors the request_details
// schema the persistence layer stores.
type RequestDetailCapture struct {
	Provider         string
	Model            string
	ConnectionID     string
	Status           string
	Latency          any
	Tokens           any
	Request          any
	ProviderRequest  any
	ProviderResponse any
	Response         any
}

// UsageRecorder is the consumer interface satisfied by the w5-b Recorder. The
// api layer must not import internal/store; the server-side adapter (in
// internal/server) wraps the recorder to translate UsageEntry → store.RequestLogEntry.
type UsageRecorder interface {
	Record(entry *UsageEntry) error
}

// PendingTracker is the consumer interface satisfied by the w5-b Tracker.
type PendingTracker interface {
	Start(model, provider, connectionID string)
	End(model, provider, connectionID string, isError bool)
}

// DetailCapture is the consumer interface satisfied by the w5-c DetailWriter.
// Save is invoked on both success and error paths. Close flushes any
// buffered items (PAR-USAGE-026; called from server shutdown).
type DetailCapture interface {
	Save(capture RequestDetailCapture) error
	Close() error
}

var sensitiveHeaderKeys = []string{"authorization", "x-api-key", "cookie", "token", "api-key"}

// sanitizeHeaders returns a copy of headers with sensitive keys removed.
// Matching is case-insensitive and by substring, matching the reference
// (internal/usage.SanitizeHeaders). This lives in the api package so the
// transport layer can redact captured request details before passing them
// across the DetailCapture boundary.
func sanitizeHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string)
	if headers == nil {
		return out
	}
	for k, v := range headers {
		lower := strings.ToLower(k)
		sensitive := false
		for _, s := range sensitiveHeaderKeys {
			if strings.Contains(lower, s) {
				sensitive = true
				break
			}
		}
		if !sensitive {
			out[k] = v
		}
	}
	return out
}

// requestHeadersFromCtx copies the incoming request headers into a plain
// map[string]string for detail capture.
func requestHeadersFromCtx(ctx *fasthttp.RequestCtx) map[string]string {
	headers := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	return headers
}

// captureRequest builds a request-detail map from the raw body and incoming
// headers, sanitizing sensitive headers before persistence.
func captureRequest(body []byte, headers map[string]string) any {
	var reqMap map[string]any
	if err := json.Unmarshal(body, &reqMap); err != nil {
		reqMap = map[string]any{"raw_body": string(body)}
	}
	if reqMap == nil {
		reqMap = map[string]any{}
	}
	reqMap["headers"] = sanitizeHeaders(headers)
	return reqMap
}

// recordGlue holds the usage-recording dependencies shared by all handlers.
// It exists so the duplicated recordError/recordNonStream/recordStream blocks
// can be expressed once and parameterized by endpoint.
type recordGlue struct {
	recorder UsageRecorder
	tracker  PendingTracker
	detail   DetailCapture
	apiKey   string // populated by handlers after the x-g0-vk gate admits the request
}

// recordError terminates pending tracking and persists a usage entry + detail
// capture for a provider error on the given endpoint.
func (g *recordGlue) recordError(endpoint, model, provider, connID string, body []byte, headers map[string]string, perr *schemas.ProviderError) {
	if g.tracker != nil {
		g.tracker.End(model, provider, connID, true)
	}
	statusCode := perr.StatusCode
	if statusCode == 0 {
		statusCode = 502
	}
	statusLabel := fmt.Sprintf("%d", statusCode)
	if g.recorder != nil {
		if err := g.recorder.Record(&UsageEntry{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			APIKey:       g.apiKey,
			Endpoint:     endpoint,
			Status:       "error",
			Tokens:       map[string]int64{},
		}); err != nil {
			log.Printf("usage record error on %s: %v", endpoint, err)
		}
	}
	if g.detail != nil {
		if err := g.detail.Save(RequestDetailCapture{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			Status:       "error",
			Request:      captureRequest(body, headers),
			Response:     map[string]any{"error": map[string]any{"message": perr.Message, "status": statusLabel}},
		}); err != nil {
			log.Printf("detail save error on %s: %v", endpoint, err)
		}
	}
}

// recordNonStream terminates pending tracking and persists a usage entry +
// detail capture for a successful non-streaming request.
func (g *recordGlue) recordNonStream(endpoint, model, provider, connID string, body []byte, headers map[string]string, promptTokens, completionTokens int64, response any) {
	if g.tracker != nil {
		g.tracker.End(model, provider, connID, false)
	}
	entry := &UsageEntry{
		Provider:     provider,
		Model:        model,
		ConnectionID: connID,
		APIKey:       g.apiKey,
		Endpoint:     endpoint,
		Status:       "ok",
	}
	if promptTokens > 0 || completionTokens > 0 {
		entry.PromptTokens = promptTokens
		entry.CompletionTokens = completionTokens
		entry.Tokens = map[string]int64{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
		}
	}
	if g.recorder != nil {
		if err := g.recorder.Record(entry); err != nil {
			log.Printf("usage record error on %s: %v", endpoint, err)
		}
	}
	if g.detail != nil {
		if err := g.detail.Save(RequestDetailCapture{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			Status:       "success",
			Request:      captureRequest(body, headers),
			Tokens:       entry.Tokens,
			Response:     response,
		}); err != nil {
			log.Printf("detail save error on %s: %v", endpoint, err)
		}
	}
}

// extractInt / toFloat are small helpers for pulling numeric fields out of an
// untyped usage map. They live next to recordStream, their only caller.
func extractInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		return int(toFloat(v))
	}
	return 0
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

func firstNonZero(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

// recordStream terminates pending tracking and persists a usage entry + detail
// capture for a streaming request. The summary carries accumulated/estimated
// usage from the stream processor.
func (g *recordGlue) recordStream(endpoint, model, provider, connID string, body []byte, headers map[string]string, summary translation.StreamSummary, sErr error) {
	isError := sErr != nil
	if g.tracker != nil {
		g.tracker.End(model, provider, connID, isError)
	}
	status := "ok"
	if isError {
		status = "error"
	}
	entry := &UsageEntry{
		Provider:     provider,
		Model:        model,
		ConnectionID: connID,
		APIKey:       g.apiKey,
		Endpoint:     endpoint,
		Status:       status,
	}
	if summary.Usage != nil {
		entry.PromptTokens = int64(firstNonZero(extractInt(summary.Usage, "prompt_tokens"), extractInt(summary.Usage, "input_tokens")))
		entry.CompletionTokens = int64(firstNonZero(extractInt(summary.Usage, "completion_tokens"), extractInt(summary.Usage, "output_tokens")))
		entry.Tokens = map[string]int64{}
		for k, v := range summary.Usage {
			entry.Tokens[k] = int64(toFloat(v))
		}
	}
	if g.recorder != nil {
		if err := g.recorder.Record(entry); err != nil {
			log.Printf("usage record error on %s: %v", endpoint, err)
		}
	}
	if g.detail != nil {
		capture := RequestDetailCapture{
			Provider:     provider,
			Model:        model,
			ConnectionID: connID,
			Status:       status,
			Request:      captureRequest(body, headers),
			Tokens:       entry.Tokens,
		}
		if isError {
			capture.Response = map[string]any{"error": sErr.Error()}
		}
		if err := g.detail.Save(capture); err != nil {
			log.Printf("detail save error on %s: %v", endpoint, err)
		}
	}
}
