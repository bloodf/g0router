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

// CustomModel is a user-defined model read from the customModels setting.
type CustomModel struct {
	ID       string
	Provider string
	Type     string
}

// CustomModelLister returns custom models for /v1/models merging (PAR-ROUTE-057).
type CustomModelLister interface {
	ListCustomModels() ([]CustomModel, error)
}

// AliasModelLister returns alias names for /v1/models merging (PAR-ROUTE-057).
type AliasModelLister interface {
	ListAliasNames() ([]string, error)
}

// SubConfigModel is a TTS/embedding model declared inside a connection's metadata.
type SubConfigModel struct {
	ID         string
	Kind       string
	ProviderID string
}

// SubConfigModelReader returns sub-config models for /v1/models (PAR-ROUTE-058).
type SubConfigModelReader interface {
	ListSubConfigModels() ([]SubConfigModel, error)
}

// LiveModel is a dynamic per-account model resolved live (Kiro/Qoder) for
// /v1/models merging (PAR-ROUTE-056).
type LiveModel struct {
	ID       string
	Provider string
}

// LiveCatalogLister returns dynamic per-account models for /v1/models
// (PAR-ROUTE-056). A non-nil error degrades to static-only silently (the live
// resolver may fail; route.js:296-298), so List does NOT 500 on this error.
type LiveCatalogLister interface {
	ListLiveModels() ([]LiveModel, error)
}

// PseudoModel is a web search/fetch pseudo-model ({alias}/search, {alias}/fetch)
// for /v1/models exposure (PAR-ROUTE-059).
type PseudoModel struct {
	ID      string
	OwnedBy string
}

// PseudoModelLister returns web search/fetch pseudo-models for /v1/models
// (PAR-ROUTE-059). It only EXPOSES the pseudo-models; serving them is a
// follow-up (ESC-WEB-EXEC).
type PseudoModelLister interface {
	ListPseudoModels() ([]PseudoModel, error)
}

// ModelsHandler handles GET /v1/models and GET /v1/models/:id.
type ModelsHandler struct {
	router            *inference.Router
	disabledChecker   DisabledChecker
	comboLister       ComboLister
	customModelLister CustomModelLister
	aliasModelLister  AliasModelLister
	subConfigReader   SubConfigModelReader
	liveCatalogLister LiveCatalogLister
	pseudoModelLister PseudoModelLister
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

// SetCustomModelLister wires a custom-model lister for /v1/models (PAR-ROUTE-057).
func (h *ModelsHandler) SetCustomModelLister(l CustomModelLister) {
	h.customModelLister = l
}

// SetAliasModelLister wires an alias lister for /v1/models (PAR-ROUTE-057).
func (h *ModelsHandler) SetAliasModelLister(l AliasModelLister) {
	h.aliasModelLister = l
}

// SetSubConfigModelReader wires a sub-config model reader for /v1/models (PAR-ROUTE-058).
func (h *ModelsHandler) SetSubConfigModelReader(r SubConfigModelReader) {
	h.subConfigReader = r
}

// SetLiveCatalogLister wires a live-catalog lister for /v1/models (PAR-ROUTE-056).
func (h *ModelsHandler) SetLiveCatalogLister(l LiveCatalogLister) {
	h.liveCatalogLister = l
}

// SetPseudoModelLister wires a web search/fetch pseudo-model lister for
// /v1/models (PAR-ROUTE-059).
func (h *ModelsHandler) SetPseudoModelLister(l PseudoModelLister) {
	h.pseudoModelLister = l
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
	seen := make(map[string]bool)
	for _, m := range resp.Data {
		seen[m.ID] = true
	}

	// Aggregate models from all Stage-1 provider catalogs, skipping disabled ones.
	// Iterate in sorted key order so that owned_by assignment is deterministic when
	// the same model ID appears in multiple provider catalogs.
	providerIDs := make([]string, 0, len(catalog.Providers))
	for id := range catalog.Providers {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)
	var catalogEntries []schemas.ModelEntry
	for _, providerID := range providerIDs {
		for _, m := range catalog.ModelsFor(providerID) {
			if m.ID == "" || seen[m.ID] {
				continue
			}
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
			seen[m.ID] = true
			catalogEntries = append(catalogEntries, schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: providerID,
			})
		}
	}

	// Sort the provider-model section; combo entries keep their list order.
	sort.Slice(catalogEntries, func(i, j int) bool {
		return catalogEntries[i].ID < catalogEntries[j].ID
	})
	resp.Data = append(resp.Data, catalogEntries...)

	// Merge custom models (PAR-ROUTE-057). Order follows route.js:358:
	// catalog IDs are seeded into the seen set first, so colliding custom IDs are skipped.
	if h.customModelLister != nil {
		customModels, err := h.customModelLister.ListCustomModels()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "failed to list custom models", nil)
			return
		}
		for _, m := range customModels {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			if m.Type != "" && m.Type != "llm" {
				continue
			}
			seen[m.ID] = true
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: m.Provider,
			})
		}
	}

	// Merge alias names (PAR-ROUTE-057). Alias entries are appended after custom entries.
	if h.aliasModelLister != nil {
		aliasNames, err := h.aliasModelLister.ListAliasNames()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "failed to list alias models", nil)
			return
		}
		for _, name := range aliasNames {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      name,
				Object:  "model",
				OwnedBy: "alias",
			})
		}
	}

	// Merge sub-config models (PAR-ROUTE-058). Sub-config entries are appended after
	// alias entries, matching route.js:364-383 placement.
	if h.subConfigReader != nil {
		subModels, err := h.subConfigReader.ListSubConfigModels()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "server_error", "failed to list sub-config models", nil)
			return
		}
		for _, m := range subModels {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			resp.Data = append(resp.Data, schemas.ModelEntry{
				ID:      m.ID,
				Object:  "model",
				OwnedBy: m.ProviderID,
			})
		}
	}

	// Merge live-catalog dynamic models (PAR-ROUTE-056). The live resolver may
	// fail (network); per route.js:296-298 a failure degrades to static-only
	// SILENTLY — do NOT 500. Catalog IDs were seeded into seen first, so colliding
	// live IDs are skipped.
	if h.liveCatalogLister != nil {
		if liveModels, err := h.liveCatalogLister.ListLiveModels(); err == nil {
			for _, m := range liveModels {
				if m.ID == "" || seen[m.ID] {
					continue
				}
				seen[m.ID] = true
				resp.Data = append(resp.Data, schemas.ModelEntry{
					ID:      m.ID,
					Object:  "model",
					OwnedBy: m.Provider,
				})
			}
		}
	}

	// Expose web search/fetch pseudo-models (PAR-ROUTE-059): {alias}/search and
	// {alias}/fetch when a provider has the config. Best-effort exposure only
	// (serving them is a follow-up, ESC-WEB-EXEC); an error degrades silently.
	if h.pseudoModelLister != nil {
		if pseudoModels, err := h.pseudoModelLister.ListPseudoModels(); err == nil {
			for _, m := range pseudoModels {
				if m.ID == "" || seen[m.ID] {
					continue
				}
				seen[m.ID] = true
				resp.Data = append(resp.Data, schemas.ModelEntry{
					ID:      m.ID,
					Object:  "model",
					OwnedBy: m.OwnedBy,
				})
			}
		}
	}

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
