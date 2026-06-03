package oauth

import (
	"context"
	"errors"
	"net/http"
)

const (
	kiroProviderID  ProviderID = "kiro"
	kiroClientID               = "kiro"
	kiroAuthURL                = "https://auth.kiro.dev/oauth/authorize"
	kiroTokenURL               = "https://auth.kiro.dev/oauth/token"
	kiroRedirectURI            = "http://localhost:54545/oauth/callback"
)

// KiroConfig configures the Kiro callback-based PKCE OAuth flow.
type KiroConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// KiroFlow implements Kiro callback-based PKCE OAuth.
type KiroFlow struct {
	oauth *callbackOAuthFlow
}

// NewKiroFlow returns a Kiro OAuth flow using defaults for zero config fields.
func NewKiroFlow(config KiroConfig) *KiroFlow {
	authURL := config.AuthURL
	if authURL == "" {
		authURL = kiroAuthURL
	}
	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = kiroTokenURL
	}
	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = kiroRedirectURI
	}
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	return &KiroFlow{
		oauth: newCallbackOAuthFlow(callbackOAuthConfig{
			Provider:    kiroProviderID,
			ClientID:    config.ClientID,
			DefaultID:   kiroClientID,
			AuthURL:     authURL,
			TokenURL:    tokenURL,
			RedirectURI: redirectURI,
			Scopes:      scopes,
			HTTPClient:  config.HTTPClient,
		}),
	}
}

func (f *KiroFlow) ProviderID() ProviderID {
	return kiroProviderID
}

func (f *KiroFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.oauth.start(ctx)
}

func (f *KiroFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.oauth.exchange(ctx, session, code)
}

func (f *KiroFlow) Refresh(ctx context.Context, refreshToken string) (TokenResult, error) {
	return f.oauth.refresh(ctx, refreshToken)
}

func (f *KiroFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("kiro oauth does not support poll")
}
