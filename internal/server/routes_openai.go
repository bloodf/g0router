package server

import (
	"github.com/bloodf/g0router/internal/api"
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
	}

	r.POST("/v1/chat/completions", chat.Handle)
	r.POST("/v1/messages", messages.Handle)
	r.POST("/v1/responses", responses.Handle)
	r.POST("/v1/embeddings", embeddings.Handle)
	r.GET("/v1/models", models.List)
	r.GET("/v1/models/test/{kind}", models.GetTestByKind)
	r.GET("/v1/models/{param}", models.GetOrByKind)
}
