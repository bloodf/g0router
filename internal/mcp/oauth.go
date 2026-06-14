package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
)

// mcpOAuthFlowTTL bounds how long an MCP authorization may take.
const mcpOAuthFlowTTL = 10 * time.Minute

// Engine drives the MCP-server OAuth authorization-code-with-PKCE flow, reusing
// the in-tree PKCE primitives (auth.GeneratePKCE) and the SHIPPED mcpoauth store.
// It discovers endpoints per the MCP authorization spec (RFC 9728
// protected-resource-metadata + RFC 8414 authorization-server-metadata) over an
// injectable *http.Client (nil → default). Tokens are kept *_enc at rest by the
// store and are MASKED in every returned account.
type Engine struct {
	store  *store.Store
	client *http.Client
}

// NewEngine builds an Engine. A nil client falls back to the package default.
func NewEngine(st *store.Store, client *http.Client) *Engine {
	if client == nil {
		client = defaultHTTPClient()
	}
	return &Engine{store: st, client: client}
}

// StartResult is returned by Start: the authorize URL the user visits + the CSRF
// state correlating the eventual callback.
type StartResult struct {
	AuthURL string
	State   string
}

// Start discovers the server's OAuth endpoints, persists a PKCE flow (verifier
// *_enc), and returns the authorize URL on the DISCOVERED authorization endpoint
// with an S256 challenge.
func (e *Engine) Start(ctx context.Context, serverURL, instanceID, redirectURI string) (*StartResult, error) {
	authzEndpoint, _, err := e.discover(ctx, serverURL)
	if err != nil {
		return nil, err
	}

	verifier, challenge, err := auth.GeneratePKCE()
	if err != nil {
		return nil, err
	}
	state, _, err := auth.GeneratePKCE() // reuse the URL-safe random for state
	if err != nil {
		return nil, err
	}

	if err := e.store.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
		State:       state,
		InstanceID:  instanceID,
		ServerURL:   serverURL,
		Verifier:    verifier,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(mcpOAuthFlowTTL).Unix(),
	}); err != nil {
		return nil, fmt.Errorf("persist mcp oauth flow: %w", err)
	}

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("resource", serverURL)
	return &StartResult{AuthURL: authzEndpoint + "?" + q.Encode(), State: state}, nil
}

// Complete consumes the persisted flow, exchanges the code for tokens at the
// discovered token endpoint, and upserts a connected account (tokens *_enc). The
// returned account has its token fields MASKED.
func (e *Engine) Complete(ctx context.Context, serverURL, state, code, redirectURI string) (*store.MCPOAuthAccount, error) {
	flow, err := e.store.ConsumeMCPOAuthFlow(state)
	if err != nil {
		return nil, fmt.Errorf("consume mcp oauth flow: %w", err)
	}

	_, tokenEndpoint, err := e.discover(ctx, serverURL)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", flow.Verifier)
	form.Set("resource", serverURL)

	tok, err := e.requestToken(ctx, tokenEndpoint, form)
	if err != nil {
		return nil, err
	}

	saved, err := e.store.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   flow.InstanceID,
		ServerURL:    serverURL,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    tok.ExpiresAt,
		Scope:        tok.Scope,
		Status:       "connected",
	})
	if err != nil {
		return nil, err
	}
	return maskAccount(saved), nil
}

// Refresh trades the account's refresh token for a new access token at
// tokenEndpoint and re-upserts the rotated tokens. The returned account is MASKED.
func (e *Engine) Refresh(ctx context.Context, account *store.MCPOAuthAccount, tokenEndpoint string) (*store.MCPOAuthAccount, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", account.RefreshToken)

	tok, err := e.requestToken(ctx, tokenEndpoint, form)
	if err != nil {
		return nil, err
	}
	refresh := tok.RefreshToken
	if refresh == "" {
		refresh = account.RefreshToken // some servers omit a rotated refresh token
	}

	saved, err := e.store.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		ID:           account.ID,
		InstanceID:   account.InstanceID,
		ServerURL:    account.ServerURL,
		AccessToken:  tok.AccessToken,
		RefreshToken: refresh,
		ExpiresAt:    tok.ExpiresAt,
		Scope:        account.Scope,
		Status:       "connected",
	})
	if err != nil {
		return nil, err
	}
	return maskAccount(saved), nil
}

