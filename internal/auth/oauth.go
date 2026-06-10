package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// oauthStateTTL bounds how long an in-flight authorization may take.
const oauthStateTTL = 10 * time.Minute

// defaultAnthropicClientID is the public Claude Code OAuth client identifier.
// Parity source: _refs/9router/src/lib/oauth/constants/oauth.js:21
const defaultAnthropicClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"

// OAuthConfig describes one provider's OAuth endpoints.
type OAuthConfig struct {
	Provider     string
	ClientID     string
	AuthorizeURL string
	TokenURL     string
	RedirectURI  string
	Scopes       []string
}

// AnthropicOAuth returns the production OAuth configuration for Anthropic
// (Claude Pro/Max OAuth with PKCE).
func AnthropicOAuth() OAuthConfig {
	clientID := os.Getenv("G0ROUTER_ANTHROPIC_CLIENT_ID")
	if clientID == "" {
		clientID = defaultAnthropicClientID
	}
	return OAuthConfig{
		Provider:     "anthropic",
		ClientID:     clientID,
		AuthorizeURL: "https://claude.ai/oauth/authorize",
		TokenURL:     "https://console.anthropic.com/v1/oauth/token",
		RedirectURI:  "https://console.anthropic.com/oauth/code/callback",
		Scopes:       []string{"org:create_api_key", "user:profile", "user:inference"},
	}
}

// OAuthToken is the result of a code exchange or refresh.
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// OAuthFlow drives the authorization-code-with-PKCE flow for one provider,
// persisting in-flight state in the store.
type OAuthFlow struct {
	cfg    OAuthConfig
	store  *store.Store
	client *http.Client
}

// NewOAuthFlow creates a flow. client may be nil to use a default HTTP client.
func NewOAuthFlow(cfg OAuthConfig, st *store.Store, client *http.Client) *OAuthFlow {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &OAuthFlow{cfg: cfg, store: st, client: client}
}

// Config returns the flow's provider configuration.
func (f *OAuthFlow) Config() OAuthConfig {
	return f.cfg
}

// Start generates state + PKCE verifier, persists them, and returns the
// authorization URL the user should visit.
func (f *OAuthFlow) Start() (authURL, state string, err error) {
	state, err = randomURLSafe(32)
	if err != nil {
		return "", "", fmt.Errorf("generate oauth state: %w", err)
	}
	verifier, err := randomURLSafe(64)
	if err != nil {
		return "", "", fmt.Errorf("generate oauth verifier: %w", err)
	}

	if err := f.store.CreateOAuthSession(&store.OAuthSession{
		State:     state,
		Provider:  f.cfg.Provider,
		Verifier:  verifier,
		ExpiresAt: time.Now().Add(oauthStateTTL).Unix(),
	}); err != nil {
		return "", "", fmt.Errorf("persist oauth state: %w", err)
	}

	challenge := pkceChallenge(verifier)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", f.cfg.ClientID)
	q.Set("redirect_uri", f.cfg.RedirectURI)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	if len(f.cfg.Scopes) > 0 {
		q.Set("scope", strings.Join(f.cfg.Scopes, " "))
	}
	return f.cfg.AuthorizeURL + "?" + q.Encode(), state, nil
}

// Exchange consumes the persisted state and trades the authorization code
// for tokens at the provider's token endpoint.
func (f *OAuthFlow) Exchange(state, code string) (*OAuthToken, error) {
	sess, err := f.store.ConsumeOAuthSession(state)
	if err != nil {
		return nil, fmt.Errorf("consume oauth state: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", f.cfg.ClientID)
	form.Set("redirect_uri", f.cfg.RedirectURI)
	form.Set("code_verifier", sess.Verifier)
	form.Set("state", state)
	return f.requestToken(form)
}

// Refresh trades a refresh token for a new access token.
func (f *OAuthFlow) Refresh(refreshToken string) (*OAuthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", f.cfg.ClientID)
	return f.requestToken(form)
}

func (f *OAuthFlow) requestToken(form url.Values) (*OAuthToken, error) {
	resp, err := f.client.PostForm(f.cfg.TokenURL, form)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	tok := &OAuthToken{AccessToken: parsed.AccessToken, RefreshToken: parsed.RefreshToken}
	if parsed.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second).Unix()
	}
	return tok, nil
}

func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := randRead(b); err != nil {
		return "", fmt.Errorf("random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
