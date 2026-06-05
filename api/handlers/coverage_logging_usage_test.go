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

// ---- Logs handler coverage ----

func TestLogsNilStoreReturns503(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs")
	Logs(ctx, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestLogsInvalidStartReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?start=bad-date")
	Logs(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestLogsInvalidEndReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?end=bad-date")
	Logs(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestLogsInvalidFromReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?from=bad-date")
	Logs(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestLogsInvalidToReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?to=bad-date")
	Logs(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestLogsInvalidStatusClassReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?status_class=bogus")
	Logs(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

// failingCountStore: GetUsage succeeds, CountUsage fails.
type failingCountStore struct{}

func (f failingCountStore) GetUsage(filter store.UsageFilter) ([]store.RequestLogEntry, error) {
	return []store.RequestLogEntry{}, nil
}
func (f failingCountStore) GetUsageSummary(filter store.UsageFilter) (*store.UsageSummary, error) {
	return &store.UsageSummary{}, nil
}
func (f failingCountStore) CountUsage(filter store.UsageFilter) (int, error) {
	return 0, errors.New("count: db locked at /secret/path")
}

func TestLogsCountFailureReturns500Redacted(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs")
	Logs(ctx, failingCountStore{})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
	if strings.Contains(string(ctx.Response.Body()), "secret") {
		t.Fatalf("response leaks internal error: %s", ctx.Response.Body())
	}
}

func TestLogsIncludesTotal(t *testing.T) {
	s := openHandlerTestStore(t)
	base := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		e := handlerUsageEntry("req-"+string(rune('a'+i)), "openai", "gpt-4o", base.Add(time.Duration(i)*time.Minute))
		logHandlerEntries(t, s, []store.RequestLogEntry{e})
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/logs?limit=1")
	Logs(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 3 {
		t.Fatalf("total = %d, want 3", resp.Total)
	}
}

// ---- Usage handler coverage ----

func TestUsageNilStoreReturns503(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage")
	Usage(ctx, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestUsageInvalidStatusClassReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?status_class=teapot")
	Usage(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestUsageInvalidStartReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?start=not-rfc3339")
	Usage(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageInvalidEndReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?end=not-rfc3339")
	Usage(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageInvalidToReturns400(t *testing.T) {
	s := openHandlerTestStore(t)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?to=not-rfc3339")
	Usage(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestUsageGetFailureReturns500Redacted(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage")
	Usage(ctx, failingLogsStore{err: errors.New("db error at /secret/path")})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
	if strings.Contains(string(ctx.Response.Body()), "secret") {
		t.Fatalf("response leaks internal error: %s", ctx.Response.Body())
	}
}

func TestUsageCountFailureReturns500Redacted(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage")
	Usage(ctx, failingCountStore{})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
	if strings.Contains(string(ctx.Response.Body()), "secret") {
		t.Fatalf("response leaks internal error: %s", ctx.Response.Body())
	}
}

func TestUsageIncludesTotal(t *testing.T) {
	s := openHandlerTestStore(t)
	base := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		e := handlerUsageEntry("req-"+string(rune('a'+i)), "openai", "gpt-4o", base.Add(time.Duration(i)*time.Minute))
		logHandlerEntries(t, s, []store.RequestLogEntry{e})
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?limit=2")
	Usage(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	var resp struct {
		Total int `json:"total"`
		Data  []struct{} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 4 {
		t.Fatalf("total = %d, want 4", resp.Total)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len = %d, want 2 (limited)", len(resp.Data))
	}
}

func TestUsageValidRFC3339StartEndFilter(t *testing.T) {
	s := openHandlerTestStore(t)
	base := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	inside := handlerUsageEntry("inside", "openai", "gpt-4o", base)
	outside := handlerUsageEntry("outside", "openai", "gpt-4o", base.Add(2*time.Hour))
	logHandlerEntries(t, s, []store.RequestLogEntry{inside, outside})

	start := base.Add(-time.Minute).Format(time.RFC3339)
	end := base.Add(time.Minute).Format(time.RFC3339)
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/usage?start="+start+"&end="+end)
	Usage(ctx, s)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var resp struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("total = %d, want 1", resp.Total)
	}
}
