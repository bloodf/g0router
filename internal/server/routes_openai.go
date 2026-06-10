package server

import (
	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/fasthttp/router"
)

// RegisterOpenAIRoutes adds /v1/* routes to the router.
func RegisterOpenAIRoutes(r *router.Router, router_ *inference.Router) {
	chat := api.NewChatHandler(router_)
	messages := api.NewMessagesHandler(router_)
	embeddings := api.NewEmbeddingsHandler(router_)
	models := api.NewModelsHandler(router_)

	r.POST("/v1/chat/completions", chat.Handle)
	r.POST("/v1/messages", messages.Handle)
	r.POST("/v1/embeddings", embeddings.Handle)
	r.GET("/v1/models", models.List)
	r.GET("/v1/models/{id}", models.Get)
}
