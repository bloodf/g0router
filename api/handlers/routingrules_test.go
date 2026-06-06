package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeRoutingRuleStore struct {
	listErr   error
	createErr error
	getRule   *store.RoutingRule
	getErr    error
	updateErr error
	deleteErr error
}

func (f *fakeRoutingRuleStore) ListRoutingRules() ([]store.RoutingRule, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return nil, nil
}

func (f *fakeRoutingRuleStore) CreateRoutingRule(name string, priority int, condField, condOperator, condValue, targetProvider string, targetModel *string) (*store.RoutingRule, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &store.RoutingRule{ID: 1, Name: name, Priority: priority, CondField: condField, CondOperator: condOperator, CondValue: condValue, TargetProvider: targetProvider, TargetModel: targetModel, IsActive: true}, nil
}

func (f *fakeRoutingRuleStore) GetRoutingRule(id int64) (*store.RoutingRule, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getRule, nil
}

func (f *fakeRoutingRuleStore) UpdateRoutingRule(id int64, name string, priority int, condField, condOperator, condValue, targetProvider string, targetModel *string, isActive bool) error {
	return f.updateErr
}

func (f *fakeRoutingRuleStore) DeleteRoutingRule(id int64) error {
	return f.deleteErr
}

func TestRoutingRulesCreateListGetUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"rule-1","priority":10,"cond_field":"model","cond_operator":"equals","cond_value":"gpt-4o","target_provider":"openai","target_model":"gpt-4o"}`, func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, body, &created)
	if created.ID == 0 {
		t.Fatal("expected rule id")
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, s, "")
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
	if len(listed.Data) != 1 || listed.Data[0].Name != "rule-1" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Get
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got struct {
		Name string `json:"name"`
	}
	decodeJSON(t, body, &got)
	if got.Name != "rule-1" {
		t.Fatalf("name = %q, want rule-1", got.Name)
	}

	// Update
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"name":"rule-1-updated","priority":20,"cond_field":"provider","cond_operator":"contains","cond_value":"azure","target_provider":"azure","target_model":"gpt-4o","is_active":false}`, func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		Name     string `json:"name"`
		Priority int    `json:"priority"`
		IsActive bool   `json:"is_active"`
	}
	decodeJSON(t, body, &updated)
	if updated.Name != "rule-1-updated" {
		t.Fatalf("name = %q, want rule-1-updated", updated.Name)
	}
	if updated.Priority != 20 {
		t.Fatalf("priority = %d, want 20", updated.Priority)
	}
	if updated.IsActive {
		t.Fatal("expected is_active=false")
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}
}

func TestRoutingRulesValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing name", `{"priority":0,"cond_field":"model","cond_operator":"equals","cond_value":"x","target_provider":"openai"}`},
		{"missing cond_field", `{"name":"r","priority":0,"cond_operator":"equals","cond_value":"x","target_provider":"openai"}`},
		{"missing cond_operator", `{"name":"r","priority":0,"cond_field":"model","cond_value":"x","target_provider":"openai"}`},
		{"missing target_provider", `{"name":"r","priority":0,"cond_field":"model","cond_operator":"equals","cond_value":"x"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := runHandler(t, fasthttp.MethodPost, tc.body, func(ctx *fasthttp.RequestCtx) {
				RoutingRules(ctx, &fakeRoutingRuleStore{}, "")
			})
			if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
				t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
			}
		})
	}
}

func TestRoutingRulesStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesListError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{listErr: errors.New("boom")}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesGetNotFound(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{getErr: store.ErrNotFound}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesUpdateNotFound(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"x","priority":0,"cond_field":"model","cond_operator":"equals","cond_value":"x","target_provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{updateErr: store.ErrNotFound}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesDeleteError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{deleteErr: errors.New("boom")}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesCreateError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"r","priority":0,"cond_field":"model","cond_operator":"equals","cond_value":"x","target_provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{createErr: errors.New("boom")}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesGetInvalidID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{}, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesPutInvalidID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"x","priority":0,"cond_field":"model","cond_operator":"equals","cond_value":"x","target_provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{}, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesPutGenericError(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"x","priority":0,"cond_field":"model","cond_operator":"equals","cond_value":"x","target_provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{updateErr: errors.New("boom")}, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesPutMissingID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"x","priority":0,"cond_field":"model","cond_operator":"equals","cond_value":"x","target_provider":"openai"}`, func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesDeleteInvalidID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{}, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestRoutingRulesDeleteMissingID(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		RoutingRules(ctx, &fakeRoutingRuleStore{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}
