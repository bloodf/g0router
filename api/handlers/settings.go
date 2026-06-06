package handlers

import (
	"encoding/json"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type settingsStore interface {
	GetSettings() (store.Settings, error)
	UpdateSettings(store.Settings) error
}

func Settings(ctx *fasthttp.RequestCtx, s settingsStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		settings, err := s.GetSettings()
		if err != nil {
			log.Printf("get settings: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to get settings")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, settings)
	case fasthttp.MethodPut:
		var settings store.Settings
		if err := json.Unmarshal(ctx.PostBody(), &settings); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if settings.LogRetentionDays < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "log_retention_days must be >= 0")
			return
		}
		if settings.LogRetentionDays > 36500 {
			writeError(ctx, fasthttp.StatusBadRequest, "log_retention_days must be <= 36500")
			return
		}
		if settings.CacheTTLSeconds < 0 {
			writeError(ctx, fasthttp.StatusBadRequest, "cache_ttl_seconds must be >= 0")
			return
		}
		if err := s.UpdateSettings(settings); err != nil {
			log.Printf("update settings: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update settings")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, settings)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
