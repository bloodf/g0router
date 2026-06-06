package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeChatStore struct {
	listErr    error
	getErr     error
	createErr  error
	updateErr  error
	deleteErr  error
	session    *store.ChatSession
	sessions   []store.ChatSession
}

func (f *fakeChatStore) ListChatSessions() ([]store.ChatSession, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.sessions, nil
}
func (f *fakeChatStore) GetChatSession(id string) (*store.ChatSession, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.session == nil {
		return nil, errors.New("not found")
	}
	return f.session, nil
}
func (f *fakeChatStore) CreateChatSession(title, model, provider, messagesJSON string) (*store.ChatSession, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &store.ChatSession{ID: "1", Title: title, Model: model, Provider: provider}, nil
}
func (f *fakeChatStore) UpdateChatSession(id string, title, messagesJSON *string) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	return nil
}
func (f *fakeChatStore) DeleteChatSession(id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return nil
}

func TestChatSessionListStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionList(ctx, &fakeChatStore{listErr: errors.New("boom")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestChatSessionGetStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionGet(ctx, &fakeChatStore{getErr: errors.New("boom")}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestChatSessionCreateStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"gpt-4","provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionCreate(ctx, &fakeChatStore{createErr: errors.New("boom")}, &fakeAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestChatSessionUpdateStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"title":"new"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionUpdate(ctx, &fakeChatStore{updateErr: errors.New("boom")}, &fakeAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestChatSessionDeleteStoreError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionDelete(ctx, &fakeChatStore{deleteErr: errors.New("boom")}, &fakeAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestChatSessionUpdateNothingToUpdate(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionUpdate(ctx, &fakeChatStore{}, &fakeAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestChatSessionListNilStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionList(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestChatSessionGetNilStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionGet(ctx, nil, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestChatSessionGetEmptyID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionGet(ctx, &fakeChatStore{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestChatSessionCreateNilStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"gpt-4","provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionCreate(ctx, nil, &fakeAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestChatSessionUpdateNilStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"title":"new"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionUpdate(ctx, nil, &fakeAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestChatSessionUpdateEmptyID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"title":"new"}`, func(ctx *fasthttp.RequestCtx) {
		ChatSessionUpdate(ctx, &fakeChatStore{}, &fakeAuditWriter{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestChatSessionDeleteNilStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionDelete(ctx, nil, &fakeAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestChatSessionDeleteNilAudit(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionDelete(ctx, &fakeChatStore{}, nil, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestChatSessionDeleteEmptyID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ChatSessionDelete(ctx, &fakeChatStore{}, &fakeAuditWriter{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}
