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
	githubCopilotProviderID    ProviderID = "github-copilot"
	githubCopilotClientID                 = "Iv1.b507a08c87ecfe98"
	githubCopilotDeviceCodeURL            = "https://github.com/login/device/code"
	githubCopilotTokenURL                 = "https://github.com/login/oauth/access_token"
	githubCopilotScope                    = "read:user"
)

// GitHubCopilotFlowConfig configures the GitHub Copilot device-code OAuth flow.
type GitHubCopilotFlowConfig struct {
	ClientID      string
	DeviceCodeURL string
	TokenURL      string
	Scopes        []string
	HTTPClient    *http.Client
}

// GitHubCopilotFlow implements GitHub Copilot device-code OAuth.
type GitHubCopilotFlow struct {
	client        *http.Client
	clientID      string
	deviceCodeURL string
	tokenURL      string
	scopes        []string
}

// NewGitHubCopilotFlow returns a GitHub Copilot OAuth flow using defaults for zero config fields.
func NewGitHubCopilotFlow(config GitHubCopilotFlowConfig) *GitHubCopilotFlow {
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	clientID := config.ClientID
	if clientID == "" {
		clientID = githubCopilotClientID
	}

	deviceCodeURL := config.DeviceCodeURL
	if deviceCodeURL == "" {
		deviceCodeURL = githubCopilotDeviceCodeURL
	}

	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = githubCopilotTokenURL
	}

	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{githubCopilotScope}
	}

	return &GitHubCopilotFlow{
		client:        client,
		clientID:      clientID,
		deviceCodeURL: deviceCodeURL,
		tokenURL:      tokenURL,
		scopes:        append([]string(nil), scopes...),
	}
}

func (f *GitHubCopilotFlow) ProviderID() ProviderID {
	return githubCopilotProviderID
}

func (f *GitHubCopilotFlow) Start(ctx context.Context) (AuthSession, error) {
	var response deviceCodeResponse
	if err := f.postForm(ctx, f.deviceCodeURL, url.Values{
		"client_id": {f.clientID},
		"scope":     {strings.Join(f.scopes, " ")},
	}, &response); err != nil {
		return AuthSession{}, fmt.Errorf("start github copilot device flow: %w", err)
	}

	authURL := response.VerificationURIComplete
	if authURL == "" {
		authURL = response.VerificationURI
	}

	return AuthSession{
		Provider:     githubCopilotProviderID,
		AuthURL:      authURL,
		SessionID:    response.DeviceCode,
		UserCode:     response.UserCode,
		Verification: response.VerificationURI,
		ExpiresIn:    response.ExpiresIn,
		PollInterval: response.Interval,
	}, nil
}

func (f *GitHubCopilotFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return TokenResult{}, errors.New("github copilot oauth does not support callback exchange")
}

func (f *GitHubCopilotFlow) Refresh(ctx context.Context, refreshToken string) (TokenResult, error) {
	return refreshTokenGrant(ctx, f.client, f.tokenURL, f.clientID, githubCopilotProviderID, refreshToken)
}

func (f *GitHubCopilotFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
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
		return PollResult{}, fmt.Errorf("poll github copilot device flow: %w", err)
	}

	if response.AccessToken == "" {
		return PollResult{}, errors.New("poll github copilot device flow: missing access token")
	}

	token := TokenResult{
		Provider:     githubCopilotProviderID,
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

func (f *GitHubCopilotFlow) postForm(ctx context.Context, endpoint string, form url.Values, target any) error {
	codex := CodexFlow{client: f.client}
	return codex.postForm(ctx, endpoint, form, target)
}
