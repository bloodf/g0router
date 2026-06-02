package oauth

import (
	"context"
	"errors"
	"net/http"
)

const (
	antigravityProviderID ProviderID = "antigravity"
	antigravityClientID              = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
)

// AntigravityConfig configures the Antigravity Google PKCE OAuth flow.
type AntigravityConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// AntigravityFlow implements Antigravity's Google callback-based PKCE OAuth flow.
type AntigravityFlow struct {
	google *googleOAuthFlow
}

// NewAntigravityFlow returns an Antigravity OAuth flow using defaults for zero config fields.
func NewAntigravityFlow(config AntigravityConfig) *AntigravityFlow {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/cclog",
			"https://www.googleapis.com/auth/experimentsandconfigs",
		}
	}

	return &AntigravityFlow{
		google: newGoogleOAuthFlow(googleOAuthConfig{
			Provider:    antigravityProviderID,
			ClientID:    config.ClientID,
			DefaultID:   antigravityClientID,
			AuthURL:     config.AuthURL,
			TokenURL:    config.TokenURL,
			RedirectURI: config.RedirectURI,
			Scopes:      scopes,
			HTTPClient:  config.HTTPClient,
		}),
	}
}

func (f *AntigravityFlow) ProviderID() ProviderID {
	return antigravityProviderID
}

func (f *AntigravityFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.google.start(ctx)
}

func (f *AntigravityFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.google.exchange(ctx, session, code)
}

func (f *AntigravityFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("antigravity oauth does not support poll")
}
