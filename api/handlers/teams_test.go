package handlers

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeTeamStore struct {
	listErr    error
	createErr  error
	getTeam    *store.Team
	getErr     error
	updateErr  error
	deleteErr  error
}

func (f *fakeTeamStore) ListTeams() ([]store.Team, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return nil, nil
}

func (f *fakeTeamStore) CreateTeam(name string, budgetUSD *float64, budgetPeriod string, rateLimitRPM *int) (*store.Team, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &store.Team{ID: 1, Name: name}, nil
}

func (f *fakeTeamStore) GetTeam(id int64) (*store.Team, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getTeam, nil
}

func (f *fakeTeamStore) UpdateTeam(id int64, name string, budgetUSD *float64, budgetPeriod string, rateLimitRPM *int) error {
	return f.updateErr
}

func (f *fakeTeamStore) DeleteTeam(id int64) error {
	return f.deleteErr
}

func TestTeamsCreateListGetUpdateDelete(t *testing.T) {
	s := newHandlerStore(t)

	// Create
	ctx, body := runHandler(t, fasthttp.MethodPost, `{"name":"engineering","budget_usd":100,"budget_period":"monthly","rate_limit_rpm":1000}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", ctx.Response.StatusCode(), body)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	decodeJSON(t, body, &created)
	if created.ID == 0 {
		t.Fatal("expected team id")
	}

	// List
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
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
	if len(listed.Data) != 1 || listed.Data[0].Name != "engineering" {
		t.Fatalf("listed = %+v", listed.Data)
	}

	// Get
	ctx, body = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var got struct {
		Name string `json:"name"`
	}
	decodeJSON(t, body, &got)
	if got.Name != "engineering" {
		t.Fatalf("name = %q, want engineering", got.Name)
	}

	// Update
	ctx, body = runHandler(t, fasthttp.MethodPut, `{"name":"eng-updated","budget_usd":200,"budget_period":"weekly","rate_limit_rpm":500}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var updated struct {
		Name         string `json:"name"`
		BudgetPeriod string `json:"budget_period"`
	}
	decodeJSON(t, body, &updated)
	if updated.Name != "eng-updated" {
		t.Fatalf("name = %q, want eng-updated", updated.Name)
	}
	if updated.BudgetPeriod != "weekly" {
		t.Fatalf("budget_period = %q, want weekly", updated.BudgetPeriod)
	}

	// Delete
	ctx, body = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), body)
	}

	// Get after delete
	ctx, _ = runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestTeamsInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestTeamsMissingName(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"budget_usd":10}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestTeamsDuplicateName(t *testing.T) {
	s := newHandlerStore(t)
	if _, err := s.CreateTeam("dup", nil, "monthly", nil); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"dup"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("status = %d, want 409", ctx.Response.StatusCode())
	}
}

func TestTeamsGetInvalidID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "abc")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestTeamsPutMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"x"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestTeamsDeleteMissingID(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestTeamsMethodNotAllowed(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestTeamsStoreNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestTeamsListError(t *testing.T) {
	fs := &fakeTeamStore{listErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestTeamsCreateError(t *testing.T) {
	fs := &fakeTeamStore{createErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"eng"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestTeamsUpdateNotFound(t *testing.T) {
	fs := &fakeTeamStore{updateErr: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"eng"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestTeamsUpdateGetAfterUpdateNotFound(t *testing.T) {
	fs := &fakeTeamStore{getTeam: &store.Team{ID: 1, Name: "eng"}}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"eng"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestTeamsDeleteError(t *testing.T) {
	fs := &fakeTeamStore{deleteErr: errors.New("db error")}
	ctx, _ := runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestTeamsUpdateConflict(t *testing.T) {
	fs := &fakeTeamStore{updateErr: errors.New("UNIQUE constraint failed: teams.name")}
	ctx, _ := runHandler(t, fasthttp.MethodPut, `{"name":"dup"}`, func(ctx *fasthttp.RequestCtx) {
		Teams(ctx, fs, "1")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("status = %d, want 409", ctx.Response.StatusCode())
	}
}
