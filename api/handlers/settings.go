package handlers

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type settingsStore interface {
	GetSettings() (store.Settings, error)
	UpdateSettings(store.Settings) error
}

type secretStore interface {
	GetAPIKeySecret() (string, error)
	SetAPIKeySecret(string) error
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
		body := ctx.PostBody()
		var settings store.Settings
		if err := json.Unmarshal(body, &settings); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}

		var rawBody map[string]any
		_ = json.Unmarshal(body, &rawBody)
		if secretVal, ok := rawBody["api_key_secret"].(string); ok && secretVal != "" {
			if ss, ok := s.(secretStore); ok {
				if err := ss.SetAPIKeySecret(secretVal); err != nil {
					log.Printf("set api_key_secret: %v", err)
					writeError(ctx, fasthttp.StatusInternalServerError, "failed to update api_key_secret")
					return
				}
			}
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
			if errors.Is(err, store.ErrRequireLoginNoUsers) {
				writeError(ctx, fasthttp.StatusConflict, err.Error())
				return
			}
			log.Printf("update settings: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update settings")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, settings)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
