package handlers

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestLogsListsRequestLogs(t *testing.T) {
	s := openHandlerTestStore(t)
	entry := handlerUsageEntry("req-log", "openai", "gpt-4o", time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC))
	entry.StatusCode = intPtr(200)
	entry.LatencyMS = intPtr(123)
	entry.ClientTool = stringPtr("codex")
	logHandlerEntries(t, s, []store.RequestLogEntry{entry})

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?model=gpt-4o")
	Logs(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Object string `json:"object"`
		Data   []struct {
			RequestID  string  `json:"request_id"`
			Model      string  `json:"model"`
			StatusCode *int    `json:"status_code"`
			LatencyMS  *int    `json:"latency_ms"`
			ClientTool *string `json:"client_tool"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal logs: %v", err)
	}
	if decoded.Object != "list" || len(decoded.Data) != 1 {
		t.Fatalf("decoded = %+v, want one list entry", decoded)
	}
	if decoded.Data[0].RequestID != "req-log" || decoded.Data[0].Model != "gpt-4o" {
		t.Fatalf("log entry = %+v, want req-log/gpt-4o", decoded.Data[0])
	}
	if decoded.Data[0].StatusCode == nil || *decoded.Data[0].StatusCode != 200 {
		t.Fatalf("status code = %v, want 200", decoded.Data[0].StatusCode)
	}
	if decoded.Data[0].LatencyMS == nil || *decoded.Data[0].LatencyMS != 123 {
		t.Fatalf("latency = %v, want 123", decoded.Data[0].LatencyMS)
	}
	if decoded.Data[0].ClientTool == nil || *decoded.Data[0].ClientTool != "codex" {
		t.Fatalf("client tool = %v, want codex", decoded.Data[0].ClientTool)
	}
}

func TestLogsRejectsInvalidLimit(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?limit=-1")

	Logs(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestLogsIncludesTotalIgnoringLimit(t *testing.T) {
	s := openHandlerTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	var entries []store.RequestLogEntry
	for i := 0; i < 5; i++ {
		entries = append(entries, handlerUsageEntry("req-"+string(rune('0'+i)), "openai", "gpt-4o", base.Add(time.Duration(i)*time.Minute)))
	}
	logHandlerEntries(t, s, entries)

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?limit=2")
	Logs(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Data  []struct{} `json:"data"`
		Limit int        `json:"limit"`
		Total int        `json:"total"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Data) != 2 {
		t.Fatalf("data len = %d, want 2 (limited)", len(decoded.Data))
	}
	if decoded.Limit != 2 {
		t.Fatalf("limit = %d, want 2", decoded.Limit)
	}
	if decoded.Total != 5 {
		t.Fatalf("total = %d, want 5 (ignores limit)", decoded.Total)
	}
}

func TestLogsFilterByStatusClass(t *testing.T) {
	s := openHandlerTestStore(t)
	base := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	ok := handlerUsageEntry("ok", "openai", "gpt-4o", base)
	ok.StatusCode = intPtr(200)
	failed := handlerUsageEntry("failed", "openai", "gpt-4o", base.Add(time.Minute))
	failed.StatusCode = intPtr(500)
	logHandlerEntries(t, s, []store.RequestLogEntry{ok, failed})

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?status_class=server_error")
	Logs(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var decoded struct {
		Data []struct {
			RequestID string `json:"request_id"`
		} `json:"data"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Total != 1 || len(decoded.Data) != 1 || decoded.Data[0].RequestID != "failed" {
		t.Fatalf("decoded = %+v, want only failed", decoded)
	}
}

func TestLogsRejectsInvalidStatusClass(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?status_class=teapot")

	Logs(ctx, s)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

type failingLogsStore struct {
	err error
}

func (f failingLogsStore) GetUsage(filter store.UsageFilter) ([]store.RequestLogEntry, error) {
	return nil, f.err
}

func (f failingLogsStore) GetUsageSummary(filter store.UsageFilter) (*store.UsageSummary, error) {
	return &store.UsageSummary{}, nil
}

func (f failingLogsStore) CountUsage(filter store.UsageFilter) (int, error) {
	return 0, f.err
}
func (f failingLogsStore) GetUsageChart(period, granularity string, now time.Time) (*store.UsageChart, error) {
	return nil, f.err
}

func TestLogsStoreFailureRedacted(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs")
	Logs(ctx, failingLogsStore{err: errors.New("sql: database is locked at /secret/path")})

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if strings.Contains(string(ctx.Response.Body()), "secret") {
		t.Fatalf("response leaks internal error: %s", ctx.Response.Body())
	}
}

func stringPtr(value string) *string {
	return &value
}
