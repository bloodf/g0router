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
	if recorder != nil {
		messages.SetUsageRecorder(recorder)
		responses.SetUsageRecorder(recorder)
		embeddings.SetUsageRecorder(recorder)
	}
	if tracker != nil {
		messages.SetPendingTracker(tracker)
		responses.SetPendingTracker(tracker)
		embeddings.SetPendingTracker(tracker)
	}
	if detail != nil {
		messages.SetDetailCapture(detail)
		responses.SetDetailCapture(detail)
		embeddings.SetDetailCapture(detail)
	}
	models := api.NewModelsHandler(router_)
	if st != nil {
		models.SetDisabledChecker(st)
		models.SetComboLister(st)
		models.SetCustomModelLister(customModelsAdapter{st: st})
		models.SetAliasModelLister(aliasModelsAdapter{st: st})
		models.SetSubConfigModelReader(subConfigModelsAdapter{st: st})

		// x-g0-vk virtual-key gate (PAR-ROUTE-030/031): resolver and quota
		// adapters keep the api package free of store/governance imports.
		vkGate := api.NewVKGate(newVKResolverAdapter(st), newVKQuotaAdapter(governance.NewQuotaEngine(st, time.Now)))
		chat.SetVKGate(vkGate)
		messages.SetVKGate(vkGate)
		responses.SetVKGate(vkGate)
		embeddings.SetVKGate(vkGate)

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
	}

	r.POST("/v1/chat/completions", chat.Handle)
	r.POST("/v1/messages", messages.Handle)
	r.POST("/v1/responses", responses.Handle)
	r.POST("/v1/embeddings", embeddings.Handle)
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
	return storeVKToAPI(vk), nil
}

func storeVKToAPI(vk *store.VirtualKey) *api.VKInfo {
	info := &api.VKInfo{
		Key:          vk.Key,
		BudgetLimit:  0,
		BudgetPeriod: "",
		RateLimitRPM: 0,
		IsActive:     vk.IsActive,
	}
	if vk.Budget != nil {
		info.BudgetLimit = vk.Budget.Limit
		info.BudgetPeriod = vk.Budget.Period
	}
	if vk.RateLimitRPM != nil {
		info.RateLimitRPM = *vk.RateLimitRPM
	}
	for _, pc := range vk.ProviderConfigs {
		info.Configs = append(info.Configs, api.VKProviderConfig{
			Provider:      pc.Provider,
			AllowedModels: pc.AllowedModels,
			KeyIDs:        pc.KeyIDs,
		})
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

func (a *vkQuotaAdapter) Allow(vk *api.VKInfo, model string) (bool, int, string) {
	return a.engine.Allow(&governance.VirtualKeyInfo{
		Key:          vk.Key,
		BudgetLimit:  vk.BudgetLimit,
		BudgetPeriod: vk.BudgetPeriod,
		RateLimitRPM: vk.RateLimitRPM,
	}, model)
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
