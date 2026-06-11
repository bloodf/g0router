package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

const (
	oidcStateCookieName    = "oidc_state"
	oidcNonceCookieName    = "oidc_nonce"
	oidcVerifierCookieName = "oidc_code_verifier"
)

// oidcCookieOptions are the attributes shared by the three OIDC cookies.
func oidcCookieOptions(ctx *fasthttp.RequestCtx) (path string, maxAge int, secure bool, sameSite fasthttp.CookieSameSite) {
	return "/", auth.OIDCCookieMaxAge, shouldUseSecureCookie(ctx), fasthttp.CookieSameSiteLaxMode
}

// shouldUseSecureCookie reports whether the OIDC cookies should carry the
// Secure flag. It mirrors the ref's logic: secure when the request is TLS or
// when an upstream proxy reports https via X-Forwarded-Proto.
func shouldUseSecureCookie(ctx *fasthttp.RequestCtx) bool {
	if string(ctx.Request.Header.Peek("X-Forwarded-Proto")) == "https" {
		return true
	}
	return ctx.IsTLS()
}

// publicOrigin returns the external origin for building the OIDC redirect URI.
func publicOrigin(ctx *fasthttp.RequestCtx) string {
	proto := "http"
	if shouldUseSecureCookie(ctx) {
		proto = "https"
	}
	host := string(ctx.Host())
	if host == "" {
		host = "localhost"
	}
	return proto + "://" + host
}

// setOIDCCookie sets one OIDC cookie with the shared options.
func setOIDCCookie(ctx *fasthttp.RequestCtx, name, value string) {
	path, maxAge, secure, sameSite := oidcCookieOptions(ctx)
	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	cookie.SetKey(name)
	cookie.SetValue(value)
	cookie.SetPath(path)
	cookie.SetMaxAge(maxAge)
	cookie.SetHTTPOnly(true)
	cookie.SetSameSite(sameSite)
	if secure {
		cookie.SetSecure(true)
	}
	ctx.Response.Header.SetCookie(cookie)
}

// deleteOIDCCookie instructs the browser to remove an OIDC cookie.
func deleteOIDCCookie(ctx *fasthttp.RequestCtx, name string) {
	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetMaxAge(0)
	cookie.SetExpire(time.Unix(0, 0))
	cookie.SetHTTPOnly(true)
	cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	if shouldUseSecureCookie(ctx) {
		cookie.SetSecure(true)
	}
	ctx.Response.Header.SetCookie(cookie)
}

// deleteAllOIDCCookies removes the three OIDC cookies.
func deleteAllOIDCCookies(ctx *fasthttp.RequestCtx) {
	deleteOIDCCookie(ctx, oidcStateCookieName)
	deleteOIDCCookie(ctx, oidcNonceCookieName)
	deleteOIDCCookie(ctx, oidcVerifierCookieName)
}

// OIDCStart handles GET /api/auth/oidc/start.
func (h *Handlers) OIDCStart(ctx *fasthttp.RequestCtx) {
	settings, err := h.store.GetSettings()
	if err != nil {
		redirectToLogin(ctx, "oidc_load_settings")
		return
	}
	if !h.oidcConfigured(settings) {
		redirectToLogin(ctx, "oidc_not_configured")
		return
	}

	issuerURL := strings.TrimRight(settings["oidc_issuer_url"], "/")
	discovery, err := auth.FetchOIDCDiscovery(issuerURL, nil)
	if err != nil {
		redirectToLogin(ctx, "oidc_discovery_failed")
		return
	}

	state, err := auth.CreateOIDCState()
	if err != nil {
		redirectToLogin(ctx, "oidc_state_failed")
		return
	}
	nonce, err := auth.CreateOIDCNonce()
	if err != nil {
		redirectToLogin(ctx, "oidc_nonce_failed")
		return
	}
	pair, err := auth.CreateOIDCPKCEPair()
	if err != nil {
		redirectToLogin(ctx, "oidc_pkce_failed")
		return
	}

	redirectURI := publicOrigin(ctx) + "/api/auth/oidc/callback"
	authURL := auth.BuildOIDCAuthorizationURL(auth.OIDCAuthURLParams{
		AuthorizationEndpoint: discovery.AuthorizationEndpoint,
		ClientID:              strings.TrimSpace(settings["oidc_client_id"]),
		RedirectURI:           redirectURI,
		Scopes:                settings["oidc_scopes"],
		State:                 state,
		Nonce:                 nonce,
		CodeChallenge:         pair.Challenge,
	})

	setOIDCCookie(ctx, oidcStateCookieName, state)
	setOIDCCookie(ctx, oidcNonceCookieName, nonce)
	setOIDCCookie(ctx, oidcVerifierCookieName, pair.Verifier)

	ctx.Redirect(authURL, fasthttp.StatusFound)
}

