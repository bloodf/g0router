package api

import (
	"sort"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// DisabledChecker reports whether a model is disabled for a given provider alias.
type DisabledChecker interface {
	IsDisabled(providerAlias, modelID string) (bool, error)
}

// ComboLister returns combo names for /v1/models promotion (PAR-ROUTE-047).
// Returns names only to keep the api layer free of store types.
type ComboLister interface {
	ListComboNames() ([]string, error)
}

// ModelsHandler handles GET /v1/models and GET /v1/models/:id.
type ModelsHandler struct {
	router          *inference.Router
	disabledChecker DisabledChecker
	comboLister     ComboLister
}

// NewModelsHandler creates a models handler.
func NewModelsHandler(router *inference.Router) *ModelsHandler {
	return &ModelsHandler{router: router}
}

// SetDisabledChecker wires a disabled-model checker into the handler.
func (h *ModelsHandler) SetDisabledChecker(dc DisabledChecker) {
	h.disabledChecker = dc
}

// SetComboLister wires a combo lister so combo names appear first in /v1/models (PAR-ROUTE-047).
func (h *ModelsHandler) SetComboLister(cl ComboLister) {
	h.comboLister = cl
}

// List handles GET /v1/models.
func (h *ModelsHandler) List(ctx *fasthttp.RequestCtx) {
	resp := &schemas.ListModelsResponse{
		Object: "list",
	}

	// Combo names appear first (PAR-ROUTE-047).
	if h.comboLister != nil {
		names, err := h.comboLister.ListComboNames()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "failed to list combos", nil)
			return
		}
		for _, name := range names {
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      name,
				Object:  "model",
				OwnedBy: "combo",
			})
		}
	}
	providerStart := len(resp.Data)

	// Aggregate models from all Stage-1 provider catalogs, skipping disabled ones.
	for providerID := range catalog.Providers {
		for _, m := range catalog.ModelsFor(providerID) {
			if h.disabledChecker != nil {
				disabled, err := h.disabledChecker.IsDisabled(providerID, m.ID)
				if err != nil {
					writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "failed to check disabled models", nil)
					return
				}
				if disabled {
					continue
				}
			}
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: providerID,
			})
		}
	}

	// Sort only the provider-model section; combo entries keep their list order.
	sort.Slice(resp.Data[providerStart:], func(i, j int) bool {
		return resp.Data[providerStart+i].ID < resp.Data[providerStart+j].ID
	})

	b, err := jsonMarshal(resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}

// kindSlugMap maps URL kind slugs to catalog Type values (PAR-ROUTE-037).
// Ported from 9router src/app/api/v1/models/[kind]/route.js KIND_SLUG_MAP.
var kindSlugMap = map[string][]string{
	"image":        {"image"},
	"tts":          {"tts"},
	"stt":          {"stt"},
	"embedding":    {"embedding"},
	"image-to-text": {"imageToText"},
	"web":          {"webSearch", "webFetch"},
}

// GetByKind handles GET /v1/models/{kind} — returns catalog models filtered by capability type.
// Valid kinds with no catalog entries return an empty list, not 404.
// Unknown kinds return 404.
func (h *ModelsHandler) GetByKind(ctx *fasthttp.RequestCtx) {
	kind, ok := ctx.UserValue("kind").(string)
	if !ok || kind == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing kind", nil)
		return
	}

	wantTypes, valid := kindSlugMap[kind]
	if !valid {
		writeError(ctx, fasthttp.StatusNotFound, "not_found_error",
			"unknown model kind: "+kind+". Supported: image, tts, stt, embedding, image-to-text, web", nil)
		return
	}

	typeSet := make(map[string]bool, len(wantTypes))
	for _, t := range wantTypes {
		typeSet[t] = true
	}

	resp := &schemas.ListModelsResponse{Object: "list"}
	for providerID := range catalog.Providers {
		for _, m := range catalog.ModelsFor(providerID) {
			mt := m.Type
			if mt == "" {
				mt = "llm"
			}
			if !typeSet[mt] {
				continue
			}
			if h.disabledChecker != nil {
				if disabled, err := h.disabledChecker.IsDisabled(providerID, m.ID); err != nil || disabled {
					continue
				}
			}
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: providerID,
			})
		}
	}
	sort.Slice(resp.Data, func(i, j int) bool {
		return resp.Data[i].ID < resp.Data[j].ID
	})

	b, err := jsonMarshal(resp)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}

// GetOrByKind dispatches GET /v1/models/{param}: if param is a known kind slug it
// filters by kind (PAR-ROUTE-037); otherwise it looks up a model by ID.
func (h *ModelsHandler) GetOrByKind(ctx *fasthttp.RequestCtx) {
	param, ok := ctx.UserValue("param").(string)
	if !ok || param == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing param", nil)
		return
	}
	if _, isKind := kindSlugMap[param]; isKind {
		ctx.SetUserValue("kind", param)
		h.GetByKind(ctx)
	} else {
		ctx.SetUserValue("id", param)
		h.Get(ctx)
	}
}

// kindTestEndpoints maps model kind slugs to the API endpoint used for testing
// that kind (PAR-ROUTE-038). Ported from 9router models/test/ping.js:pingModelByKind.
var kindTestEndpoints = map[string]string{
	"embedding":    "/v1/embeddings",
	"image":        "/v1/images/generations",
	"image-to-text": "/v1/images/generations",
	"tts":          "/v1/audio/speech",
	"stt":          "/v1/audio/transcriptions",
	"web":          "/v1/chat/completions",
	"llm":          "/v1/chat/completions",
}

// ModelTestRoute is the response body for GET /v1/models/test/{kind}.
type ModelTestRoute struct {
	Kind     string `json:"kind"`
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
}

// GetTestByKind handles GET /v1/models/test/{kind} — returns the API endpoint
// to use when pinging a model of the given kind (PAR-ROUTE-038).
func (h *ModelsHandler) GetTestByKind(ctx *fasthttp.RequestCtx) {
	kind, ok := ctx.UserValue("kind").(string)
	if !ok || kind == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing kind", nil)
		return
	}

	endpoint, valid := kindTestEndpoints[kind]
	if !valid {
		writeError(ctx, fasthttp.StatusNotFound, "not_found_error",
			"unknown model kind: "+kind+". Supported: embedding, image, image-to-text, tts, stt, web, llm", nil)
		return
	}

	b, err := jsonMarshal(ModelTestRoute{Kind: kind, Endpoint: endpoint, Method: "POST"})
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
}

// Get handles GET /v1/models/:id.
func (h *ModelsHandler) Get(ctx *fasthttp.RequestCtx) {
	id, ok := ctx.UserValue("id").(string)
	if !ok || id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing model id", nil)
		return
	}

	for providerID := range catalog.Providers {
		for _, m := range catalog.ModelsFor(providerID) {
			if m.ID != id {
				continue
			}
			entry := schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: providerID,
			}
			b, err := jsonMarshal(entry)
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.SetContentTypeBytes([]byte("text/plain"))
				ctx.SetBodyString("internal error")
				return
			}
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetContentTypeBytes([]byte("application/json"))
			ctx.SetBody(b)
			return
		}
	}

	writeError(ctx, fasthttp.StatusNotFound, "not_found_error", "model not found", nil)
}
