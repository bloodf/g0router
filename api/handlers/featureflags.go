package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type featureFlagView struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func newFeatureFlagView(f store.FeatureFlag) featureFlagView {
	return featureFlagView{
		ID:          f.ID,
		Key:         f.Key,
		Enabled:     f.Enabled,
		Description: f.Description,
		CreatedAt:   f.CreatedAt,
	}
}

type toggleFeatureFlagRequest struct {
	Enabled bool `json:"enabled"`
}

type featureFlagStore interface {
	ListFeatureFlags() ([]store.FeatureFlag, error)
	GetFeatureFlag(id int64) (*store.FeatureFlag, error)
	GetFeatureFlagByKey(key string) (*store.FeatureFlag, error)
	ToggleFeatureFlag(id int64, enabled bool) error
}

func FeatureFlags(ctx *fasthttp.RequestCtx, s featureFlagStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			flags, err := s.ListFeatureFlags()
			if err != nil {
				log.Printf("list feature flags: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list feature flags")
				return
			}
			views := make([]featureFlagView, 0, len(flags))
			for _, f := range flags {
				views = append(views, newFeatureFlagView(f))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[featureFlagView]{Data: views})
			return
		}
		flagID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid feature flag id")
			return
		}
		f, err := s.GetFeatureFlag(flagID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "feature flag not found")
				return
			}
			log.Printf("get feature flag: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to get feature flag")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newFeatureFlagView(*f))
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "feature flag id required")
			return
		}
		flagID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid feature flag id")
			return
		}
		var req toggleFeatureFlagRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.ToggleFeatureFlag(flagID, req.Enabled); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "feature flag not found")
				return
			}
			log.Printf("toggle feature flag: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to toggle feature flag")
			return
		}
		updated, err := s.GetFeatureFlag(flagID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "feature flag not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newFeatureFlagView(*updated))
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
