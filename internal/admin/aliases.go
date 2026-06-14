package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type aliasDTO struct {
	ID       string `json:"id"`
	Alias    string `json:"alias"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func toAliasDTO(rec *store.AliasRecord) aliasDTO {
	return aliasDTO{
		ID:       rec.ID,
		Alias:    rec.Alias,
		Provider: rec.Provider,
		Model:    rec.Model,
	}
}

type aliasRequest struct {
	Alias    string `json:"alias"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// mirrorResolutionAlias best-effort writes the admin alias into the gateway
// model_aliases resolver table (alias → provider/model) so admin edits feed the
// live resolver. A failure is logged and never fails the parent request: the
// binding contract is the id-keyed admin DTO, not the mirror write.
func (h *Handlers) mirrorResolutionAlias(alias, provider, model string) {
	target := model
	if provider != "" && model != "" {
		target = provider + "/" + model
	}
	if alias == "" || target == "" || alias == target {
		return
	}
	if err := h.store.CreateAlias(alias, target); err != nil {
		log.Printf("aliases: mirror resolution alias %q failed: %v", alias, err)
	}
}

// ListAliases handles GET /api/aliases.
func (h *Handlers) ListAliases(ctx *fasthttp.RequestCtx) {
	recs, err := h.store.ListAliasRecords()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list aliases")
		return
	}
	out := make([]aliasDTO, 0, len(recs))
	for _, rec := range recs {
		out = append(out, toAliasDTO(rec))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateAlias handles POST /api/aliases.
func (h *Handlers) CreateAlias(ctx *fasthttp.RequestCtx) {
	var req aliasRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Alias == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "alias is required")
		return
	}
	created, err := h.store.CreateAliasRecord(&store.AliasRecord{
		Alias:    req.Alias,
		Provider: req.Provider,
		Model:    req.Model,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create alias")
		return
	}
	h.mirrorResolutionAlias(created.Alias, created.Provider, created.Model)
	h.recordAudit(ctx, "create_alias", created.ID, fmt.Sprintf("Created alias %s", created.Alias))
	writeData(ctx, fasthttp.StatusCreated, toAliasDTO(created))
}

// GetAlias handles GET /api/aliases/{id}.
func (h *Handlers) GetAlias(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	rec, err := h.store.GetAliasRecordByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alias not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load alias")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toAliasDTO(rec))
}

// UpdateAlias handles PUT /api/aliases/{id}.
func (h *Handlers) UpdateAlias(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req aliasRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Alias == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "alias is required")
		return
	}
	existing, err := h.store.GetAliasRecordByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alias not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load alias")
		return
	}
	existing.Alias = req.Alias
	existing.Provider = req.Provider
	existing.Model = req.Model
	if err := h.store.UpdateAliasRecord(existing); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update alias")
		return
	}
	updated, err := h.store.GetAliasRecordByID(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load alias")
		return
	}
	h.mirrorResolutionAlias(updated.Alias, updated.Provider, updated.Model)
	h.recordAudit(ctx, "update_alias", updated.ID, fmt.Sprintf("Updated alias %s", updated.Alias))
	writeData(ctx, fasthttp.StatusOK, toAliasDTO(updated))
}

// DeleteAlias handles DELETE /api/aliases/{id}.
func (h *Handlers) DeleteAlias(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	if err := h.store.DeleteAliasRecord(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alias not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete alias")
		return
	}
	h.recordAudit(ctx, "delete_alias", id, "Deleted alias")
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Alias deleted successfully"})
}
