package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeFeatureFlagStore struct {
	listErr     error
	flags       []store.FeatureFlag
	getResult   *store.FeatureFlag
	getErr      error
	toggleErr   error
}

func (f *fakeFeatureFlagStore) ListFeatureFlags() ([]store.FeatureFlag, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.flags, nil
}

func (f *fakeFeatureFlagStore) GetFeatureFlag(id int64) (*store.FeatureFlag, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getResult, nil
}

func (f *fakeFeatureFlagStore) ToggleFeatureFlag(id int64, enabled bool) error {
	return f.toggleErr
}

func TestFeatureFlagsList(t *testing.T) {
	s := newHandlerStore(t)

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []struct {
			Key     string `json:"key"`
			Enabled bool   `json:"enabled"`
		} `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 5 {
		t.Fatalf("len = %d, want 5", len(listed.Data))
	}
}

func TestFeatureFlagsToggle(t *testing.T) {
	s := newHandlerStore(t)

	flags, err := s.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, string(rune('0'+flags[0].ID)))
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("toggle status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		Enabled bool `json:"enabled"`
	}
	decodeJSON(t, body, &updated)
	if !updated.Enabled {
		t.Fatalf("expected enabled")
	}
}

func TestFeatureFlagsToggleNotFound(t *testing.T) {
	fs := &fakeFeatureFlagStore{toggleErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, fs, "9999")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsGetNotFound(t *testing.T) {
	fs := &fakeFeatureFlagStore{getErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, fs, "9999")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsInvalidID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsPutMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":`, func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsListError(t *testing.T) {
	fs := &fakeFeatureFlagStore{listErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, fs, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestFeatureFlagsToggleError(t *testing.T) {
	fs := &fakeFeatureFlagStore{toggleErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"enabled":true}`, func(ctx *fasthttp.RequestCtx) {
		FeatureFlags(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}