// OIDCCallback handles GET /api/auth/oidc/callback.
func (h *Handlers) OIDCCallback(ctx *fasthttp.RequestCtx) {
	storedState := string(ctx.Request.Header.Cookie(oidcStateCookieName))
	storedNonce := string(ctx.Request.Header.Cookie(oidcNonceCookieName))
	codeVerifier := string(ctx.Request.Header.Cookie(oidcVerifierCookieName))

	returnedState := string(ctx.QueryArgs().Peek("state"))
	code := string(ctx.QueryArgs().Peek("code"))

	if err := auth.ValidateOIDCState(storedState, returnedState); err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusUnauthorized, "oidc state mismatch")
		return
	}

	settings, err := h.store.GetSettings()
	if err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusInternalServerError, "load settings")
		return
	}
	if !h.oidcConfigured(settings) {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusUnauthorized, "oidc not configured")
		return
	}

	issuerURL := strings.TrimRight(settings["oidc_issuer_url"], "/")
	discovery, err := auth.FetchOIDCDiscovery(issuerURL, nil)
	if err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusInternalServerError, "oidc discovery failed")
		return
	}

	tokenData, err := auth.ExchangeOIDCCode(auth.OIDCCodeExchangeParams{
		TokenEndpoint: discovery.TokenEndpoint,
		ClientID:      strings.TrimSpace(settings["oidc_client_id"]),
		ClientSecret:  strings.TrimSpace(settings["oidc_client_secret"]),
		Code:          code,
		RedirectURI:   publicOrigin(ctx) + "/api/auth/oidc/callback",
		CodeVerifier:  codeVerifier,
	}, nil)
	if err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusUnauthorized, "oidc token exchange failed")
		return
	}

	idToken, _ := tokenData["id_token"].(string)
	if idToken == "" {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusUnauthorized, "oidc id_token missing")
		return
	}

	if err := auth.VerifyOIDCNonce(idToken, storedNonce); err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusUnauthorized, "oidc nonce mismatch")
		return
	}

	claims, err := parseIDTokenPayload(idToken)
	if err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusUnauthorized, "oidc id_token invalid")
		return
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	name := pickOIDCName(claims)

	user, err := h.ensureOIDCUser(email, name, sub)
	if err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusInternalServerError, "oidc user setup failed")
		return
	}

	token, err := h.sessions.CreateOIDCSession(user.ID)
	if err != nil {
		deleteAllOIDCCookies(ctx)
		writeError(ctx, fasthttp.StatusInternalServerError, "oidc session failed")
		return
	}

	deleteAllOIDCCookies(ctx)

	sessionCookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(sessionCookie)
	sessionCookie.SetKey(sessionCookieName)
	sessionCookie.SetValue(token)
	sessionCookie.SetPath("/")
	sessionCookie.SetHTTPOnly(true)
	sessionCookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	sessionCookie.SetExpire(time.Now().Add(7 * 24 * time.Hour))
	ctx.Response.Header.SetCookie(sessionCookie)

	ctx.Redirect("/dashboard", fasthttp.StatusFound)
}

