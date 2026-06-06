package handlers

import (
	"encoding/json"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestChatSessionListGetCreateUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"title":"Test Session","model":"gpt-4","provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created struct {
		Data chatSessionResponse `json:"data"`
	}
	decodeJSON(t, body, &created)
	if created.Data.ID == "" || created.Data.Model != "gpt-4" || created.Data.Provider != "openai" {
		t.Fatalf("created = %+v", created.Data)
	}
	if string(created.Data.Messages) != "[]" {
		t.Fatalf("messages = %s, want []", created.Data.Messages)
	}

	// Audit for create
	entry := lastAuditEntry(t, s, "chat_session.create")
	if entry == nil || entry.Target != created.Data.ID {
		t.Fatalf("audit entry for create = %+v", entry)
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []chatSessionListItem `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].ID != created.Data.ID {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Get
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionGet(ctx, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got struct {
		Data chatSessionResponse `json:"data"`
	}
	decodeJSON(t, body, &got)
	if got.Data.ID != created.Data.ID {
		t.Fatalf("got = %+v", got.Data)
	}

	// Update title
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"title":"Updated Title"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionUpdate(ctx, s, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		Data chatSessionResponse `json:"data"`
	}
	decodeJSON(t, body, &updated)
	if updated.Data.Title != "Updated Title" {
		t.Fatalf("updated title = %q, want Updated Title", updated.Data.Title)
	}

	// Audit for update
	entry = lastAuditEntry(t, s, "chat_session.update")
	if entry == nil || entry.Target != created.Data.ID {
		t.Fatalf("audit entry for update = %+v", entry)
	}

	// Update messages
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"messages":[{"role":"user","content":"hello"}]}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionUpdate(ctx, s, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update messages status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	decodeJSON(t, body, &updated)
	var msgs []map[string]any
	if err := json.Unmarshal(updated.Data.Messages, &msgs); err != nil {
		t.Fatalf("messages unmarshal: %v; body=%s", err, updated.Data.Messages)
	}
	if len(msgs) != 1 || msgs[0]["role"] != "user" {
		t.Fatalf("messages = %+v", msgs)
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionDelete(ctx, s, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	// Audit for delete
	entry = lastAuditEntry(t, s, "chat_session.delete")
	if entry == nil || entry.Target != created.Data.ID {
		t.Fatalf("audit entry for delete = %+v", entry)
	}

	// Get after delete should 404
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionGet(ctx, s, created.Data.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestChatSessionCreateMissingModel(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestChatSessionCreateMissingProvider(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"model":"gpt-4"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionCreate(ctx, s, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestChatSessionUpdateInvalidMessages(t *testing.T) {
	s := newHandlerStore(t)

	created, err := s.CreateChatSession("test", "gpt-4", "openai", "[]")
	if err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"messages":"not valid json"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionUpdate(ctx, s, s, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestChatSessionGetNotFound(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionGet(ctx, s, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestChatSessionStoreUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func(*fasthttp.RequestCtx)
	}{
		{"list", func(ctx *fasthttp.RequestCtx) { ChatSessionList(ctx, nil) }},
		{"get", func(ctx *fasthttp.RequestCtx) { ChatSessionGet(ctx, nil, "1") }},
		{"create", func(ctx *fasthttp.RequestCtx) { ChatSessionCreate(ctx, nil, nil) }},
		{"update", func(ctx *fasthttp.RequestCtx) { ChatSessionUpdate(ctx, nil, nil, "1") }},
		{"delete", func(ctx *fasthttp.RequestCtx) { ChatSessionDelete(ctx, nil, nil, "1") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, body := runHandler(t, fasthttp.MethodPost, `{}`, tc.fn)
			if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
			}
		})
	}
}
