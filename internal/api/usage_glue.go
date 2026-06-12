package api

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