// discover fetches the protected-resource-metadata then the first
// authorization-server-metadata, returning the authorize + token endpoints.
func (e *Engine) discover(ctx context.Context, serverURL string) (authzEndpoint, tokenEndpoint string, err error) {
	prmBody, err := e.getJSON(ctx, wellKnown(serverURL, "oauth-protected-resource"))
	if err != nil {
		return "", "", fmt.Errorf("fetch protected-resource-metadata: %w", err)
	}
	servers, err := parseProtectedResourceMetadata(prmBody)
	if err != nil {
		return "", "", err
	}
	if len(servers) == 0 {
		return "", "", fmt.Errorf("no authorization_servers in protected-resource-metadata")
	}

	asmBody, err := e.getJSON(ctx, wellKnown(servers[0], "oauth-authorization-server"))
	if err != nil {
		return "", "", fmt.Errorf("fetch authorization-server-metadata: %w", err)
	}
	authzEndpoint, tokenEndpoint, err = parseAuthServerMetadata(asmBody)
	if err != nil {
		return "", "", err
	}
	if authzEndpoint == "" || tokenEndpoint == "" {
		return "", "", fmt.Errorf("authorization-server-metadata missing endpoints")
	}
	return authzEndpoint, tokenEndpoint, nil
}

// wellKnown builds a RFC 8615 well-known URL from a base URL and suffix.
func wellKnown(base, suffix string) string {
	u, err := url.Parse(base)
	if err != nil {
		return strings.TrimRight(base, "/") + "/.well-known/" + suffix
	}
	u.Path = "/.well-known/" + suffix
	u.RawQuery = ""
	return u.String()
}

func (e *Engine) getJSON(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("metadata endpoint returned %d", resp.StatusCode)
	}
	return body, nil
}

// tokenResponse mirrors the OAuth token endpoint reply.
type tokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
	Scope        string
}

func (e *Engine) requestToken(ctx context.Context, tokenEndpoint string, form url.Values) (*tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	out := &tokenResponse{AccessToken: parsed.AccessToken, RefreshToken: parsed.RefreshToken, Scope: parsed.Scope}
	if parsed.ExpiresIn > 0 {
		out.ExpiresAt = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second).Unix()
	}
	return out, nil
}

// parseProtectedResourceMetadata extracts authorization_servers from RFC 9728
// protected-resource-metadata. PURE.
func parseProtectedResourceMetadata(body []byte) ([]string, error) {
	var prm struct {
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.Unmarshal(body, &prm); err != nil {
		return nil, fmt.Errorf("decode protected-resource-metadata: %w", err)
	}
	return prm.AuthorizationServers, nil
}

// parseAuthServerMetadata extracts the authorize + token endpoints from RFC 8414
// authorization-server-metadata. PURE.
func parseAuthServerMetadata(body []byte) (authzEndpoint, tokenEndpoint string, err error) {
	var asm struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
	}
	if err := json.Unmarshal(body, &asm); err != nil {
		return "", "", fmt.Errorf("decode authorization-server-metadata: %w", err)
	}
	return asm.AuthorizationEndpoint, asm.TokenEndpoint, nil
}

// needsRefresh reports whether a token expiring at expiresAt (unix seconds) should
// be refreshed at now, given a lead window. A zero expiresAt (unknown) never forces
// a refresh. PURE.
func needsRefresh(expiresAt int64, now time.Time, lead time.Duration) bool {
	if expiresAt == 0 {
		return false
	}
	return time.Unix(expiresAt, 0).Before(now.Add(lead))
}

// maskAccount returns a copy of the account with token fields cleared, so an
// engine return value can never echo a cleartext secret.
func maskAccount(a *store.MCPOAuthAccount) *store.MCPOAuthAccount {
	if a == nil {
		return nil
	}
	masked := *a
	masked.AccessToken = ""
	masked.RefreshToken = ""
	return &masked
}
