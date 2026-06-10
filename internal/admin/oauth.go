package admin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// OAuthStart handles GET /api/oauth/{provider}/start.
func (h *Handlers) OAuthStart(ctx *fasthttp.RequestCtx) {
	providerType, ok := pathID(ctx.UserValue("provider"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	flow, ok := h.flows[providerType]
	if !ok {
		writeError(ctx, fasthttp.StatusNotFound, fmt.Sprintf("no oauth flow for provider %q", providerType))
		return
	}

	authURL, state, err := flow.Start()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "start oauth flow")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"auth_url": authURL,
		"state":    state,
	})
}

// OAuthCallback handles POST /api/oauth/{provider}/callback. It exchanges
// the authorization code for tokens and stores them as an oauth connection.
func (h *Handlers) OAuthCallback(ctx *fasthttp.RequestCtx) {
	providerType, ok := pathID(ctx.UserValue("provider"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	flow, ok := h.flows[providerType]
	if !ok {
		writeError(ctx, fasthttp.StatusNotFound, fmt.Sprintf("no oauth flow for provider %q", providerType))
		return
	}

	var req struct {
		State      string `json:"state"`
		Code       string `json:"code"`
		ProviderID string `json:"provider_id"`
		Name       string `json:"name"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.State == "" || req.Code == "" || req.ProviderID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "state, code, and provider_id are required")
		return
	}
	if _, err := h.store.GetProvider(req.ProviderID); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusBadRequest, "unknown provider_id")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load provider")
		return
	}

	token, err := flow.Exchange(req.State, req.Code)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("oauth exchange failed: %v", err))
		return
	}

	name := req.Name
	if name == "" {
		name = providerType + " oauth"
	}
	conn := &store.Connection{
		ProviderID:   req.ProviderID,
		Name:         name,
		Kind:         "oauth",
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
	}
	if err := h.store.CreateConnection(conn); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "store oauth connection")
		return
	}
	writeData(ctx, fasthttp.StatusCreated, toConnectionDTO(conn))
}

// RefreshConnection handles POST /api/connections/{id}/refresh. It uses the
// stored refresh token to obtain a new access token.
func (h *Handlers) RefreshConnection(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}

	conn, err := h.store.GetConnection(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "connection not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load connection")
		return
	}
	if conn.RefreshToken == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "connection has no refresh token")
		return
	}

	provider, err := h.store.GetProvider(conn.ProviderID)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load provider")
		return
	}
	flow, ok := h.flows[provider.Type]
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("no oauth flow for provider %q", provider.Type))
		return
	}

	token, err := flow.Refresh(conn.RefreshToken)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("oauth refresh failed: %v", err))
		return
	}

	conn.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		conn.RefreshToken = token.RefreshToken
	}
	if token.ExpiresAt != 0 {
		conn.ExpiresAt = token.ExpiresAt
	}
	if err := h.store.UpdateConnection(conn); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "store refreshed tokens")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toConnectionDTO(conn))
}
