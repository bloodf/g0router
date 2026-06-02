package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	anthropicProviderID  ProviderID = "anthropic"
	anthropicClientID               = "9d6bc642-e7b0-4445-8dab-bd2d0804e37c"
	anthropicAuthURL                = "https://console.anthropic.com/oauth/authorize"
	anthropicTokenURL               = "https://api.anthropic.com/v1/oauth/token"
	anthropicRedirectURI            = "http://localhost:54545/oauth/callback"
	anthropicScope                  = "org:create_api_key"
)

// AnthropicConfig configures the Anthropic PKCE OAuth flow.
type AnthropicConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// AnthropicFlow implements Anthropic's callback-based PKCE OAuth flow.
type AnthropicFlow struct {
	client      *http.Client
	clientID    string
	authURL     string
	tokenURL    string
	redirectURI string
	scopes      []string
}

// NewAnthropicFlow returns an Anthropic OAuth flow with the documented defaults.
func NewAnthropicFlow() *AnthropicFlow {
	return NewAnthropicFlowWithConfig(AnthropicConfig{})
}

// NewAnthropicFlowWithConfig returns an Anthropic OAuth flow with testable endpoints.
func NewAnthropicFlowWithConfig(config AnthropicConfig) *AnthropicFlow {
	clientID := config.ClientID
	if clientID == "" {
		clientID = anthropicClientID
	}
	authURL := config.AuthURL
	if authURL == "" {
		authURL = anthropicAuthURL
	}
	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = anthropicTokenURL
	}
	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = anthropicRedirectURI
	}
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{anthropicScope}
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &AnthropicFlow{
		client:      client,
		clientID:    clientID,
		authURL:     authURL,
		tokenURL:    tokenURL,
		redirectURI: redirectURI,
		scopes:      append([]string(nil), scopes...),
	}
}

func (f *AnthropicFlow) ProviderID() ProviderID {
	return anthropicProviderID
}

func (f *AnthropicFlow) Start(ctx context.Context) (AuthSession, error) {
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
	query.Set("code_challenge_method", "S256")
	query.Set("code_challenge", codeChallenge(verifier))
	authURL.RawQuery = query.Encode()

	return AuthSession{
		Provider:  f.ProviderID(),
		AuthURL:   authURL.String(),
		SessionID: state + "." + verifier,
	}, nil
}

func (f *AnthropicFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	if session.Provider != f.ProviderID() {
		return TokenResult{}, fmt.Errorf("anthropic exchange: provider mismatch: %s", session.Provider)
	}
	if code == "" {
		return TokenResult{}, fmt.Errorf("anthropic exchange: code is required")
	}

	_, verifier, err := parseAnthropicSessionID(session.SessionID)
	if err != nil {
		return TokenResult{}, fmt.Errorf("anthropic exchange: %w", err)
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

	var token anthropicTokenResponse
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

func (f *AnthropicFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, fmt.Errorf("anthropic oauth does not support poll")
}

type anthropicTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func randomURLToken(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func parseAnthropicSessionID(sessionID string) (string, string, error) {
	parts := strings.SplitN(sessionID, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid session id")
	}
	return parts[0], parts[1], nil
}

func splitScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	return strings.Fields(scope)
}
