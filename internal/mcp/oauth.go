package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrOAuthFlowNotFound             = errors.New("mcp oauth: flow not found")
	ErrReauthRequired                = errors.New("mcp oauth: reauth required")
	errOAuthTokenEndpointUnavailable = errors.New("mcp oauth: token endpoint unavailable")
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

type OAuthAccountLabelStore interface {
	AccountLabelForInstance(instanceID string) (string, error)
}

type OAuthHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type OAuthEngine struct {
	store OAuthStore
	http  OAuthHTTPClient
}

type OAuthStartConfig struct {
	InstanceID        string
	AuthorizationURL  string
	RedirectURI       string
	ResourceURI       string
	ExpirationSeconds int
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	AccountLabel string `json:"account_label"`
	Subject      string `json:"subject"`
	Sub          string `json:"sub"`
	Email        string `json:"email"`
	Issuer       string `json:"issuer"`
	Iss          string `json:"iss"`
}

func NewOAuthEngine(store OAuthStore, client OAuthHTTPClient) *OAuthEngine {
	return &OAuthEngine{store: store, http: noRedirectOAuthClient(client)}
}

func BuildOAuthStartFlow(config OAuthStartConfig) (OAuthFlow, error) {
	if strings.TrimSpace(config.InstanceID) == "" {
		return OAuthFlow{}, fmt.Errorf("instance id is required")
	}
	if strings.TrimSpace(config.AuthorizationURL) == "" {
		return OAuthFlow{}, fmt.Errorf("authorization url is required")
	}
	if strings.TrimSpace(config.RedirectURI) == "" {
		return OAuthFlow{}, fmt.Errorf("redirect uri is required")
	}
	if strings.TrimSpace(config.ResourceURI) == "" {
		return OAuthFlow{}, fmt.Errorf("resource uri is required")
	}

	state, err := randomURLToken(24)
	if err != nil {
		return OAuthFlow{}, fmt.Errorf("generate state: %w", err)
	}
	verifier, err := randomURLToken(32)
	if err != nil {
		return OAuthFlow{}, fmt.Errorf("generate code verifier: %w", err)
	}
	redirectURI, err := redirectWithInstanceID(config.RedirectURI, config.InstanceID)
	if err != nil {
		return OAuthFlow{}, err
	}

	authorizationURL, err := url.Parse(config.AuthorizationURL)
	if err != nil {
		return OAuthFlow{}, fmt.Errorf("parse authorization url: %w", err)
	}
	query := authorizationURL.Query()
	query.Set("state", state)
	query.Set("resource", config.ResourceURI)
	query.Set("redirect_uri", redirectURI)
	query.Set("code_challenge_method", "S256")
	query.Set("code_challenge", pkceChallenge(verifier))
	authorizationURL.RawQuery = query.Encode()

	expiresAt := time.Now().Add(10 * time.Minute)
	if config.ExpirationSeconds > 0 {
		expiresAt = time.Now().Add(time.Duration(config.ExpirationSeconds) * time.Second)
	}

	return OAuthFlow{
		InstanceID:         config.InstanceID,
		State:              state,
		CodeVerifierSecret: verifier,
		RedirectURI:        redirectURI,
		AuthorizationURL:   authorizationURL.String(),
		ResourceURI:        config.ResourceURI,
		ExpiresAt:          expiresAt,
	}, nil
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

	tokenEndpoint, err := tokenEndpointForFlow(flow)
	if err != nil {
		return OAuthAccount{}, err
	}
	token, err := e.exchangeAuthorizationCode(ctx, tokenEndpoint, flow, code)
	if err != nil {
		return OAuthAccount{}, err
	}
	account := OAuthAccount{
		InstanceID:   instanceID,
		AccountLabel: accountLabelFromToken(token, selectedAccountLabel(e.store, instanceID)),
		Subject:      firstNonEmpty(token.Subject, token.Sub),
		Email:        token.Email,
		Issuer:       firstNonEmpty(token.Issuer, token.Iss),
		ResourceURI:  flow.ResourceURI,
		Scopes:       splitScopes(token.Scope),
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		AuthMetadata: map[string]string{"token_endpoint": tokenEndpoint},
	}
	if token.ExpiresIn > 0 {
		account.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if err := e.store.SaveAccount(account); err != nil {
		return OAuthAccount{}, err
	}
	return account, nil
}

func (e *OAuthEngine) RefreshAccount(ctx context.Context, account OAuthAccount) (OAuthAccount, error) {
	if account.RefreshToken == "" {
		return OAuthAccount{}, ErrReauthRequired
	}
	tokenEndpoint := ""
	if account.AuthMetadata != nil {
		tokenEndpoint = strings.TrimSpace(account.AuthMetadata["token_endpoint"])
	}
	if tokenEndpoint == "" {
		return OAuthAccount{}, ErrReauthRequired
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", account.RefreshToken)
	if account.ResourceURI != "" {
		form.Set("resource", account.ResourceURI)
	}
	token, err := e.postToken(ctx, tokenEndpoint, form)
	if err != nil {
		return OAuthAccount{}, err
	}
	if token.AccessToken == "" {
		return OAuthAccount{}, errors.New("refresh token response: access token is required")
	}

	refreshed := account
	refreshed.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	if token.Scope != "" {
		refreshed.Scopes = splitScopes(token.Scope)
	}
	if token.ExpiresIn > 0 {
		refreshed.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if refreshed.AuthMetadata == nil {
		refreshed.AuthMetadata = map[string]string{}
	}
	refreshed.AuthMetadata["token_endpoint"] = tokenEndpoint
	if err := e.store.SaveAccount(refreshed); err != nil {
		return OAuthAccount{}, err
	}
	return refreshed, nil
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

func (e *OAuthEngine) exchangeAuthorizationCode(ctx context.Context, tokenEndpoint string, flow OAuthFlow, code string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", flow.CodeVerifierSecret)
	form.Set("redirect_uri", flow.RedirectURI)
	if flow.ResourceURI != "" {
		form.Set("resource", flow.ResourceURI)
	}
	token, err := e.postToken(ctx, tokenEndpoint, form)
	if err != nil {
		return tokenResponse{}, err
	}
	if token.AccessToken == "" {
		return tokenResponse{}, errors.New("token response: access token is required")
	}
	return token, nil
}

func (e *OAuthEngine) postToken(ctx context.Context, tokenEndpoint string, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("post token request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			return tokenResponse{}, errOAuthTokenEndpointUnavailable
		}
		return tokenResponse{}, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}
	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return tokenResponse{}, fmt.Errorf("decode token response: %w", err)
	}
	return token, nil
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

func tokenEndpointForFlow(flow OAuthFlow) (string, error) {
	parsed, err := url.Parse(flow.AuthorizationURL)
	if err != nil {
		return "", fmt.Errorf("parse authorization url: %w", err)
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return "", errOAuthTokenEndpointUnavailable
	}
	if strings.HasSuffix(parsed.Path, "/authorize") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/authorize") + "/token"
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String(), nil
	}
	return "", errOAuthTokenEndpointUnavailable
}

func redirectWithInstanceID(rawURL, instanceID string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse redirect uri: %w", err)
	}
	query := parsed.Query()
	query.Set("instance_id", "b64:"+base64.RawURLEncoding.EncodeToString([]byte(instanceID)))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func randomURLToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func splitScopes(scope string) []string {
	fields := strings.Fields(scope)
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func accountLabelFromToken(token tokenResponse, selected string) string {
	if token.AccountLabel != "" {
		return token.AccountLabel
	}
	if selected != "" {
		return selected
	}
	if token.Email != "" {
		return token.Email
	}
	if token.Subject != "" {
		return token.Subject
	}
	if token.Sub != "" {
		return token.Sub
	}
	return "default"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func selectedAccountLabel(store OAuthStore, instanceID string) string {
	labelStore, ok := store.(OAuthAccountLabelStore)
	if !ok {
		return ""
	}
	label, err := labelStore.AccountLabelForInstance(instanceID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(label)
}

func noRedirectOAuthClient(client OAuthHTTPClient) OAuthHTTPClient {
	if client == nil {
		return noRedirectHTTPClient(http.DefaultClient)
	}
	if httpClient, ok := client.(*http.Client); ok {
		return noRedirectHTTPClient(httpClient)
	}
	return client
}

func noRedirectHTTPClient(client *http.Client) *http.Client {
	cloned := *client
	cloned.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &cloned
}
