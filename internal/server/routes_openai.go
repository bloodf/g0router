package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/store"
	"github.com/fasthttp/router"
)

// RegisterOpenAIRoutes adds /v1/* routes to the router. recorder/tracker/detail
// are the api-layer consumer-interface adapters for the w5-b/c usage glue
// (constructed by New in server.go when a store is present). They may be nil
// — handlers tolerate nil glue so tests and embedders can opt out.
func RegisterOpenAIRoutes(r *router.Router, router_ *inference.Router, st *store.Store, refresher api.CredentialRefresher, comboDisp api.ComboDispatcher, recorder api.UsageRecorder, tracker api.PendingTracker, detail api.DetailCapture) {
	chat := api.NewChatHandler(router_)
	if refresher != nil {
		chat.SetCredentialRefresher(refresher)
	}
	if comboDisp != nil {
		chat.SetComboDispatcher(comboDisp)
	}
	if recorder != nil {
		chat.SetUsageRecorder(recorder)
	}
	if tracker != nil {
		chat.SetPendingTracker(tracker)
	}
	if detail != nil {
		chat.SetDetailCapture(detail)
	}
	messages := api.NewMessagesHandler(router_)
	responses := api.NewResponsesHandler(router_)
	embeddings := api.NewEmbeddingsHandler(router_)
	completions := api.NewCompletionsHandler(router_)
	audio := api.NewAudioHandler(router_)
	images := api.NewImagesHandler(router_)
	files := api.NewFilesHandler(router_)
	batches := api.NewBatchesHandler(router_)
	inputTokens := api.NewInputTokensHandler(router_)
	if recorder != nil {
		messages.SetUsageRecorder(recorder)
		responses.SetUsageRecorder(recorder)
		embeddings.SetUsageRecorder(recorder)
		completions.SetUsageRecorder(recorder)
		audio.SetUsageRecorder(recorder)
		images.SetUsageRecorder(recorder)
		files.SetUsageRecorder(recorder)
		batches.SetUsageRecorder(recorder)
		inputTokens.SetUsageRecorder(recorder)
	}
	if tracker != nil {
		messages.SetPendingTracker(tracker)
		responses.SetPendingTracker(tracker)
		embeddings.SetPendingTracker(tracker)
		completions.SetPendingTracker(tracker)
		audio.SetPendingTracker(tracker)
		images.SetPendingTracker(tracker)
		files.SetPendingTracker(tracker)
		batches.SetPendingTracker(tracker)
		inputTokens.SetPendingTracker(tracker)
	}
	if detail != nil {
		messages.SetDetailCapture(detail)
		responses.SetDetailCapture(detail)
		embeddings.SetDetailCapture(detail)
		completions.SetDetailCapture(detail)
		audio.SetDetailCapture(detail)
		images.SetDetailCapture(detail)
		files.SetDetailCapture(detail)
		batches.SetDetailCapture(detail)
		inputTokens.SetDetailCapture(detail)
	}
	models := api.NewModelsHandler(router_)
	if st != nil {
		models.SetDisabledChecker(st)
		models.SetComboLister(st)
		models.SetCustomModelLister(customModelsAdapter{st: st})
		models.SetAliasModelLister(aliasModelsAdapter{st: st})
		models.SetSubConfigModelReader(subConfigModelsAdapter{st: st})
		// Live model catalog override (PAR-ROUTE-056) + upstream-connection gate
		// (PAR-ROUTE-060). The resolver skips upstream connections and fetches
		// dynamic models via an injectable fetcher. A nil fetcher (the default
		// here) means no live call until a real HTTP fetcher is wired — recorded
		// as an integration follow-up (ESC-LIVE-FETCH) so /v1/models never makes
		// an un-hermetic network call by default.
		models.SetLiveCatalogLister(liveCatalogAdapter{
			resolver: inference.NewLiveCatalogResolver(st, nil),
		})
		// Web search/fetch pseudo-model exposure (PAR-ROUTE-059).
		models.SetPseudoModelLister(pseudoModelsAdapter{st: st})

		// x-g0-vk virtual-key gate (PAR-ROUTE-030/031): resolver and quota
		// adapters keep the api package free of store/governance imports.
		vkGate := api.NewVKGate(newVKResolverAdapter(st), newVKQuotaAdapter(governance.NewQuotaEngine(st, time.Now)))
		chat.SetVKGate(vkGate)
		messages.SetVKGate(vkGate)
		responses.SetVKGate(vkGate)
		embeddings.SetVKGate(vkGate)
		completions.SetVKGate(vkGate)
		audio.SetVKGate(vkGate)
		images.SetVKGate(vkGate)
		files.SetVKGate(vkGate)
		batches.SetVKGate(vkGate)
		inputTokens.SetVKGate(vkGate)

		// VK KeyID pinning selector (PAR-ROUTE-030).
		selector := &vkPinnedSelector{
			st:     st,
			engine: inference.NewSelectionEngine(st, st, nil, time.Now),
			rr:     make(map[string]int),
		}
		chat.SetVKPinnedResolver(selector)
		messages.SetVKPinnedResolver(selector)
		responses.SetVKPinnedResolver(selector)
		embeddings.SetVKPinnedResolver(selector)
		completions.SetVKPinnedResolver(selector)
		audio.SetVKPinnedResolver(selector)
		images.SetVKPinnedResolver(selector)
		files.SetVKPinnedResolver(selector)
		batches.SetVKPinnedResolver(selector)
		inputTokens.SetVKPinnedResolver(selector)
	}

	r.POST("/v1/chat/completions", chat.Handle)
	r.POST("/v1/messages", messages.Handle)
	r.POST("/v1/responses", responses.Handle)
	r.POST("/v1/responses/input_tokens", inputTokens.Handle)
	r.POST("/v1/embeddings", embeddings.Handle)
	r.POST("/v1/completions", completions.Handle)
	r.POST("/v1/audio/speech", audio.Speech)
	r.POST("/v1/audio/transcriptions", audio.Transcription)
	r.POST("/v1/images/generations", images.Generations)
	r.POST("/v1/images/edits", images.Edits)
	r.POST("/v1/images/variations", images.Variations)
	r.POST("/v1/files", files.Upload)
	r.GET("/v1/files", files.List)
	r.GET("/v1/files/{file_id}", files.Retrieve)
	r.DELETE("/v1/files/{file_id}", files.Delete)
	r.GET("/v1/files/{file_id}/content", files.Content)
	r.POST("/v1/batches", batches.Create)
	r.GET("/v1/batches", batches.List)
	r.GET("/v1/batches/{batch_id}", batches.Retrieve)
	r.POST("/v1/batches/{batch_id}/cancel", batches.Cancel)
	r.GET("/v1/models", models.List)
	r.GET("/v1/models/test/{kind}", models.GetTestByKind)
	r.GET("/v1/models/{param}", models.GetOrByKind)
}

