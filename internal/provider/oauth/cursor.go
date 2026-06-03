package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	cursorProviderID  ProviderID = "cursor"
	cursorClientID               = "cursor"
	cursorAuthURL                = "https://auth.cursor.com/oauth/authorize"
	cursorTokenURL               = "https://auth.cursor.com/oauth/token"
	cursorRedirectURI            = "http://localhost:54545/oauth/cursor/callback"
)

// CursorConfig configures the Cursor PKCE OAuth flow.
type CursorConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// CursorFlow implements Cursor's callback-based PKCE OAuth flow.
type CursorFlow struct {
	client      *http.Client
	clientID    string
	authURL     string
	tokenURL    string
	redirectURI string
	scopes      []string
}

// NewCursorFlow returns a Cursor OAuth flow with the documented defaults.
func NewCursorFlow() *CursorFlow {
	return NewCursorFlowWithConfig(CursorConfig{})
}

// NewCursorFlowWithConfig returns a Cursor OAuth flow with testable endpoints.
func NewCursorFlowWithConfig(config CursorConfig) *CursorFlow {
	clientID := config.ClientID
	if clientID == "" {
		clientID = cursorClientID
	}
	authURL := config.AuthURL
	if authURL == "" {
		authURL = cursorAuthURL
	}
	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = cursorTokenURL
	}
	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = cursorRedirectURI
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &CursorFlow{
		client:      client,
		clientID:    clientID,
		authURL:     authURL,
		tokenURL:    tokenURL,
		redirectURI: redirectURI,
		scopes:      append([]string(nil), config.Scopes...),
	}
}

func (f *CursorFlow) ProviderID() ProviderID {
	return cursorProviderID
}

func (f *CursorFlow) Start(ctx context.Context) (AuthSession, error) {
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
		Provider:  f.ProviderID(),
		AuthURL:   authURL.String(),
		SessionID: state + "." + verifier,
	}, nil
}

func (f *CursorFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	if session.Provider != f.ProviderID() {
		return TokenResult{}, fmt.Errorf("cursor exchange: provider mismatch: %s", session.Provider)
	}
	if code == "" {
		return TokenResult{}, fmt.Errorf("cursor exchange: code is required")
	}

	_, verifier, err := parseCursorSessionID(session.SessionID)
	if err != nil {
		return TokenResult{}, fmt.Errorf("cursor exchange: %w", err)
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

	var token cursorTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return TokenResult{}, fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" {
		return TokenResult{}, fmt.Errorf("decode token response: access token is required")
	}

	result := TokenResult{
		Provider:     f.ProviderID(),
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

func (f *CursorFlow) Refresh(ctx context.Context, refreshToken string) (TokenResult, error) {
	return refreshTokenGrant(ctx, f.client, f.tokenURL, f.clientID, cursorProviderID, refreshToken)
}

func (f *CursorFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, fmt.Errorf("cursor oauth does not support poll")
}

type cursorTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func parseCursorSessionID(sessionID string) (string, string, error) {
	parts := strings.SplitN(sessionID, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid session id")
	}
	return parts[0], parts[1], nil
}
