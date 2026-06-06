package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeDisabledModelStore struct {
	listDisabledModelsErr  error
	listCustomModelsErr    error
	createDisabledModelErr error
	createCustomModelErr   error
	deleteDisabledModelErr error
	deleteCustomModelErr   error
	disabledModels         []store.DisabledModel
	customModels           []store.CustomModel
	disabledModel          *store.DisabledModel
	customModel            *store.CustomModel
}

func (f *fakeDisabledModelStore) ListDisabledModels() ([]store.DisabledModel, error) {
	if f.listDisabledModelsErr != nil {
		return nil, f.listDisabledModelsErr
	}
	return f.disabledModels, nil
}

func (f *fakeDisabledModelStore) CreateDisabledModel(provider, model string) (*store.DisabledModel, error) {
	if f.createDisabledModelErr != nil {
		return nil, f.createDisabledModelErr
	}
	if f.disabledModel != nil {
		return f.disabledModel, nil
	}
	return &store.DisabledModel{ID: "1", Provider: provider, Model: model, CreatedAt: "2024-01-01"}, nil
}

func (f *fakeDisabledModelStore) DeleteDisabledModel(provider, model string) error {
	if f.deleteDisabledModelErr != nil {
		return f.deleteDisabledModelErr
	}
	return nil
}

func (f *fakeDisabledModelStore) IsModelDisabled(provider, model string) (bool, error) {
	return false, nil
}

func (f *fakeDisabledModelStore) ListCustomModels() ([]store.CustomModel, error) {
	if f.listCustomModelsErr != nil {
		return nil, f.listCustomModelsErr
	}
	return f.customModels, nil
}

func (f *fakeDisabledModelStore) CreateCustomModel(provider, model, displayName string) (*store.CustomModel, error) {
	if f.createCustomModelErr != nil {
		return nil, f.createCustomModelErr
	}
	if f.customModel != nil {
		return f.customModel, nil
	}
	return &store.CustomModel{ID: "1", Provider: provider, Model: model, DisplayName: displayName, CreatedAt: "2024-01-01"}, nil
}

func (f *fakeDisabledModelStore) DeleteCustomModel(id string) error {
	if f.deleteCustomModelErr != nil {
		return f.deleteCustomModelErr
	}
	return nil
}

func TestDisabledModelsListStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		DisabledModelsList(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsCreateStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsDeleteStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodDelete, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsListStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsList(ctx, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsCreateStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsDeleteStoreNil(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, nil, nil, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsListMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		DisabledModelsList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsCreateMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsDeleteMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsListMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsList(ctx, s)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsCreateMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsDeleteMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, s, &fakeAuthAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsListStoreError(t *testing.T) {
	fake := &fakeDisabledModelStore{listDisabledModelsErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		DisabledModelsList(ctx, fake)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsListStoreError(t *testing.T) {
	fake := &fakeDisabledModelStore{listCustomModelsErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsList(ctx, fake)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsCreateDuplicate(t *testing.T) {
	s := newHandlerStore(t)

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("first create status = %d, want 201", ctx.Response.StatusCode())
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("duplicate status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsCreateDuplicate(t *testing.T) {
	s := newHandlerStore(t)

	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("first create status = %d, want 201", ctx.Response.StatusCode())
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("duplicate status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsCreateInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsCreateMissingFields(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsDeleteEmptyID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, s, &fakeAuthAuditWriter{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsDeleteInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodDelete, `{"provider":`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsDeleteMissingFields(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodDelete, `{"provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsCreateStoreError(t *testing.T) {
	fake := &fakeDisabledModelStore{createDisabledModelErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, fake, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsCreateStoreError(t *testing.T) {
	fake := &fakeDisabledModelStore{createCustomModelErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, fake, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsDeleteStoreError(t *testing.T) {
	fake := &fakeDisabledModelStore{deleteDisabledModelErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodDelete, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, fake, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsDeleteStoreError(t *testing.T) {
	fake := &fakeDisabledModelStore{deleteCustomModelErr: store.ErrNotFound}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, fake, &fakeAuthAuditWriter{}, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsCreateAuditError(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsCreateAuditError(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsDeleteAuditError(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201", ctx.Response.StatusCode())
	}

	ctx, body := runHandler(t, fasthttp.MethodDelete, `{"provider":"openai","model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsDeleteAuditError(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"provider":"openai","model":"gpt-custom"}`, func(ctx *fasthttp.RequestCtx) {
		CustomModelsCreate(ctx, s, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201", ctx.Response.StatusCode())
	}
	var created customModelResponse
	decodeJSON(t, ctx.Response.Body(), &created)

	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, s, &fakeAuthAuditWriter{appendErr: errors.New("audit error")}, created.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestDisabledModelsDeleteStoreErrorNotFound(t *testing.T) {
	fake := &fakeDisabledModelStore{deleteDisabledModelErr: store.ErrNotFound}
	ctx, body := runHandler(t, fasthttp.MethodDelete, `{"provider":"missing","model":"model"}`, func(ctx *fasthttp.RequestCtx) {
		DisabledModelsDelete(ctx, fake, &fakeAuthAuditWriter{})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestCustomModelsDeleteStoreErrorGeneric(t *testing.T) {
	fake := &fakeDisabledModelStore{deleteCustomModelErr: errors.New("db error")}
	ctx, body := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		CustomModelsDelete(ctx, fake, &fakeAuthAuditWriter{}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}
