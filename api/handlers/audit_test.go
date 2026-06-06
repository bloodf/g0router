package handlers

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeAuditStore struct {
	entries []store.AuditEntry
	total   int
	err     error
}

func (f *fakeAuditStore) ListAudit(filter store.AuditFilter) ([]store.AuditEntry, int, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.entries, f.total, nil
}

func newAuditCtx(method, uri string) *fasthttp.RequestCtx {
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(uri)
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, &net.TCPAddr{}, nil)
	return ctx
}

func TestAuditStoreNil(t *testing.T) {
	ctx := newAuditCtx(fasthttp.MethodGet, "/api/audit")
	Audit(ctx, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAuditMethodNotAllowed(t *testing.T) {
	ctx := newAuditCtx(fasthttp.MethodPost, "/api/audit")
	Audit(ctx, &fakeAuditStore{})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestAuditInvalidLimit(t *testing.T) {
	ctx := newAuditCtx(fasthttp.MethodGet, "/api/audit?limit=-1")
	Audit(ctx, &fakeAuditStore{})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuditInvalidOffset(t *testing.T) {
	ctx := newAuditCtx(fasthttp.MethodGet, "/api/audit?offset=abc")
	Audit(ctx, &fakeAuditStore{})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAuditListError(t *testing.T) {
	store := &fakeAuditStore{err: errors.New("boom")}
	ctx := newAuditCtx(fasthttp.MethodGet, "/api/audit")
	Audit(ctx, store)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAuditSuccess(t *testing.T) {
	entries := []store.AuditEntry{
		{ID: 1, Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Action: "auth.login", Target: "admin"},
	}
	store := &fakeAuditStore{entries: entries, total: 1}
	ctx := newAuditCtx(fasthttp.MethodGet, "/api/audit?limit=10&offset=0&action=auth.login")
	Audit(ctx, store)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	var resp auditListResponse
	decodeJSON(t, ctx.Response.Body(), &resp)
	if resp.Total != 1 {
		t.Fatalf("total = %d, want 1", resp.Total)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Action != "auth.login" {
		t.Fatalf("action = %q, want auth.login", resp.Data[0].Action)
	}
}

func TestAuditLogResponsesEmpty(t *testing.T) {
	resp := auditLogResponses(nil)
	if len(resp) != 0 {
		t.Fatalf("len = %d, want 0", len(resp))
	}
}

func TestAuditLogResponses(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entries := []store.AuditEntry{
		{ID: 1, Timestamp: now, ActorAPIKeyID: "key1", Action: "test", Target: "user", Details: "detail"},
	}
	resp := auditLogResponses(entries)
	if len(resp) != 1 {
		t.Fatalf("len = %d, want 1", len(resp))
	}
	if resp[0].ID != 1 {
		t.Fatalf("id = %d, want 1", resp[0].ID)
	}
	if resp[0].Timestamp != now.Format(time.RFC3339) {
		t.Fatalf("timestamp = %q, want %q", resp[0].Timestamp, now.Format(time.RFC3339))
	}
}
