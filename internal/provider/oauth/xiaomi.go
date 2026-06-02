package oauth

import (
	"context"
	"errors"
	"net/http"
)

const (
	xiaomiProviderID   ProviderID = "xiaomi"
	xiaomiClientID                = "xiaomi"
	xiaomiAuthURL                 = "https://account.xiaomi.com/oauth2/authorize"
	xiaomiTokenURL                = "https://account.xiaomi.com/oauth2/token"
	xiaomiRedirectURI             = "http://localhost:54545/oauth/xiaomi/callback"
	xiaomiOpenIDScope             = "openid"
	xiaomiProfileScope            = "profile"
)

// XiaomiConfig configures the Xiaomi OAuth2 callback flow.
type XiaomiConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// XiaomiFlow implements Xiaomi's callback-based OAuth2 flow.
type XiaomiFlow struct {
	oauth *googleOAuthFlow
}

// NewXiaomiFlow returns a Xiaomi OAuth flow using defaults for zero config fields.
func NewXiaomiFlow(config XiaomiConfig) *XiaomiFlow {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{xiaomiOpenIDScope, xiaomiProfileScope}
	}

	return &XiaomiFlow{
		oauth: newGoogleOAuthFlow(googleOAuthConfig{
			Provider:    xiaomiProviderID,
			ClientID:    config.ClientID,
			DefaultID:   xiaomiClientID,
			AuthURL:     defaultString(config.AuthURL, xiaomiAuthURL),
			TokenURL:    defaultString(config.TokenURL, xiaomiTokenURL),
			RedirectURI: defaultString(config.RedirectURI, xiaomiRedirectURI),
			Scopes:      scopes,
			HTTPClient:  config.HTTPClient,
		}),
	}
}

func (f *XiaomiFlow) ProviderID() ProviderID {
	return xiaomiProviderID
}

func (f *XiaomiFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.oauth.start(ctx)
}

func (f *XiaomiFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.oauth.exchange(ctx, session, code)
}

func (f *XiaomiFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("xiaomi oauth does not support poll")
}

func defaultString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
