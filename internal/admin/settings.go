package admin

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// GetSettings handles GET /api/settings.
func (h *Handlers) GetSettings(ctx *fasthttp.RequestCtx) {
	settings, err := h.store.GetSettings()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load settings")
		return
	}
	writeData(ctx, fasthttp.StatusOK, settings)
}

// PutSettings handles PUT /api/settings with a flat key→value JSON object.
func (h *Handlers) PutSettings(ctx *fasthttp.RequestCtx) {
	var values map[string]string
	if err := json.Unmarshal(ctx.PostBody(), &values); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.store.SetSettings(values); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "save settings")
		return
	}
	settings, err := h.store.GetSettings()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load settings")
		return
	}
	writeData(ctx, fasthttp.StatusOK, settings)
}
