package server

import (
	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
)

// usageRecorderAdapter bridges the w5-b usage.Recorder (which accepts the
// repository-side *store.RequestLogEntry) to the api layer's UsageRecorder
// interface (which accepts api.UsageEntry). This is the same pattern as
// comboDispatcher above: the api layer's consumer interface lets the
// transport stay free of repository imports.
type usageRecorderAdapter struct {
	rec *usage.Recorder
}

func newUsageRecorderAdapter(rec *usage.Recorder) *usageRecorderAdapter {
	return &usageRecorderAdapter{rec: rec}
}

func (a *usageRecorderAdapter) Record(entry *api.UsageEntry) error {
	if entry == nil {
		return nil
	}
	rep := &store.RequestLogEntry{
		Provider:         entry.Provider,
		Model:            entry.Model,
		ConnectionID:     entry.ConnectionID,
		APIKey:           entry.APIKey,
		Endpoint:         entry.Endpoint,
		PromptTokens:     entry.PromptTokens,
		CompletionTokens: entry.CompletionTokens,
		Cost:             entry.Cost,
		Status:           entry.Status,
		Tokens:           entry.Tokens,
	}
	return a.rec.Record(rep)
}

// pendingTrackerAdapter bridges the w5-b usage.Tracker to api.PendingTracker.
type pendingTrackerAdapter struct {
	tr *usage.Tracker
}

func newPendingTrackerAdapter(tr *usage.Tracker) *pendingTrackerAdapter {
	return &pendingTrackerAdapter{tr: tr}
}

func (a *pendingTrackerAdapter) Start(model, provider, connectionID string) {
	a.tr.Start(model, provider, connectionID)
}

func (a *pendingTrackerAdapter) End(model, provider, connectionID string, isError bool) {
	a.tr.End(model, provider, connectionID, isError)
}

// detailCaptureAdapter bridges the w5-c usage.DetailWriter to api.DetailCapture.
type detailCaptureAdapter struct {
	w *usage.DetailWriter
}

func newDetailCaptureAdapter(w *usage.DetailWriter) *detailCaptureAdapter {
	return &detailCaptureAdapter{w: w}
}

func (a *detailCaptureAdapter) Save(capture api.RequestDetailCapture) error {
	return a.w.Save(usage.RequestDetail{
		Provider:         capture.Provider,
		Model:            capture.Model,
		ConnectionID:     capture.ConnectionID,
		Status:           capture.Status,
		Latency:          capture.Latency,
		Tokens:           capture.Tokens,
		Request:          capture.Request,
		ProviderRequest:  capture.ProviderRequest,
		ProviderResponse: capture.ProviderResponse,
		Response:         capture.Response,
	})
}

// Close flushes any buffered request_details rows. Wired into Server.Close()
// so the observability buffer is drained on graceful shutdown.
func (a *detailCaptureAdapter) Close() error {
	return a.w.Close()
}
