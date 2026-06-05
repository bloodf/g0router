package oauth

import (
	"context"
	"errors"
	"net/http"
)

const (
	gitlabProviderID  ProviderID = "gitlab-duo"
	gitlabClientID               = "da4edff2e6ebd2bc3208611e2768bc1c1dd7be791dc5ff26ca34ca9ee44f7d4b"
	gitlabAuthURL                = "https://gitlab.com/oauth/authorize"
	gitlabTokenURL               = "https://gitlab.com/oauth/token"
	gitlabRedirectURI            = "http://localhost:8080/callback"
)

// GitLabConfig configures the GitLab callback-based PKCE OAuth flow.
type GitLabConfig struct {
	ClientID    string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	Scopes      []string
	HTTPClient  *http.Client
}

// GitLabFlow implements GitLab callback-based PKCE OAuth.
type GitLabFlow struct {
	oauth *callbackOAuthFlow
}

// NewGitLabFlow returns a GitLab OAuth flow using defaults for zero config fields.
func NewGitLabFlow(config GitLabConfig) *GitLabFlow {
	authURL := config.AuthURL
	if authURL == "" {
		authURL = gitlabAuthURL
	}
	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = gitlabTokenURL
	}
	redirectURI := config.RedirectURI
	if redirectURI == "" {
		redirectURI = gitlabRedirectURI
	}
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"api"}
	}

	return &GitLabFlow{
		oauth: newCallbackOAuthFlow(callbackOAuthConfig{
			Provider:    gitlabProviderID,
			ClientID:    config.ClientID,
			DefaultID:   gitlabClientID,
			AuthURL:     authURL,
			TokenURL:    tokenURL,
			RedirectURI: redirectURI,
			Scopes:      scopes,
			HTTPClient:  config.HTTPClient,
		}),
	}
}

func (f *GitLabFlow) ProviderID() ProviderID {
	return gitlabProviderID
}

func (f *GitLabFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.oauth.start(ctx)
}

func (f *GitLabFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.oauth.exchange(ctx, session, code)
}

func (f *GitLabFlow) Refresh(ctx context.Context, refreshToken string) (TokenResult, error) {
	return f.oauth.refresh(ctx, refreshToken)
}

func (f *GitLabFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("gitlab-duo oauth does not support poll")
}
