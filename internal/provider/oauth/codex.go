package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	codexProviderID     ProviderID = "codex"
	codexClientID                  = "DQ1Ij3iIOC1S0aQCBk5KFj9m4gQZLrIf"
	codexDeviceCodeURL             = "https://auth.openai.com/oauth/device/code"
	codexTokenURL                  = "https://auth.openai.com/oauth/token"
	deviceCodeGrantType            = "urn:ietf:params:oauth:grant-type:device_code"
)

// CodexFlowConfig configures the OpenAI Codex device-code OAuth flow.
type CodexFlowConfig struct {
	ClientID      string
	DeviceCodeURL string
	TokenURL      string
	HTTPClient    *http.Client
}

// CodexFlow implements OpenAI Codex device-code OAuth.
type CodexFlow struct {
	client        *http.Client
	clientID      string
	deviceCodeURL string
	tokenURL      string
}

// NewCodexFlow returns a Codex OAuth flow using defaults for zero config fields.
func NewCodexFlow(config CodexFlowConfig) *CodexFlow {
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	clientID := config.ClientID
	if clientID == "" {
		clientID = codexClientID
	}

	deviceCodeURL := config.DeviceCodeURL
	if deviceCodeURL == "" {
		deviceCodeURL = codexDeviceCodeURL
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = codexTokenURL
	}

	return &CodexFlow{
		client:        client,
		clientID:      clientID,
		deviceCodeURL: deviceCodeURL,
		tokenURL:      tokenURL,
	}
}

func (f *CodexFlow) ProviderID() ProviderID {
	return codexProviderID
}

func (f *CodexFlow) Start(ctx context.Context) (AuthSession, error) {
	var response deviceCodeResponse
	if err := f.postForm(ctx, f.deviceCodeURL, url.Values{
		"client_id": {f.clientID},
	}, &response); err != nil {
		return AuthSession{}, fmt.Errorf("start codex device flow: %w", err)
	}

	return AuthSession{
		Provider:     codexProviderID,
		AuthURL:      response.VerificationURIComplete,
		SessionID:    response.DeviceCode,
		UserCode:     response.UserCode,
		Verification: response.VerificationURI,
		ExpiresIn:    response.ExpiresIn,
		PollInterval: response.Interval,
	}, nil
}

func (f *CodexFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return TokenResult{}, errors.New("codex oauth does not support callback exchange")
}

func (f *CodexFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	var response tokenResponse
	err := f.postForm(ctx, f.tokenURL, url.Values{
		"client_id":   {f.clientID},
		"device_code": {session.SessionID},
		"grant_type":  {deviceCodeGrantType},
	}, &response)
	if err != nil {
		var oauthErr oauthError
		if errors.As(err, &oauthErr) {
			return PollResult{Status: oauthErr.pollStatus()}, nil
		}
		return PollResult{}, fmt.Errorf("poll codex device flow: %w", err)
	}

	if response.AccessToken == "" {
		return PollResult{}, errors.New("poll codex device flow: missing access token")
	}

	token := TokenResult{
		Provider:     codexProviderID,
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		TokenType:    response.TokenType,
		Scopes:       strings.Fields(response.Scope),
	}
	if response.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)
	}

	return PollResult{
		Status: PollStatusComplete,
		Token:  &token,
	}, nil
}

func (f *CodexFlow) postForm(ctx context.Context, endpoint string, form url.Values, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("post form: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var oauthErr oauthError
		if err := json.NewDecoder(resp.Body).Decode(&oauthErr); err != nil {
			return fmt.Errorf("decode error response: %w", err)
		}
		return oauthErr
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type oauthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e oauthError) Error() string {
	if e.Description != "" {
		return e.Code + ": " + e.Description
	}
	return e.Code
}

func (e oauthError) pollStatus() PollStatus {
	switch e.Code {
	case "authorization_pending":
		return PollStatusPending
	case "slow_down":
		return PollStatusSlowDown
	case "expired_token":
		return PollStatusExpired
	case "access_denied":
		return PollStatusDenied
	default:
		return PollStatusPending
	}
}
