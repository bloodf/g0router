package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func Settings(ctx *fasthttp.RequestCtx, s *store.Store) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		settings, err := s.GetSettings()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("get settings: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, settings)
	case fasthttp.MethodPut:
		var settings store.Settings
		if err := json.Unmarshal(ctx.PostBody(), &settings); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.UpdateSettings(settings); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("update settings: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, settings)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
