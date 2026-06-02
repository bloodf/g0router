package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrOAuthFlowNotFound = errors.New("mcp oauth: flow not found")
	ErrReauthRequired    = errors.New("mcp oauth: reauth required")
)

type OAuthFlow struct {
	ID                 string
	InstanceID         string
	State              string
	CodeVerifierSecret string
	RedirectURI        string
	AuthorizationURL   string
	ResourceURI        string
	ExpiresAt          time.Time
	CreatedAt          string
}

type OAuthAccount struct {
	ID           string
	InstanceID   string
	AccountLabel string
	Subject      string
	Email        string
	Issuer       string
	ResourceURI  string
	Scopes       []string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	AuthMetadata map[string]string
	CreatedAt    string
	UpdatedAt    string
}

type OAuthStore interface {
	ConsumeFlow(instanceID, state string) (OAuthFlow, error)
	SaveAccount(account OAuthAccount) error
}

type OAuthHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type OAuthEngine struct {
	store OAuthStore
	http  OAuthHTTPClient
}

func NewOAuthEngine(store OAuthStore, client OAuthHTTPClient) *OAuthEngine {
	if client == nil {
		client = http.DefaultClient
	}
	return &OAuthEngine{store: store, http: client}
}

func (e *OAuthEngine) CompleteCallback(ctx context.Context, instanceID, callbackURL string) (OAuthAccount, error) {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return OAuthAccount{}, fmt.Errorf("parse callback url: %w", err)
	}
	code := parsed.Query().Get("code")
	state := parsed.Query().Get("state")
	if code == "" || state == "" {
		return OAuthAccount{}, ErrOAuthFlowNotFound
	}
	flow, err := e.store.ConsumeFlow(instanceID, state)
	if err != nil {
		return OAuthAccount{}, err
	}
	if !flow.ExpiresAt.IsZero() && time.Now().After(flow.ExpiresAt) {
		return OAuthAccount{}, ErrReauthRequired
	}

	account := OAuthAccount{
		InstanceID:   instanceID,
		AccountLabel: "default",
		ResourceURI:  flow.ResourceURI,
		AccessToken:  "mcp_" + code,
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	if err := e.store.SaveAccount(account); err != nil {
		return OAuthAccount{}, err
	}
	return account, nil
}

func (e *OAuthEngine) AuthorizeRequest(req *http.Request, account OAuthAccount) error {
	if account.AccessToken == "" || (!account.ExpiresAt.IsZero() && time.Now().After(account.ExpiresAt)) {
		return ErrReauthRequired
	}
	if account.ResourceURI != "" && !strings.HasPrefix(req.URL.String(), account.ResourceURI) {
		return ErrReauthRequired
	}
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	req.Header.Set("MCP-Protocol-Version", protocolVersion)
	return nil
}

type CredentialEnv struct {
	Actual   map[string]string
	Redacted map[string]string
}

func StdioCredentialEnv(account OAuthAccount) CredentialEnv {
	actual := map[string]string{
		"MCP_ACCESS_TOKEN": account.AccessToken,
	}
	if account.RefreshToken != "" {
		actual["MCP_REFRESH_TOKEN"] = account.RefreshToken
	}
	return CredentialEnv{
		Actual:   actual,
		Redacted: redactSecretMap(actual),
	}
}

var _ = context.Background
