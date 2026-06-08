package handlers

import (
	"strconv"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestVirtualKeysCreateListGetUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"prod-key","budget_usd":10,"budget_period":"monthly","rate_limit_rpm":60,"rate_limit_tpm":10000}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created virtualKeyView
	decodeJSON(t, body, &created)
	if created.ID == "" || created.Name != "prod-key" {
		t.Fatalf("created = %+v", created)
	}
	if !strings.HasPrefix(created.Prefix, "gvk-") {
		t.Fatalf("prefix = %q, want gvk- prefix", created.Prefix)
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	// Prefix is intentionally public; ensure the listed entry matches the created key.
	if !strings.Contains(string(body), created.Prefix) {
		t.Fatalf("list missing created key prefix: %s", body)
	}
	var listed struct {
		Data []virtualKeyView `json:"data"`
	}
	decodeJSON(t, body, &listed)
	if len(listed.Data) != 1 || listed.Data[0].Name != "prod-key" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Get
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got struct {
		Name string `json:"name"`
	}
	decodeJSON(t, body, &got)
	if got.Name != "prod-key" {
		t.Fatalf("name = %q, want prod-key", got.Name)
	}

	// Update
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"name":"prod-updated","budget_usd":20,"budget_period":"weekly","rate_limit_rpm":120,"rate_limit_tpm":20000,"is_active":false}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		Name         string `json:"name"`
		BudgetPeriod string `json:"budget_period"`
		IsActive     bool   `json:"is_active"`
	}
	decodeJSON(t, body, &updated)
	if updated.Name != "prod-updated" {
		t.Fatalf("name = %q, want prod-updated", updated.Name)
	}
	if updated.BudgetPeriod != "weekly" {
		t.Fatalf("budget_period = %q, want weekly", updated.BudgetPeriod)
	}
	if updated.IsActive {
		t.Fatal("expected is_active false")
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	// Get after delete
	ctx, _ = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysWithTeam(t *testing.T) {
	s := newHandlerStore(t)
	team, err := s.CreateTeam("eng", nil, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"team-key","team_id":1}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created virtualKeyView
	decodeJSON(t, body, &created)
	if created.TeamID == nil || *created.TeamID != strconv.FormatInt(team.ID, 10) {
		t.Fatalf("team_id = %v, want %d", created.TeamID, team.ID)
	}
}

func TestVirtualKeysInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysMissingName(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"budget_usd":10}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysGetInvalidID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysPutMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"x"}`, func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysDeleteMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestVirtualKeysStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		VirtualKeys(ctx, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}