// vkResolverAdapter adapts the virtual-key store lookup to the api.VKResolver seam.
type vkResolverAdapter struct {
	st *store.Store
}

func newVKResolverAdapter(st *store.Store) *vkResolverAdapter {
	return &vkResolverAdapter{st: st}
}

func (a *vkResolverAdapter) ResolveVK(key string) (*api.VKInfo, error) {
	vk, err := a.st.GetVirtualKeyByKey(key)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return storeVKToAPI(a.st, vk), nil
}

// storeVKToAPI maps a store virtual key to the api.VKInfo the gate consumes. When
// the VK owns a team (team_id != ""), it loads the team via st.GetTeamByID and
// populates the Team* hierarchy fields (bf-gov-1, D3). A missing/un-teamed VK
// leaves the Team* fields zero so the engine skips the Team tier.
func storeVKToAPI(st *store.Store, vk *store.VirtualKey) *api.VKInfo {
	info := &api.VKInfo{
		Key:          vk.Key,
		BudgetLimit:  0,
		BudgetPeriod: "",
		RateLimitRPM: 0,
		IsActive:     vk.IsActive,
		TeamID:       vk.TeamID,
	}
	if vk.Budget != nil {
		info.BudgetLimit = vk.Budget.Limit
		info.BudgetPeriod = vk.Budget.Period
	}
	if vk.RateLimitRPM != nil {
		info.RateLimitRPM = *vk.RateLimitRPM
	}
	if vk.RateLimit != nil {
		info.TokenMax = vk.RateLimit.TokenMax
		info.TokenResetPeriod = vk.RateLimit.TokenResetPeriod
		info.RequestMax = vk.RateLimit.RequestMax
		info.RequestResetPeriod = vk.RateLimit.RequestResetPeriod
	}
	for _, pc := range vk.ProviderConfigs {
		info.Configs = append(info.Configs, api.VKProviderConfig{
			Provider:          pc.Provider,
			AllowedModels:     pc.AllowedModels,
			BlacklistedModels: pc.BlacklistedModels,
			KeyIDs:            pc.KeyIDs,
			AllowAllKeys:      pc.AllowAllKeys,
		})
	}
	if vk.TeamID != "" && st != nil {
		if team, err := st.GetTeamByID(vk.TeamID); err == nil && team != nil {
			info.TeamBudgetLimit = team.BudgetUSD
			info.TeamBudgetPeriod = team.BudgetPeriod
			info.TeamRateLimitRPM = team.RateLimitRPM
		}
	}
	return info
}

