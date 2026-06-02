package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	xaiProviderID  ProviderID = "xai"
	xaiClientID               = "xai"
	xaiAuthURL                = "https://accounts.x.ai/oauth/authorize"
	xaiTokenURL               = "https://accounts.x.ai/oauth/token"
	xaiRedirectURI            = "http://localhost:54545/oauth/callback"
)

// XAIConfig configures the xAI callback-based PKCE OAuth flow.
type XAIConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// XAIFlow implements xAI callback-based PKCE OAuth.
type XAIFlow struct {
	oauth *callbackOAuthFlow
}

// NewXAIFlow returns an xAI OAuth flow using defaults for zero config fields.
func NewXAIFlow(config XAIConfig) *XAIFlow {
	authURL := config.AuthURL
	if authURL == "" {
		authURL = xaiAuthURL
	}
	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = xaiTokenURL
	}
	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = xaiRedirectURI
	}
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	return &XAIFlow{
		oauth: newCallbackOAuthFlow(callbackOAuthConfig{
			Provider:    xaiProviderID,
			ClientID:    config.ClientID,
			DefaultID:   xaiClientID,
			AuthURL:     authURL,
			TokenURL:    tokenURL,
			RedirectURI: redirectURI,
			Scopes:      scopes,
			HTTPClient:  config.HTTPClient,
		}),
	}
}

func (f *XAIFlow) ProviderID() ProviderID {
	return xaiProviderID
}

func (f *XAIFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.oauth.start(ctx)
}

func (f *XAIFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.oauth.exchange(ctx, session, code)
}

func (f *XAIFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("xai oauth does not support poll")
}

type callbackOAuthConfig struct {
	Provider    ProviderID
	ClientID    string
	DefaultID   string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

type callbackOAuthFlow struct {
	provider    ProviderID
	client      *http.Client
	clientID    string
	authURL     string
	tokenURL    string
	redirectURI string
	scopes      []string
}

func newCallbackOAuthFlow(config callbackOAuthConfig) *callbackOAuthFlow {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	clientID := config.ClientID
	if clientID == "" {
		clientID = config.DefaultID
	}

	return &callbackOAuthFlow{
		provider:    config.Provider,
		client:      client,
		clientID:    clientID,
		authURL:     config.AuthURL,
		tokenURL:    config.TokenURL,
		redirectURI: config.RedirectURI,
		scopes:      append([]string(nil), config.Scopes...),
	}
}

func (f *callbackOAuthFlow) start(ctx context.Context) (AuthSession, error) {
	verifier, err := randomURLToken(32)
	if err != nil {
		return AuthSession{}, fmt.Errorf("generate code verifier: %w", err)
	}
	state, err := randomURLToken(24)
	if err != nil {
		return AuthSession{}, fmt.Errorf("generate state: %w", err)
	}

	authURL, err := url.Parse(f.authURL)
	if err != nil {
		return AuthSession{}, fmt.Errorf("parse auth url: %w", err)
	}
	query := authURL.Query()
	query.Set("client_id", f.clientID)
	query.Set("redirect_uri", f.redirectURI)
	query.Set("response_type", "code")
	if len(f.scopes) > 0 {
		query.Set("scope", strings.Join(f.scopes, " "))
	}
	query.Set("state", state)
	query.Set("code_challenge_method", "S256")
	query.Set("code_challenge", codeChallenge(verifier))
	authURL.RawQuery = query.Encode()

	return AuthSession{
		Provider:  f.provider,
		AuthURL:   authURL.String(),
		SessionID: state + "." + verifier,
	}, nil
}

func (f *callbackOAuthFlow) exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	if session.Provider != f.provider {
		return TokenResult{}, fmt.Errorf("%s exchange: provider mismatch: %s", f.provider, session.Provider)
	}
	if code == "" {
		return TokenResult{}, fmt.Errorf("%s exchange: code is required", f.provider)
	}

	_, verifier, err := parseCallbackSessionID(session.SessionID)
	if err != nil {
		return TokenResult{}, fmt.Errorf("%s exchange: %w", f.provider, err)
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", f.clientID)
	form.Set("redirect_uri", f.redirectURI)
	form.Set("code", code)
	form.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResult{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return TokenResult{}, fmt.Errorf("exchange token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if readErr != nil {
			return TokenResult{}, fmt.Errorf("exchange token: status %d: read body: %w", resp.StatusCode, readErr)
		}
		return TokenResult{}, fmt.Errorf("exchange token: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return TokenResult{}, fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" {
		return TokenResult{}, errors.New("decode token response: access token is required")
	}

	result := TokenResult{
		Provider:     f.provider,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Scopes:       splitScopes(token.Scope),
	}
	if token.ExpiresIn > 0 {
		result.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return result, nil
}

func parseCallbackSessionID(sessionID string) (string, string, error) {
	parts := strings.SplitN(sessionID, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("invalid session id")
	}
	return parts[0], parts[1], nil
}
