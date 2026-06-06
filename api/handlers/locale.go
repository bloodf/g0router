package handlers

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type localeStore interface {
	GetSettings() (store.Settings, error)
	UpdateSettings(store.Settings) error
}

type localeResponse struct {
	Locale string `json:"locale"`
}

// Locale handles GET and POST /api/locale.
func Locale(ctx *fasthttp.RequestCtx, s localeStore) {
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
		writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": localeResponse{Locale: settings.Locale}})
	case fasthttp.MethodPost:
		var req localeResponse
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		req.Locale = strings.TrimSpace(req.Locale)
		if req.Locale == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "locale is required")
			return
		}
		settings, err := s.GetSettings()
		if err != nil {
			log.Printf("get settings: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to get settings")
			return
		}
		settings.Locale = req.Locale
		if err := s.UpdateSettings(settings); err != nil {
			log.Printf("update settings: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update settings")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": localeResponse{Locale: req.Locale}})
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