// vkQuotaAdapter adapts the governance quota engine to the api.VKQuotaChecker seam.
type vkQuotaAdapter struct {
	engine *governance.QuotaEngine
}

func newVKQuotaAdapter(engine *governance.QuotaEngine) *vkQuotaAdapter {
	return &vkQuotaAdapter{engine: engine}
}

// Allow maps the api.VKInfo to the governance VirtualKeyInfo and evaluates the
// full dual-dimension chain via engine.Evaluate (bf-gov-3, D8). The typed
// Decision is surfaced live in the gate's {data,error} response: its snake_case
// error.code is derived at the denial-write site from the returned reason via
// api.DecisionCodeForReason (e.g. "token limit exceeded" -> error.code
// "token_limited"). The (bool,int,string) signature is preserved so every
// existing caller is unchanged.
func (a *vkQuotaAdapter) Allow(vk *api.VKInfo, model string) (bool, int, string) {
	res := a.engine.Evaluate(&governance.VirtualKeyInfo{
		Key:                vk.Key,
		BudgetLimit:        vk.BudgetLimit,
		BudgetPeriod:       vk.BudgetPeriod,
		RateLimitRPM:       vk.RateLimitRPM,
		TokenMax:           vk.TokenMax,
		TokenResetPeriod:   vk.TokenResetPeriod,
		RequestMax:         vk.RequestMax,
		RequestResetPeriod: vk.RequestResetPeriod,
		TeamID:             vk.TeamID,
		TeamBudgetLimit:    vk.TeamBudgetLimit,
		TeamBudgetPeriod:   vk.TeamBudgetPeriod,
		TeamRateLimitRPM:   vk.TeamRateLimitRPM,
	}, model)
	return res.Decision == governance.DecisionAllow, res.Status, res.Reason
}

// customModelsAdapter adapts the customModels setting to api.CustomModelLister.
type customModelsAdapter struct {
	st *store.Store
}

func (a customModelsAdapter) ListCustomModels() ([]api.CustomModel, error) {
	raw, err := a.st.GetSetting("customModels")
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get customModels setting: %w", err)
	}
	if raw == "" {
		return nil, nil
	}
	var records []customModelRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		// Malformed setting contributes zero entries (route.js helpers/jsonCol.js).
		return nil, nil
	}
	var out []api.CustomModel
	for _, r := range records {
		if r.ID == "" {
			continue
		}
		if r.Type != "" && r.Type != "llm" {
			continue
		}
		out = append(out, api.CustomModel{
			ID:       r.ID,
			Provider: r.Provider,
			Type:     r.Type,
		})
	}
	return out, nil
}

type customModelRecord struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Type     string `json:"type"`
}

// aliasModelsAdapter adapts the model_aliases table to api.AliasModelLister.
type aliasModelsAdapter struct {
	st *store.Store
}

func (a aliasModelsAdapter) ListAliasNames() ([]string, error) {
	aliases, err := a.st.ListAliases()
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}
	out := make([]string, 0, len(aliases))
	for _, a := range aliases {
		if a.Name == "" {
			continue
		}
		out = append(out, a.Name)
	}
	return out, nil
}

// subConfigModelsAdapter adapts connection metadata to api.SubConfigModelReader.
type subConfigModelsAdapter struct {
	st *store.Store
}

func (a subConfigModelsAdapter) ListSubConfigModels() ([]api.SubConfigModel, error) {
	conns, err := a.st.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	var out []api.SubConfigModel
	for _, c := range conns {
		if c.Metadata == "" {
			continue
		}
		var meta connectionMetadata
		if err := json.Unmarshal([]byte(c.Metadata), &meta); err != nil {
			// Unparseable metadata on one connection contributes no entries (route.js helpers/jsonCol.js).
			continue
		}
		for _, m := range meta.ProviderSpecificData.TTSConfig.Models {
			if m.ID == "" {
				continue
			}
			out = append(out, api.SubConfigModel{ID: m.ID, Kind: "tts", ProviderID: c.ProviderID})
		}
		for _, m := range meta.ProviderSpecificData.EmbeddingConfig.Models {
			if m.ID == "" {
				continue
			}
			out = append(out, api.SubConfigModel{ID: m.ID, Kind: "embedding", ProviderID: c.ProviderID})
		}
	}
	return out, nil
}

