package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeAlertChannelStore struct {
	listErr      error
	createResult *store.AlertChannel
	createErr    error
	getResult    *store.AlertChannel
	getErr       error
	updateErr    error
	deleteErr    error
}

func (f *fakeAlertChannelStore) ListAlertChannels() ([]store.AlertChannel, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return nil, nil
}

func (f *fakeAlertChannelStore) CreateAlertChannel(name, channelType, config string, events []string, isActive bool) (*store.AlertChannel, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult != nil {
		return f.createResult, nil
	}
	return &store.AlertChannel{ID: 1, Name: name, ChannelType: channelType, Config: config, Events: events, IsActive: isActive}, nil
}

func (f *fakeAlertChannelStore) GetAlertChannel(id int64) (*store.AlertChannel, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getResult, nil
}

func (f *fakeAlertChannelStore) UpdateAlertChannel(id int64, name, channelType, config string, events []string, isActive bool) error {
	return f.updateErr
}

func (f *fakeAlertChannelStore) DeleteAlertChannel(id int64) error {
	return f.deleteErr
}

func TestAlertChannelsCreateListGetUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)
	s.SetEncKey("test-key")

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"ops","channel_type":"webhook","config":{"url":"https://hooks.example.com"},"events":["quota_depleted"],"is_active":true}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created struct {
		ID          int64    `json:"id"`
		Name        string   `json:"name"`
		ChannelType string   `json:"channel_type"`
		Config      map[string]any `json:"config"`
		Events      []string `json:"events"`
		IsActive    bool     `json:"is_active"`
	}
	decodeJSON(t, body, &created)
	if created.ID == 0 || created.Name != "ops" {
		t.Fatalf("created = %+v", created)
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var listed struct {
		Data []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Name != "ops" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Get
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got struct {
		Name string `json:"name"`
	}
	decodeJSON(t, body, &got)
	if got.Name != "ops" {
		t.Fatalf("name = %q", got.Name)
	}

	// Update
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"name":"ops-updated","channel_type":"discord","config":{"webhook_url":"https://discord.com"},"events":["budget_exhausted"],"is_active":false}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		Name     string `json:"name"`
		IsActive bool   `json:"is_active"`
	}
	decodeJSON(t, body, &updated)
	if updated.Name != "ops-updated" {
		t.Fatalf("name = %q", updated.Name)
	}
	if updated.IsActive {
		t.Fatal("expected inactive")
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	// Get after delete
	ctx, _ = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsMissingName(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"channel_type":"webhook"}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsMissingChannelType(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"ops"}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsGetInvalidID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsPutMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"ops"}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsDeleteMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsListError(t *testing.T) {
	fs := &fakeAlertChannelStore{listErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsCreateError(t *testing.T) {
	fs := &fakeAlertChannelStore{createErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"ops","channel_type":"webhook","config":{},"events":[]}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsUpdateNotFound(t *testing.T) {
	fs := &fakeAlertChannelStore{updateErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"ops","channel_type":"webhook","config":{},"events":[]}`, func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsDeleteError(t *testing.T) {
	fs := &fakeAlertChannelStore{deleteErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannels(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsTestEndpoint(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := json.Marshal(map[string]any{"ok": true})
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
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

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannelsTest(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var res struct {
		Success bool `json:"success"`
	}
	decodeJSON(t, body, &res)
	if !res.Success {
		t.Fatalf("expected success, body=%s", body)
	}
	_ = receivedBody
}

func TestAlertChannelsTestEndpointInactive(t *testing.T) {
	fs := &fakeAlertChannelStore{
		getResult: &store.AlertChannel{
			ID:          1,
			Name:        "ops",
			ChannelType: "webhook",
			Config:      `{"url":"https://example.com"}`,
			Events:      []string{"quota_depleted"},
			IsActive:    false,
		},
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannelsTest(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", ctx.Response.StatusCode(), body)
	}
	var res struct {
		Error string `json:"error"`
	}
	decodeJSON(t, body, &res)
	if res.Error == "" {
		t.Fatal("expected error message")
	}
}

func TestAlertChannelsTestEndpointInvalidID(t *testing.T) {
	fs := &fakeAlertChannelStore{}
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannelsTest(ctx, fs, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestAlertChannelsTestEndpointNotFound(t *testing.T) {
	fs := &fakeAlertChannelStore{getErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		AlertChannelsTest(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}
