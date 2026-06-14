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
	// The OIDC client secret is encrypted at rest: route it to the encrypted
	// accessor and strip it from the plaintext settings map so it is never
	// persisted (or echoed) as plaintext.
	if secret, ok := values["oidc_client_secret"]; ok {
		if secret != "" {
			if err := h.store.SetOIDCSecret(secret); err != nil {
				writeError(ctx, fasthttp.StatusInternalServerError, "save settings")
				return
			}
		}
		delete(values, "oidc_client_secret")
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
