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
	geminiProviderID  ProviderID = "gemini"
	geminiClientID               = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	googleAuthURL                = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL               = "https://oauth2.googleapis.com/token"
	googleRedirectURI            = "http://localhost:54545/oauth/callback"
)

// GeminiConfig configures the Gemini CLI Google PKCE OAuth flow.
type GeminiConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// GeminiFlow implements Gemini CLI's Google callback-based PKCE OAuth flow.
type GeminiFlow struct {
	google *googleOAuthFlow
}

// NewGeminiFlow returns a Gemini OAuth flow using defaults for zero config fields.
func NewGeminiFlow(config GeminiConfig) *GeminiFlow {
	return &GeminiFlow{
		google: newGoogleOAuthFlow(googleOAuthConfig{
			Provider:    geminiProviderID,
			ClientID:    config.ClientID,
			DefaultID:   geminiClientID,
			AuthURL:     config.AuthURL,
			TokenURL:    config.TokenURL,
			RedirectURI: config.RedirectURI,
			Scopes:      config.Scopes,
			HTTPClient:  config.HTTPClient,
		}),
	}
}

func (f *GeminiFlow) ProviderID() ProviderID {
	return geminiProviderID
}

func (f *GeminiFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.google.start(ctx)
}

func (f *GeminiFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.google.exchange(ctx, session, code)
}

func (f *GeminiFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("gemini oauth does not support poll")
}

type googleOAuthConfig struct {
	Provider    ProviderID
	ClientID    string
	DefaultID   string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

type googleOAuthFlow struct {
	provider    ProviderID
	client      *http.Client
	clientID    string
	authURL     string
	tokenURL    string
	redirectURI string
	scopes      []string
}

func newGoogleOAuthFlow(config googleOAuthConfig) *googleOAuthFlow {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	clientID := config.ClientID
	if clientID == "" {
		clientID = config.DefaultID
	}

	authURL := config.AuthURL
	if authURL == "" {
		authURL = googleAuthURL
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = googleTokenURL
	}

	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = googleRedirectURI
	}

	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	}

	return &googleOAuthFlow{
		provider:    config.Provider,
		client:      client,
		clientID:    clientID,
		authURL:     authURL,
		tokenURL:    tokenURL,
		redirectURI: redirectURI,
		scopes:      append([]string(nil), scopes...),
	}
}

func (f *googleOAuthFlow) start(ctx context.Context) (AuthSession, error) {
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
	query.Set("scope", strings.Join(f.scopes, " "))
	query.Set("state", state)
	query.Set("access_type", "offline")
	query.Set("prompt", "consent")
	query.Set("code_challenge_method", "S256")
	query.Set("code_challenge", codeChallenge(verifier))
	authURL.RawQuery = query.Encode()

	return AuthSession{
		Provider:  f.provider,
		AuthURL:   authURL.String(),
		SessionID: state + "." + verifier,
	}, nil
}

func (f *googleOAuthFlow) exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	if session.Provider != f.provider {
		return TokenResult{}, fmt.Errorf("%s exchange: provider mismatch: %s", f.provider, session.Provider)
	}
	if code == "" {
		return TokenResult{}, fmt.Errorf("%s exchange: code is required", f.provider)
	}

	_, verifier, err := parseGoogleSessionID(session.SessionID)
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

	var token googleTokenResponse
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

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func parseGoogleSessionID(sessionID string) (string, string, error) {
	parts := strings.SplitN(sessionID, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("invalid session id")
	}
	return parts[0], parts[1], nil
}
