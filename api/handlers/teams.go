package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type createTeamRequest struct {
	Name         string   `json:"name"`
	BudgetUSD    *float64 `json:"budget_usd"`
	BudgetPeriod string   `json:"budget_period"`
	RateLimitRPM *int     `json:"rate_limit_rpm"`
}

type updateTeamRequest struct {
	Name         string   `json:"name"`
	BudgetUSD    *float64 `json:"budget_usd"`
	BudgetPeriod string   `json:"budget_period"`
	RateLimitRPM *int     `json:"rate_limit_rpm"`
}

type teamView struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	BudgetUSD     *float64 `json:"budget_usd"`
	BudgetPeriod  string   `json:"budget_period"`
	BudgetUsedUSD float64  `json:"budget_used_usd"`
	RateLimitRPM  *int     `json:"rate_limit_rpm"`
	CreatedAt     string   `json:"created_at"`
}

func newTeamView(team store.Team) teamView {
	return teamView{
		ID:            team.ID,
		Name:          team.Name,
		BudgetUSD:     team.BudgetUSD,
		BudgetPeriod:  team.BudgetPeriod,
		BudgetUsedUSD: team.BudgetUsedUSD,
		RateLimitRPM:  team.RateLimitRPM,
		CreatedAt:     team.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type teamStore interface {
	ListTeams() ([]store.Team, error)
	CreateTeam(name string, budgetUSD *float64, budgetPeriod string, rateLimitRPM *int) (*store.Team, error)
	GetTeam(id int64) (*store.Team, error)
	UpdateTeam(id int64, name string, budgetUSD *float64, budgetPeriod string, rateLimitRPM *int) error
	DeleteTeam(id int64) error
}

func isSQLiteConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func Teams(ctx *fasthttp.RequestCtx, s teamStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			teams, err := s.ListTeams()
			if err != nil {
				log.Printf("list teams: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list teams")
				return
			}
			views := make([]teamView, 0, len(teams))
			for _, team := range teams {
				views = append(views, newTeamView(team))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[teamView]{Data: views})
			return
		}
		teamID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid team id")
			return
		}
		team, err := s.GetTeam(teamID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "team not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newTeamView(*team))
	case fasthttp.MethodPost:
		var req createTeamRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name is required")
			return
		}
		team, err := s.CreateTeam(req.Name, req.BudgetUSD, req.BudgetPeriod, req.RateLimitRPM)
		if err != nil {
			if isSQLiteConstraintError(err) {
				writeError(ctx, fasthttp.StatusConflict, "team name already exists")
				return
			}
			log.Printf("create team: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create team")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, newTeamView(*team))
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "team id required")
			return
		}
		teamID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid team id")
			return
		}
		var req updateTeamRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.UpdateTeam(teamID, req.Name, req.BudgetUSD, req.BudgetPeriod, req.RateLimitRPM); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "team not found")
				return
			}
			if isSQLiteConstraintError(err) {
				writeError(ctx, fasthttp.StatusConflict, "team name already exists")
				return
			}
			log.Printf("update team: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update team")
			return
		}
		updated, err := s.GetTeam(teamID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "team not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newTeamView(*updated))
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "team id required")
			return
		}
		teamID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid team id")
			return
		}
		if err := s.DeleteTeam(teamID); err != nil {
			log.Printf("delete team: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete team")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
