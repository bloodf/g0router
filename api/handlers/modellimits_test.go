package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeModelLimitStore struct {
	listErr   error
	createErr error
	getLimit  *store.ModelLimit
	getErr    error
	updateErr error
	deleteErr error
}

func (f *fakeModelLimitStore) ListModelLimits() ([]store.ModelLimit, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return nil, nil
}

func (f *fakeModelLimitStore) CreateModelLimit(model string, maxTokens, maxRPM *int, allowedKeyIDs []string) (*store.ModelLimit, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &store.ModelLimit{ID: 1, Model: model, MaxTokens: maxTokens, MaxRPM: maxRPM, AllowedKeyIDs: allowedKeyIDs}, nil
}

func (f *fakeModelLimitStore) GetModelLimit(id int64) (*store.ModelLimit, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getLimit, nil
}

func (f *fakeModelLimitStore) UpdateModelLimit(id int64, model string, maxTokens, maxRPM *int, allowedKeyIDs []string) error {
	return f.updateErr
}

func (f *fakeModelLimitStore) DeleteModelLimit(id int64) error {
	return f.deleteErr
}

func TestModelLimitsCreateListGetUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"model":"gpt-4o","max_tokens":4096,"max_rpm":60,"allowed_key_ids":["key-1"]}`, func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created struct {
		ID    int64  `json:"id"`
		Model string `json:"model"`
	}
	decodeJSON(t, body, &created)
	if created.ID == 0 {
		t.Fatal("expected limit id")
	}
	if created.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", created.Model)
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []struct {
			ID    int64  `json:"id"`
			Model string `json:"model"`
		} `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Model != "gpt-4o" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Get
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got struct {
		Model string `json:"model"`
	}
	decodeJSON(t, body, &got)
	if got.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", got.Model)
	}

	// Update
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"model":"gpt-4o","max_tokens":8192,"max_rpm":120,"allowed_key_ids":["key-2"]}`, func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		MaxTokens int `json:"max_tokens"`
	}
	decodeJSON(t, body, &updated)
	if updated.MaxTokens != 8192 {
		t.Fatalf("max_tokens = %d, want 8192", updated.MaxTokens)
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestModelLimitsCreateDuplicate(t *testing.T) {
	s := newHandlerStore(t)

	_, _ = s.CreateModelLimit("gpt-4o", nil, nil, nil)

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("status = %d, want 409", ctx.Response.StatusCode())
	}
}

func TestModelLimitsValidation(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"max_tokens":100}`, func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, &fakeModelLimitStore{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestModelLimitsStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestModelLimitsListError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, &fakeModelLimitStore{listErr: errors.New("boom")}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestModelLimitsGetNotFound(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, &fakeModelLimitStore{getErr: store.ErrNotFound}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestModelLimitsUpdateNotFound(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"model":"x"}`, func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, &fakeModelLimitStore{updateErr: store.ErrNotFound}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestModelLimitsDeleteError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		ModelLimits(ctx, &fakeModelLimitStore{deleteErr: errors.New("boom")}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}
