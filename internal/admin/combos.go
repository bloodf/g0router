package admin

import (
	"encoding/json"
	"regexp"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// comboNameRe validates combo names per the API-route rule (PAR-ROUTE-004).
// Mirrors /^[a-zA-Z0-9_.\-]+$/ in src/app/api/combos/route.js:7.
var comboNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

func comboResponse(c *store.Combo) map[string]any {
	models := c.Models
	if models == nil {
		models = []string{}
	}
	return map[string]any{"name": c.Name, "models": models}
}

// ListCombos handles GET /api/combos.
func (h *Handlers) ListCombos(ctx *fasthttp.RequestCtx) {
	combos, err := h.store.ListCombos()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list combos")
		return
	}
	out := make([]map[string]any, 0, len(combos))
	for _, c := range combos {
		out = append(out, comboResponse(c))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateCombo handles POST /api/combos.
// Body: {"name": "...", "models": ["..."]}
func (h *Handlers) CreateCombo(ctx *fasthttp.RequestCtx) {
	var req struct {
		Name   string   `json:"name"`
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if !comboNameRe.MatchString(req.Name) {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid combo name: must match ^[a-zA-Z0-9_.\\-]+$")
		return
	}
	if req.Models == nil {
		req.Models = []string{}
	}
	c := &store.Combo{Name: req.Name, Models: req.Models}
	if err := h.store.CreateCombo(c); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create combo")
		return
	}
	writeData(ctx, fasthttp.StatusCreated, comboResponse(c))
}

// UpdateCombo handles PUT /api/combos/{name}.
// Body: {"models": ["..."]}
func (h *Handlers) UpdateCombo(ctx *fasthttp.RequestCtx) {
	name, ok := ctx.UserValue("name").(string)
	if !ok || name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "missing combo name")
		return
	}
	var req struct {
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Models == nil {
		req.Models = []string{}
	}
	if err := h.store.UpdateCombo(name, req.Models); err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "combo not found")
		return
	}
	writeData(ctx, fasthttp.StatusOK, comboResponse(&store.Combo{Name: name, Models: req.Models}))
}

// DeleteCombo handles DELETE /api/combos/{name}.
func (h *Handlers) DeleteCombo(ctx *fasthttp.RequestCtx) {
	name, ok := ctx.UserValue("name").(string)
	if !ok || name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "missing combo name")
		return
	}
	if err := h.store.DeleteCombo(name); err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "combo not found")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]bool{"success": true})
}
