package server

import (
	"errors"
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

		// x-g0-vk virtual-key gate (PAR-ROUTE-030/031): resolver and quota
		// adapters keep the api package free of store/governance imports.
		vkGate := api.NewVKGate(newVKResolverAdapter(st), newVKQuotaAdapter(governance.NewQuotaEngine(st, time.Now)))
		chat.SetVKGate(vkGate)
		messages.SetVKGate(vkGate)
		responses.SetVKGate(vkGate)
		embeddings.SetVKGate(vkGate)
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
	modelSet := map[string]struct{}{}
	for _, pc := range vk.ProviderConfigs {
		for _, m := range pc.AllowedModels {
			modelSet[m] = struct{}{}
		}
	}
	for m := range modelSet {
		info.AllowedModels = append(info.AllowedModels, m)
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
