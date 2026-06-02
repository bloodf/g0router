package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/valyala/fasthttp"
)

type OAuthFlows map[oauth.ProviderID]oauth.Flow

type oauthExchangeRequest struct {
	Session oauth.AuthSession `json:"session"`
	Code    string            `json:"code"`
}

func OAuthStart(ctx *fasthttp.RequestCtx, flows OAuthFlows) {
	flow, ok := oauthFlowForPath(ctx, flows)
	if !ok {
		return
	}

	session, err := flow.Start(context.Background())
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("start oauth: %v", err))
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, session)
}

func OAuthPoll(ctx *fasthttp.RequestCtx, flows OAuthFlows) {
	flow, ok := oauthFlowForPath(ctx, flows)
	if !ok {
		return
	}

	sessionID := strings.TrimSpace(string(ctx.QueryArgs().Peek("session_id")))
	if sessionID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "session_id is required")
		return
	}

	result, err := flow.Poll(context.Background(), oauth.AuthSession{
		Provider:  flow.ProviderID(),
		SessionID: sessionID,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("poll oauth: %v", err))
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, result)
}

func OAuthCallback(ctx *fasthttp.RequestCtx, flows OAuthFlows) {
	if oauthErr := strings.TrimSpace(string(ctx.QueryArgs().Peek("error"))); oauthErr != "" {
		writeError(ctx, fasthttp.StatusBadRequest, "oauth callback: "+oauthErr)
		return
	}

	provider := oauth.ProviderID(strings.TrimSpace(string(ctx.QueryArgs().Peek("provider"))))
	if provider == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider is required")
		return
	}

	flow, ok := oauthFlow(ctx, flows, provider)
	if !ok {
		return
	}

	code := strings.TrimSpace(string(ctx.QueryArgs().Peek("code")))
	if code == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "code is required")
		return
	}

	sessionID := strings.TrimSpace(string(ctx.QueryArgs().Peek("session_id")))
	if sessionID == "" {
		sessionID = strings.TrimSpace(string(ctx.QueryArgs().Peek("state")))
	}
	if sessionID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "session_id is required")
		return
	}

	exchangeOAuth(ctx, flow, oauth.AuthSession{
		Provider:  provider,
		SessionID: sessionID,
	}, code)
}

func OAuthExchange(ctx *fasthttp.RequestCtx, flows OAuthFlows) {
	flow, ok := oauthFlowForPath(ctx, flows)
	if !ok {
		return
	}

	var req oauthExchangeRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Code == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "code is required")
		return
	}
	if req.Session.SessionID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "session.session_id is required")
		return
	}
	if req.Session.Provider == "" {
		req.Session.Provider = flow.ProviderID()
	}

	exchangeOAuth(ctx, flow, req.Session, req.Code)
}

func exchangeOAuth(ctx *fasthttp.RequestCtx, flow oauth.Flow, session oauth.AuthSession, code string) {
	token, err := flow.Exchange(context.Background(), session, code)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("exchange oauth: %v", err))
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, token)
}

func oauthFlowForPath(ctx *fasthttp.RequestCtx, flows OAuthFlows) (oauth.Flow, bool) {
	provider := oauthProviderFromPath(ctx)
	if provider == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider is required")
		return nil, false
	}

	return oauthFlow(ctx, flows, provider)
}

func oauthFlow(ctx *fasthttp.RequestCtx, flows OAuthFlows, provider oauth.ProviderID) (oauth.Flow, bool) {
	flow, ok := flows[provider]
	if !ok || flow == nil {
		writeError(ctx, fasthttp.StatusNotFound, "oauth provider not found")
		return nil, false
	}

	return flow, true
}

func oauthProviderFromPath(ctx *fasthttp.RequestCtx) oauth.ProviderID {
	parts := strings.Split(strings.Trim(string(ctx.Path()), "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "oauth" {
		return ""
	}
	return oauth.ProviderID(parts[2])
}
