package admin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type teamDTO struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	BudgetUSD     float64 `json:"budget_usd"`
	BudgetUsedUSD float64 `json:"budget_used_usd"`
	BudgetPeriod  string  `json:"budget_period"`
	RateLimitRPM  int     `json:"rate_limit_rpm"`
}

func toTeamDTO(t *store.Team) teamDTO {
	return teamDTO{
		ID:            t.ID,
		Name:          t.Name,
		BudgetUSD:     t.BudgetUSD,
		BudgetUsedUSD: t.BudgetUsedUSD,
		BudgetPeriod:  t.BudgetPeriod,
		RateLimitRPM:  t.RateLimitRPM,
	}
}

type teamRequest struct {
	Name         string  `json:"name"`
	BudgetUSD    float64 `json:"budget_usd"`
	BudgetPeriod string  `json:"budget_period"`
	RateLimitRPM int     `json:"rate_limit_rpm"`
}

func validateTeamRequest(req *teamRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.BudgetUSD < 0 {
		return fmt.Errorf("budget_usd must be non-negative")
	}
	if req.BudgetPeriod != "" && req.BudgetPeriod != "daily" && req.BudgetPeriod != "weekly" && req.BudgetPeriod != "monthly" {
		return fmt.Errorf("budget_period must be daily, weekly, or monthly")
	}
	if req.RateLimitRPM < 0 {
		return fmt.Errorf("rate_limit_rpm must be non-negative")
	}
	return nil
}

// ListTeams handles GET /api/teams. The response data is a bare array.
func (h *Handlers) ListTeams(ctx *fasthttp.RequestCtx) {
	teams, err := h.store.ListTeams()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list teams")
		return
	}
	out := make([]teamDTO, 0, len(teams))
	for _, t := range teams {
		out = append(out, toTeamDTO(t))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateTeam handles POST /api/teams.
func (h *Handlers) CreateTeam(ctx *fasthttp.RequestCtx) {
	var req teamRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateTeamRequest(&req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	created, err := h.store.CreateTeam(&store.Team{
		Name:         req.Name,
		BudgetUSD:    req.BudgetUSD,
		BudgetPeriod: req.BudgetPeriod,
		RateLimitRPM: req.RateLimitRPM,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create team")
		return
	}
	h.recordAudit(ctx, "create_team", created.Name, "Created team "+created.Name)
	writeData(ctx, fasthttp.StatusCreated, toTeamDTO(created))
}

// GetTeam handles GET /api/teams/{id}.
func (h *Handlers) GetTeam(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	t, err := h.store.GetTeamByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "team not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load team")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toTeamDTO(t))
}

// UpdateTeam handles PUT /api/teams/{id}.
func (h *Handlers) UpdateTeam(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req teamRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateTeamRequest(&req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.store.GetTeamByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "team not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load team")
		return
	}

	if err := h.store.UpdateTeam(&store.Team{
		ID:            id,
		Name:          req.Name,
		BudgetUSD:     req.BudgetUSD,
		BudgetUsedUSD: existing.BudgetUsedUSD,
		BudgetPeriod:  req.BudgetPeriod,
		RateLimitRPM:  req.RateLimitRPM,
	}); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update team")
		return
	}

	updated, err := h.store.GetTeamByID(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load team")
		return
	}
	h.recordAudit(ctx, "update_team", updated.Name, "Updated team "+updated.Name)
	writeData(ctx, fasthttp.StatusOK, toTeamDTO(updated))
}

// DeleteTeam handles DELETE /api/teams/{id}.
func (h *Handlers) DeleteTeam(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.store.GetTeamByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "team not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load team")
		return
	}
	if err := h.store.DeleteTeam(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "team not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete team")
		return
	}
	h.recordAudit(ctx, "delete_team", existing.Name, "Deleted team "+existing.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Team deleted successfully"})
}
