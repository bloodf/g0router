package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/auth"
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

	redirectURI := h.resolveRedirectURI(ctx, providerType)
	var authURL, state string
	var err error
	if redirectURI != "" {
		authURL, state, err = flow.StartWithRedirect(redirectURI)
	} else {
		authURL, state, err = flow.Start()
	}
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

	redirectURI := h.resolveRedirectURI(ctx, providerType)
	var token *auth.OAuthToken
	var err error
	if redirectURI != "" {
		token, err = flow.ExchangeWithRedirect(req.State, req.Code, redirectURI)
	} else {
		token, err = flow.Exchange(req.State, req.Code)
	}
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

// resolveRedirectURI returns the redirect URI for OAuth flows.
// Priority: explicit settings override, then request Origin header,
// then derived from request scheme+host. Empty means "use flow default".
func (h *Handlers) resolveRedirectURI(ctx *fasthttp.RequestCtx, providerType string) string {
	settings, err := h.store.GetSettings()
	if err == nil {
		if override, ok := settings["oauth_redirect_uri"]; ok && override != "" {
			return override
		}
	}

	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		scheme := string(ctx.Request.URI().Scheme())
		if scheme == "" {
			scheme = "http"
		}
		host := string(ctx.Request.Host())
		if host == "" {
			host = string(ctx.Request.URI().Host())
		}
		if host != "" {
			origin = scheme + "://" + host
		}
	}
	if origin != "" {
		return strings.TrimRight(origin, "/") + "/api/oauth/" + providerType + "/callback"
	}
	return ""
}
