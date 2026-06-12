package server

import (
	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/store"
	"github.com/fasthttp/router"
)

// RegisterOpenAIRoutes adds /v1/* routes to the router.
func RegisterOpenAIRoutes(r *router.Router, router_ *inference.Router, st *store.Store, refresher api.CredentialRefresher, comboDisp api.ComboDispatcher) {
	chat := api.NewChatHandler(router_)
	if refresher != nil {
		chat.SetCredentialRefresher(refresher)
	}
	if comboDisp != nil {
		chat.SetComboDispatcher(comboDisp)
	}
	messages := api.NewMessagesHandler(router_)
	responses := api.NewResponsesHandler(router_)
	embeddings := api.NewEmbeddingsHandler(router_)
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
