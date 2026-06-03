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
	deepseekProviderID ProviderID = "deepseek"
	deepseekClientID              = "deepseek"
	deepseekTokenURL              = "https://api.deepseek.com/oauth/token"
)

// DeepSeekConfig configures the DeepSeek password OAuth flow.
type DeepSeekConfig struct {
	ClientID   string
	TokenURL   string
	HTTPClient *http.Client
}

// DeepSeekFlow implements DeepSeek's password-style OAuth exchange.
type DeepSeekFlow struct {
	client   *http.Client
	clientID string
	tokenURL string
}

// NewDeepSeekFlow returns a DeepSeek OAuth flow using defaults for zero config fields.
func NewDeepSeekFlow(config DeepSeekConfig) *DeepSeekFlow {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	clientID := config.ClientID
	if clientID == "" {
		clientID = deepseekClientID
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = deepseekTokenURL
	}

	return &DeepSeekFlow{
		client:   client,
		clientID: clientID,
		tokenURL: tokenURL,
	}
}

func (f *DeepSeekFlow) ProviderID() ProviderID {
	return deepseekProviderID
}

func (f *DeepSeekFlow) Start(ctx context.Context) (AuthSession, error) {
	return AuthSession{}, errors.New("deepseek oauth does not support start")
}

func (f *DeepSeekFlow) Exchange(ctx context.Context, session AuthSession, password string) (TokenResult, error) {
	if session.Provider != f.ProviderID() {
		return TokenResult{}, fmt.Errorf("deepseek exchange: provider mismatch: %s", session.Provider)
	}
	if session.SessionID == "" {
		return TokenResult{}, errors.New("deepseek exchange: username is required")
	}
	if password == "" {
		return TokenResult{}, errors.New("deepseek exchange: password is required")
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", f.clientID)
	form.Set("username", session.SessionID)
	form.Set("password", password)

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

func (f *DeepSeekFlow) Refresh(ctx context.Context, refreshToken string) (TokenResult, error) {
	return refreshTokenGrant(ctx, f.client, f.tokenURL, f.clientID, deepseekProviderID, refreshToken)
}

func (f *DeepSeekFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("deepseek oauth does not support poll")
}
