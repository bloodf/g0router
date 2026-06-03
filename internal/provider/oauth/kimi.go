package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	kimiProviderID    ProviderID = "kimi"
	kimiClientID                 = "kimi"
	kimiDeviceCodeURL            = "https://kimi.moonshot.cn/oauth/device/code"
	kimiTokenURL                 = "https://kimi.moonshot.cn/oauth/token"
)

// KimiFlowConfig configures the Kimi device-code OAuth flow.
type KimiFlowConfig struct {
	ClientID      string
	DeviceCodeURL string
	TokenURL      string
	HTTPClient    *http.Client
}

// KimiFlow implements Kimi's device-code OAuth flow.
type KimiFlow struct {
	client        *http.Client
	clientID      string
	deviceCodeURL string
	tokenURL      string
}

// NewKimiFlow returns a Kimi OAuth flow using defaults for zero config fields.
func NewKimiFlow(config KimiFlowConfig) *KimiFlow {
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	clientID := config.ClientID
	if clientID == "" {
		clientID = kimiClientID
	}

	deviceCodeURL := config.DeviceCodeURL
	if deviceCodeURL == "" {
		deviceCodeURL = kimiDeviceCodeURL
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = kimiTokenURL
	}

	return &KimiFlow{
		client:        client,
		clientID:      clientID,
		deviceCodeURL: deviceCodeURL,
		tokenURL:      tokenURL,
	}
}

func (f *KimiFlow) ProviderID() ProviderID {
	return kimiProviderID
}

func (f *KimiFlow) Start(ctx context.Context) (AuthSession, error) {
	var response deviceCodeResponse
	if err := f.postForm(ctx, f.deviceCodeURL, url.Values{
		"client_id": {f.clientID},
	}, &response); err != nil {
		return AuthSession{}, fmt.Errorf("start kimi device flow: %w", err)
	}

	authURL := response.VerificationURIComplete
	if authURL == "" {
		authURL = response.VerificationURI
	}

	return AuthSession{
		Provider:     kimiProviderID,
		AuthURL:      authURL,
		SessionID:    response.DeviceCode,
		UserCode:     response.UserCode,
		Verification: response.VerificationURI,
		ExpiresIn:    response.ExpiresIn,
		PollInterval: response.Interval,
	}, nil
}

func (f *KimiFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return TokenResult{}, errors.New("kimi oauth does not support callback exchange")
}

func (f *KimiFlow) Refresh(ctx context.Context, refreshToken string) (TokenResult, error) {
	return refreshTokenGrant(ctx, f.client, f.tokenURL, f.clientID, kimiProviderID, refreshToken)
}

func (f *KimiFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
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
		return PollResult{}, fmt.Errorf("poll kimi device flow: %w", err)
	}

	if response.AccessToken == "" {
		return PollResult{}, errors.New("poll kimi device flow: missing access token")
	}

	token := TokenResult{
		Provider:     kimiProviderID,
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

func (f *KimiFlow) postForm(ctx context.Context, endpoint string, form url.Values, target any) error {
	codex := CodexFlow{client: f.client}
	return codex.postForm(ctx, endpoint, form, target)
}
