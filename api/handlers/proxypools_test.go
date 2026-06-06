package handlers

import (
	"bytes"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestProxyPoolListGetCreateUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"us-east-1","protocol":"http","host":"proxy.example.com","port":8080,"username":"user1","password":"secret123"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoPassword(t, body)
	var created struct {
		Data proxyPoolResponse `json:"data"`
	}
	decodeJSON(t, body, &created)
	if created.Data.ID == "" || created.Data.Name != "us-east-1" {
		t.Fatalf("created = %+v", created.Data)
	}

	// Verify password stored
	stored, err := s.GetProxyPool(created.Data.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if stored.Password != "secret123" {
		t.Fatalf("stored password = %q, want secret123", stored.Password)
	}

	// Audit for create
	entry := lastAuditEntry(t, s, "proxy_pool.create")
	if entry == nil || entry.Target != "us-east-1" {
		t.Fatalf("audit entry for create = %+v", entry)
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoPassword(t, body)
	var listed struct {
		Data []proxyPoolResponse `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].ID != created.Data.ID {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Get
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolGet(ctx, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoPassword(t, body)
	var got struct {
		Data proxyPoolResponse `json:"data"`
	}
	decodeJSON(t, body, &got)
	if got.Data.ID != created.Data.ID {
		t.Fatalf("got = %+v", got.Data)
	}

	// Update partial
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"name":"us-east-2"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	assertNoPassword(t, body)
	var updated struct {
		Data proxyPoolResponse `json:"data"`
	}
	decodeJSON(t, body, &updated)
	if updated.Data.Name != "us-east-2" || updated.Data.Protocol != "http" {
		t.Fatalf("updated = %+v", updated.Data)
	}

	// Verify password still preserved after partial update
	stored, err = s.GetProxyPool(created.Data.ID)
	if err != nil {
		t.Fatalf("GetProxyPool after update: %v", err)
	}
	if stored.Password != "secret123" {
		t.Fatalf("stored password after partial update = %q, want secret123", stored.Password)
	}

	// Audit for update
	entry = lastAuditEntry(t, s, "proxy_pool.update")
	if entry == nil || entry.Target != created.Data.ID {
		t.Fatalf("audit entry for update = %+v", entry)
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolDelete(ctx, s, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	// Audit for delete
	entry = lastAuditEntry(t, s, "proxy_pool.delete")
	if entry == nil || entry.Target != created.Data.ID {
		t.Fatalf("audit entry for delete = %+v", entry)
	}

	// Get after delete should 404
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolGet(ctx, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolCreateInvalidProtocol(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"test","protocol":"ftp","host":"host.com","port":8080}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
	if !bytes.Contains(body, []byte("protocol")) {
		t.Fatalf("expected protocol error, got: %s", body)
	}
}

func TestProxyPoolCreateInvalidPort(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"test","protocol":"http","host":"host.com","port":99999}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
	if !bytes.Contains(body, []byte("port")) {
		t.Fatalf("expected port error, got: %s", body)
	}
}

func TestProxyPoolUpdateInvalidProtocol(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := store.ProxyPool{
		Name:     "test",
		Protocol: "http",
		Host:     "host.com",
		Port:     8080,
	}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"protocol":"ftp"}`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolUpdate(ctx, s, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolBatchImport(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	lines := `{"lines":["proxy.example.com:8080","socks5://user:pass@socks.example.com:1080","http://http.example.com:3128","invalid-line","http://badport:99999"]}`
	ctx, body := runHandler(t, fasthttp.MethodPost, lines, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolBatchImport(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("batch status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var resp batchImportResponse
	decodeJSON(t, body, &resp)
	if len(resp.Created) != 3 {
		t.Fatalf("created = %d, want 3; body=%s", len(resp.Created), body)
	}
	if len(resp.Errors) != 2 {
		t.Fatalf("errors = %d, want 2; body=%s", len(resp.Errors), body)
	}

	// Check auto-generated names
	if resp.Created[0].Name != "proxy-proxy.example.com-8080" {
		t.Fatalf("name[0] = %q, want proxy-proxy.example.com-8080", resp.Created[0].Name)
	}
	if resp.Created[1].Name != "proxy-socks.example.com-1080" {
		t.Fatalf("name[1] = %q, want proxy-socks.example.com-1080", resp.Created[1].Name)
	}
	if resp.Created[1].Protocol != "socks5" {
		t.Fatalf("protocol[1] = %q, want socks5", resp.Created[1].Protocol)
	}
	if resp.Created[1].Username != "user" {
		t.Fatalf("username[1] = %q, want user", resp.Created[1].Username)
	}

	// Verify password stored for socks5 entry
	stored, err := s.GetProxyPool(resp.Created[1].ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if stored.Password != "pass" {
		t.Fatalf("stored password = %q, want pass", stored.Password)
	}

	// Audit for batch
	entry := lastAuditEntry(t, s, "proxy_pool.batch_create")
	if entry == nil || entry.Target != "3" {
		t.Fatalf("audit entry for batch = %+v", entry)
	}
}

func TestProxyPoolTest(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := store.ProxyPool{
		Name:     "test",
		Protocol: "http",
		Host:     "host.com",
		Port:     8080,
	}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolTest(ctx, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("test status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var resp struct {
		Ok        bool   `json:"ok"`
		LatencyMs int    `json:"latency_ms"`
		Error     string `json:"error"`
	}
	decodeJSON(t, body, &resp)
	if !resp.Ok {
		t.Fatalf("ok = false, want true")
	}
}

func TestProxyPoolTestNotFound(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		ProxyPoolTest(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolCreateInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		ProxyPoolCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestProxyPoolStoreUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func(*fasthttp.RequestCtx)
	}{
		{"list", func(ctx *fasthttp.RequestCtx) { ProxyPoolList(ctx, nil) }},
		{"get", func(ctx *fasthttp.RequestCtx) { ProxyPoolGet(ctx, nil, "1") }},
		{"create", func(ctx *fasthttp.RequestCtx) { ProxyPoolCreate(ctx, nil, nil) }},
		{"update", func(ctx *fasthttp.RequestCtx) { ProxyPoolUpdate(ctx, nil, nil, "1") }},
		{"delete", func(ctx *fasthttp.RequestCtx) { ProxyPoolDelete(ctx, nil, nil, "1") }},
		{"test", func(ctx *fasthttp.RequestCtx) { ProxyPoolTest(ctx, nil, "1") }},
		{"batch", func(ctx *fasthttp.RequestCtx) { ProxyPoolBatchImport(ctx, nil, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, tc.fn)
			if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
			}
		})
	}
}

func assertNoPassword(t *testing.T, body []byte) {
	t.Helper()
	if bytes.Contains(body, []byte(`"password"`)) {
		t.Fatalf("response contains password field: %s", body)
	}
}
