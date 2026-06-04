package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type OAuthFlows map[oauth.ProviderID]oauth.Flow

type oauthStartRequest struct {
	AccountLabel string `json:"account_label"`
}

type oauthExchangeRequest struct {
	Session oauth.AuthSession `json:"session"`
	State   string            `json:"state"`
	Code    string            `json:"code"`
}

type oauthPollResponse struct {
	Status     oauth.PollStatus         `json:"status"`
	Connection *oauthConnectionResponse `json:"connection,omitempty"`
}

type oauthConnectionResponse struct {
	ID        string   `json:"id"`
	Provider  string   `json:"provider"`
	Name      string   `json:"name"`
	AuthType  string   `json:"auth_type"`
	ExpiresAt *int64   `json:"expires_at,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
}

func OAuthStart(ctx *fasthttp.RequestCtx, s *store.Store, flows OAuthFlows) {
	flow, runtimeProvider, ok := oauthFlowForPath(ctx, flows)
	if !ok {
		return
	}
	req, ok := decodeOAuthStartRequest(ctx)
	if !ok {
		return
	}

	session, err := flow.Start(requestContext(ctx))
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("start oauth: %v", err))
		return
	}
	session.Provider = runtimeProvider
	if err := createOAuthSession(s, &session, req.AccountLabel); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("store oauth session: %v", err))
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, session)
}

func OAuthPoll(ctx *fasthttp.RequestCtx, s *store.Store, flows OAuthFlows) {
	flow, runtimeProvider, ok := oauthFlowForPath(ctx, flows)
	if !ok {
		return
	}

	sessionID := strings.TrimSpace(string(ctx.QueryArgs().Peek("session_id")))
	if sessionID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "session_id is required")
		return
	}
	authSession := oauth.AuthSession{
		Provider:  flow.ProviderID(),
		SessionID: sessionID,
	}
	storedSession, err := getOAuthSession(s, sessionID)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("oauth session: %v", err))
		return
	}
	if storedSession != nil {
		if oauth.CanonicalFlowProviderID(oauth.ProviderID(storedSession.Provider)) != flow.ProviderID() {
			writeError(ctx, fasthttp.StatusBadRequest, "oauth session provider mismatch")
			return
		}
		authSession.SessionID = storedSession.State
		if storedSession.CodeVerifier != "" {
			authSession.SessionID = storedSession.State + "." + storedSession.CodeVerifier
		}
	}

	result, err := flow.Poll(requestContext(ctx), authSession)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("poll oauth: %v", err))
		return
	}

	response := oauthPollResponse{Status: result.Status}
	if result.Token != nil {
		accountLabel := ""
		if storedSession != nil {
			consumed, err := consumeOAuthSession(s, sessionID)
			if err != nil {
				writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("oauth session: %v", err))
				return
			}
			accountLabel = consumed.AccountLabel
		}
		connection, err := persistOAuthConnection(s, *result.Token, accountLabel, runtimeProvider.String())
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("persist oauth connection: %v", err))
			return
		}
		response.Connection = connection
	}
	writeJSON(ctx, fasthttp.StatusOK, response)
}

func OAuthCallback(ctx *fasthttp.RequestCtx, s *store.Store, flows OAuthFlows) {
	if oauthErr := strings.TrimSpace(string(ctx.QueryArgs().Peek("error"))); oauthErr != "" {
		writeError(ctx, fasthttp.StatusBadRequest, "oauth callback: "+oauthErr)
		return
	}

	code := strings.TrimSpace(string(ctx.QueryArgs().Peek("code")))
	if code == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "code is required")
		return
	}
	state := strings.TrimSpace(string(ctx.QueryArgs().Peek("state")))
	if state == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "state is required")
		return
	}

	exchangeStoredOAuth(ctx, s, flows, state, code)
}

func OAuthExchange(ctx *fasthttp.RequestCtx, s *store.Store, flows OAuthFlows) {
	flow, _, ok := oauthFlowForPath(ctx, flows)
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
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = strings.TrimSpace(req.Session.SessionID)
	}
	if state == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "state is required")
		return
	}

	session, err := consumeOAuthSession(s, state)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("oauth session: %v", err))
		return
	}
	if oauth.CanonicalFlowProviderID(oauth.ProviderID(session.Provider)) != flow.ProviderID() {
		writeError(ctx, fasthttp.StatusBadRequest, "oauth session provider mismatch")
		return
	}
	exchangeOAuth(ctx, s, flow, session, req.Code)
}

func exchangeStoredOAuth(ctx *fasthttp.RequestCtx, s *store.Store, flows OAuthFlows, state, code string) {
	session, err := consumeOAuthSession(s, state)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("oauth session: %v", err))
		return
	}
	flow, ok := oauthFlow(ctx, flows, oauth.ProviderID(session.Provider))
	if !ok {
		return
	}
	exchangeOAuth(ctx, s, flow, session, code)
}

func exchangeOAuth(ctx *fasthttp.RequestCtx, s *store.Store, flow oauth.Flow, session *store.OAuthSession, code string) {
	authSession := oauth.AuthSession{
		Provider:  flow.ProviderID(),
		SessionID: session.State,
	}
	if session.CodeVerifier != "" {
		authSession.SessionID = session.State + "." + session.CodeVerifier
	}

	token, err := flow.Exchange(requestContext(ctx), authSession, code)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadGateway, "oauth exchange failed")
		return
	}

	connection, err := persistOAuthConnection(s, token, session.AccountLabel, session.Provider)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("persist oauth connection: %v", err))
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, connection)
}

func oauthFlowForPath(ctx *fasthttp.RequestCtx, flows OAuthFlows) (oauth.Flow, oauth.ProviderID, bool) {
	provider := oauthProviderFromPath(ctx)
	if provider == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider is required")
		return nil, "", false
	}

	flow, ok := oauthFlow(ctx, flows, provider)
	return flow, provider, ok
}

func oauthFlow(ctx *fasthttp.RequestCtx, flows OAuthFlows, provider oauth.ProviderID) (oauth.Flow, bool) {
	canonical := oauth.CanonicalFlowProviderID(provider)
	flow, ok := flows[canonical]
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
	return oauth.ProviderID(strings.ToLower(strings.TrimSpace(parts[2])))
}

func decodeOAuthStartRequest(ctx *fasthttp.RequestCtx) (oauthStartRequest, bool) {
	if len(ctx.PostBody()) == 0 {
		return oauthStartRequest{}, true
	}
	var req oauthStartRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return oauthStartRequest{}, false
	}
	return req, true
}

func createOAuthSession(s *store.Store, session *oauth.AuthSession, accountLabel string) error {
	if s == nil || session.SessionID == "" {
		return nil
	}
	state, verifier := splitStoredOAuthSession(session.SessionID)
	expiresAt := time.Now().Add(10 * time.Minute)
	if session.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(session.ExpiresIn) * time.Second)
	}
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:        state,
		Provider:     string(session.Provider),
		CodeVerifier: verifier,
		RedirectURI:  redirectURIFromAuthURL(session.AuthURL),
		AccountLabel: strings.TrimSpace(accountLabel),
		ExpiresAt:    expiresAt,
	}); err != nil {
		return err
	}
	session.SessionID = state
	return nil
}

func consumeOAuthSession(s *store.Store, state string) (*store.OAuthSession, error) {
	if s == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	return s.ConsumeOAuthSession(state)
}

func getOAuthSession(s *store.Store, state string) (*store.OAuthSession, error) {
	if s == nil {
		return nil, nil
	}
	session, err := s.GetOAuthSession(state)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return session, nil
}

func splitStoredOAuthSession(sessionID string) (string, string) {
	state, verifier, ok := strings.Cut(sessionID, ".")
	if ok && state != "" && verifier != "" {
		return state, verifier
	}
	return sessionID, ""
}

func redirectURIFromAuthURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("redirect_uri")
}

func persistOAuthConnection(s *store.Store, token oauth.TokenResult, accountLabel, runtimeProvider string) (*oauthConnectionResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("store unavailable")
	}
	conn := provider.ConnectionFromOAuthTokenForProvider(token, accountLabel, runtimeProvider)
	if err := s.CreateConnection(conn); err != nil {
		return nil, err
	}
	return &oauthConnectionResponse{
		ID:        conn.ID,
		Provider:  conn.Provider,
		Name:      conn.Name,
		AuthType:  string(conn.AuthType),
		ExpiresAt: conn.ExpiresAt,
		Scopes:    append([]string(nil), token.Scopes...),
	}, nil
}