// OIDCTest handles POST /api/auth/oidc/test.
func (h *Handlers) OIDCTest(ctx *fasthttp.RequestCtx) {
	var body struct {
		IssuerURL    string `json:"issuerUrl"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		RedirectURI  string `json:"redirectUri"`
		Scopes       string `json:"scopes"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}

	settings, err := h.store.GetSettings()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load settings")
		return
	}

	issuerURL := strings.TrimSpace(body.IssuerURL)
	if issuerURL == "" {
		issuerURL = strings.TrimSpace(settings["oidc_issuer_url"])
	}
	clientID := strings.TrimSpace(body.ClientID)
	if clientID == "" {
		clientID = strings.TrimSpace(settings["oidc_client_id"])
	}
	clientSecret := body.ClientSecret
	if clientSecret == "" {
		clientSecret = settings["oidc_client_secret"]
	}
	scopes := strings.TrimSpace(body.Scopes)
	if scopes == "" {
		scopes = strings.TrimSpace(settings["oidc_scopes"])
	}
	if scopes == "" {
		scopes = "openid profile email"
	}
	redirectURI := strings.TrimSpace(body.RedirectURI)
	if redirectURI == "" {
		redirectURI = publicOrigin(ctx) + "/api/auth/oidc/callback"
	}

	if issuerURL == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "issuerUrl is required")
		return
	}
	if clientID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "clientId is required")
		return
	}

	discovery, err := auth.FetchOIDCDiscovery(issuerURL, nil)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "failed to load OIDC discovery document")
		return
	}

	probe, err := auth.ProbeOIDCClientSecret(discovery.TokenEndpoint, clientID, clientSecret, redirectURI, nil)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "oidc secret probe failed")
		return
	}

	if probe.Tested && probe.Valid != nil && !*probe.Valid {
		writeData(ctx, fasthttp.StatusOK, map[string]any{
			"ok":                  false,
			"discoveryOk":         true,
			"clientSecretTested":  true,
			"clientSecretValid":   false,
			"issuerUrl":           issuerURL,
			"clientId":            clientID,
			"scopes":              scopes,
			"redirectUri":         redirectURI,
			"authorizationEndpoint": discovery.AuthorizationEndpoint,
			"tokenEndpoint":       discovery.TokenEndpoint,
			"jwksUri":             discovery.JWKSURI,
			"error":               fmt.Sprintf("Discovery loaded, but the client secret is not valid: %s", probe.Message),
		})
		return
	}

	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"ok":                    true,
		"discoveryOk":           true,
		"clientSecretTested":    probe.Tested,
		"clientSecretValid":     probe.Valid,
		"issuerUrl":             issuerURL,
		"clientId":              clientID,
		"scopes":                scopes,
		"redirectUri":           redirectURI,
		"authorizationEndpoint": discovery.AuthorizationEndpoint,
		"tokenEndpoint":         discovery.TokenEndpoint,
		"jwksUri":               discovery.JWKSURI,
		"message":               probe.Message,
	})
}

// redirectToLogin redirects the browser to the login page with an error.
func redirectToLogin(ctx *fasthttp.RequestCtx, errorCode string) {
	ctx.Redirect("/login?error="+url.QueryEscape(errorCode), fasthttp.StatusFound)
}

// ensureOIDCUser finds or creates a dashboard user for the OIDC identity.
func (h *Handlers) ensureOIDCUser(email, name, sub string) (*store.User, error) {
	if sub == "" {
		return nil, fmt.Errorf("oidc id_token missing sub claim")
	}
	username := "oidc:" + sub
	user, err := h.store.GetUserByUsername(username)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("lookup oidc user: %w", err)
	}
	user, err = h.store.CreateUser(username, "")
	if err != nil {
		return nil, fmt.Errorf("create oidc user: %w", err)
	}
	_ = email
	_ = name
	return user, nil
}

// pickOIDCName selects a human-readable name from OIDC claims.
func pickOIDCName(claims map[string]any) string {
	for _, key := range []string{"preferred_username", "email", "name", "given_name"} {
		if v, ok := claims[key].(string); ok && v != "" {
			return v
		}
	}
	if v, ok := claims["sub"].(string); ok && v != "" {
		return v
	}
	return "OIDC user"
}

// parseIDTokenPayload extracts the payload claims from an ID token.
func parseIDTokenPayload(idToken string) (map[string]any, error) {
	return auth.ParseIDTokenPayload(idToken)
}
