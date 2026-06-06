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

type createModelLimitRequest struct {
	Model         string   `json:"model"`
	MaxTokens     *int     `json:"max_tokens"`
	MaxRPM        *int     `json:"max_rpm"`
	AllowedKeyIDs []string `json:"allowed_key_ids"`
}

type updateModelLimitRequest struct {
	Model         string   `json:"model"`
	MaxTokens     *int     `json:"max_tokens"`
	MaxRPM        *int     `json:"max_rpm"`
	AllowedKeyIDs []string `json:"allowed_key_ids"`
}

type modelLimitView struct {
	ID            int64    `json:"id"`
	Model         string   `json:"model"`
	MaxTokens     *int     `json:"max_tokens"`
	MaxRPM        *int     `json:"max_rpm"`
	AllowedKeyIDs []string `json:"allowed_key_ids"`
	CreatedAt     string   `json:"created_at"`
}

func newModelLimitView(limit store.ModelLimit) modelLimitView {
	return modelLimitView{
		ID:            limit.ID,
		Model:         limit.Model,
		MaxTokens:     limit.MaxTokens,
		MaxRPM:        limit.MaxRPM,
		AllowedKeyIDs: limit.AllowedKeyIDs,
		CreatedAt:     limit.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type modelLimitStore interface {
	ListModelLimits() ([]store.ModelLimit, error)
	CreateModelLimit(model string, maxTokens, maxRPM *int, allowedKeyIDs []string) (*store.ModelLimit, error)
	GetModelLimit(id int64) (*store.ModelLimit, error)
	UpdateModelLimit(id int64, model string, maxTokens, maxRPM *int, allowedKeyIDs []string) error
	DeleteModelLimit(id int64) error
}

func ModelLimits(ctx *fasthttp.RequestCtx, s modelLimitStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			limits, err := s.ListModelLimits()
			if err != nil {
				log.Printf("list model limits: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list model limits")
				return
			}
			views := make([]modelLimitView, 0, len(limits))
			for _, limit := range limits {
				views = append(views, newModelLimitView(limit))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[modelLimitView]{Data: views})
			return
		}
		limitID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid model limit id")
			return
		}
		limit, err := s.GetModelLimit(limitID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "model limit not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newModelLimitView(*limit))
	case fasthttp.MethodPost:
		var req createModelLimitRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Model) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "model is required")
			return
		}
		limit, err := s.CreateModelLimit(req.Model, req.MaxTokens, req.MaxRPM, req.AllowedKeyIDs)
		if err != nil {
			if isSQLiteConstraintError(err) {
				writeError(ctx, fasthttp.StatusConflict, "model limit already exists for this model")
				return
			}
			log.Printf("create model limit: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create model limit")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, newModelLimitView(*limit))
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "model limit id required")
			return
		}
		limitID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid model limit id")
			return
		}
		var req updateModelLimitRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.UpdateModelLimit(limitID, req.Model, req.MaxTokens, req.MaxRPM, req.AllowedKeyIDs); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "model limit not found")
				return
			}
			if isSQLiteConstraintError(err) {
				writeError(ctx, fasthttp.StatusConflict, "model limit already exists for this model")
				return
			}
			log.Printf("update model limit: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update model limit")
			return
		}
		updated, err := s.GetModelLimit(limitID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "model limit not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newModelLimitView(*updated))
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "model limit id required")
			return
		}
		limitID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid model limit id")
			return
		}
		if err := s.DeleteModelLimit(limitID); err != nil {
			log.Printf("delete model limit: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete model limit")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
