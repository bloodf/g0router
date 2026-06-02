package handlers

import (
	"encoding/json"
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

func stringPtr(value string) *string {
	return &value
}
