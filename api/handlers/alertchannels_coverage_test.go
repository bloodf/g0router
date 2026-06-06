package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestAlertChannelsPutInvalidID(t *testing.T) {
	fs := &fakeAlertChannelStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"ops","channel_type":"webhook","config":{},"events":[]}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsPutInvalidJSON(t *testing.T) {
	fs := &fakeAlertChannelStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsPutMissingName(t *testing.T) {
	fs := &fakeAlertChannelStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"channel_type":"webhook"}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsPutMissingChannelType(t *testing.T) {
	fs := &fakeAlertChannelStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"ops"}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsPutGetError(t *testing.T) {
	fs := &fakeAlertChannelStore{getErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"ops","channel_type":"webhook","config":{},"events":[]}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsDeleteInvalidID(t *testing.T) {
	fs := &fakeAlertChannelStore{}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsDeleteNotFound(t *testing.T) {
	fs := &fakeAlertChannelStore{deleteErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsTestEndpointDispatchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	fs := &fakeAlertChannelStore{
		getResult: &store.AlertChannel{
			ID:          1,
			Name:        "ops",
			ChannelType: "webhook",
			Config:      `{"url":"` + server.URL + `"}`,
			Events:      []string{"quota_depleted"},
			IsActive:    true,
		},
	}

	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannelsTest(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsTestEndpointNoURL(t *testing.T) {
	fs := &fakeAlertChannelStore{
		getResult: &store.AlertChannel{
			ID:          1,
			Name:        "ops",
			ChannelType: "webhook",
			Config:      `{}`,
			Events:      []string{"quota_depleted"},
			IsActive:    true,
		},
	}

	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannelsTest(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}