type connectionMetadata struct {
	ProviderSpecificData providerSpecificData `json:"providerSpecificData"`
}

type providerSpecificData struct {
	TTSConfig       subConfigContainer `json:"ttsConfig"`
	EmbeddingConfig subConfigContainer `json:"embeddingConfig"`
	WebSearchConfig webPseudoConfig    `json:"webSearchConfig"`
	WebFetchConfig  webPseudoConfig    `json:"webFetchConfig"`
}

// webPseudoConfig carries the alias used to expose {alias}/search and
// {alias}/fetch pseudo-models (PAR-ROUTE-059). The alias names the search/fetch
// surface; enabled gates exposure.
type webPseudoConfig struct {
	Alias   string `json:"alias"`
	Enabled bool   `json:"enabled"`
}

// liveCatalogAdapter adapts inference.LiveCatalogResolver to api.LiveCatalogLister.
type liveCatalogAdapter struct {
	resolver *inference.LiveCatalogResolver
}

func (a liveCatalogAdapter) ListLiveModels() ([]api.LiveModel, error) {
	resolved, err := a.resolver.ResolveLiveModels()
	if err != nil {
		return nil, err
	}
	out := make([]api.LiveModel, 0, len(resolved))
	for _, m := range resolved {
		out = append(out, api.LiveModel{ID: m.ID, Provider: m.Provider})
	}
	return out, nil
}

// pseudoModelsAdapter adapts connection metadata to api.PseudoModelLister: it
// exposes {alias}/search and {alias}/fetch when a connection declares web
// search/fetch config (PAR-ROUTE-059). No network — config only.
type pseudoModelsAdapter struct {
	st *store.Store
}

func (a pseudoModelsAdapter) ListPseudoModels() ([]api.PseudoModel, error) {
	conns, err := a.st.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	var out []api.PseudoModel
	for _, c := range conns {
		if c.Metadata == "" {
			continue
		}
		var meta connectionMetadata
		if err := json.Unmarshal([]byte(c.Metadata), &meta); err != nil {
			// Unparseable metadata contributes no entries (route.js helpers/jsonCol.js).
			continue
		}
		if ws := meta.ProviderSpecificData.WebSearchConfig; ws.Enabled && ws.Alias != "" {
			out = append(out, api.PseudoModel{ID: ws.Alias + "/search", OwnedBy: ws.Alias})
		}
		if wf := meta.ProviderSpecificData.WebFetchConfig; wf.Enabled && wf.Alias != "" {
			out = append(out, api.PseudoModel{ID: wf.Alias + "/fetch", OwnedBy: wf.Alias})
		}
	}
	return out, nil
}

type subConfigContainer struct {
	Models []subConfigModelRecord `json:"models"`
}

type subConfigModelRecord struct {
	ID string `json:"id"`
}

// vkPinnedSelector implements api.VKPinnedKeyResolver by mapping virtual-key KeyIDs
// to real connections via inference.SelectConnection. It round-robins across eligible
// KeyIDs and rejects strategy-fallback results so pinning cannot silently land elsewhere.
type vkPinnedSelector struct {
	st     *store.Store
	engine *inference.SelectionEngine
	mu     sync.Mutex
	rr     map[string]int
}

func (s *vkPinnedSelector) ResolvePinned(providerID, model string, keyIDs []string) (string, string, bool) {
	if len(keyIDs) == 0 {
		return "", "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	key := providerID + ":" + model
	start := s.rr[key] % len(keyIDs)
	for i := range keyIDs {
		idx := (start + i) % len(keyIDs)
		keyID := keyIDs[idx]
		conn, err := s.engine.SelectConnection(providerID, model, nil, keyID)
		if err != nil {
			continue
		}
		if conn.ID != keyID {
			continue
		}
		credential := conn.AccessToken
		if credential == "" {
			credential = conn.Secret
		}
		s.rr[key] = (start + i + 1) % len(keyIDs)
		return conn.ID, credential, true
	}
	return "", "", false
}
