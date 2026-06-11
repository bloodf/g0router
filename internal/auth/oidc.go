package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// OIDC cookie constants.
const (
	OIDCStateCookieName    = "oidc_state"
	OIDCNonceCookieName    = "oidc_nonce"
	OIDCVerifierCookieName = "oidc_code_verifier"
	OIDCCookieMaxAge       = 600
)

const oidcDefaultScopes = "openid profile email"

// OIDCDiscovery is a minimal OpenID Connect discovery document.
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// FetchOIDCDiscovery fetches and parses the issuer's well-known configuration.
func FetchOIDCDiscovery(issuerURL string, client *http.Client) (*OIDCDiscovery, error) {
	if client == nil {
		client = http.DefaultClient
	}
	issuerURL = strings.TrimRight(issuerURL, "/")
	req, err := http.NewRequest(http.MethodGet, issuerURL+"/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, fmt.Errorf("build discovery request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read discovery document: %w", err)
	}

	var doc OIDCDiscovery
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decode discovery document: %w", err)
	}
	return &doc, nil
}

// OIDCPKCEPair holds a PKCE verifier and its S256 challenge.
type OIDCPKCEPair struct {
	Verifier  string
	Challenge string
}

// CreateOIDCPKCEPair generates a new PKCE verifier/challenge pair using the
// existing randomURLSafe and pkceChallenge primitives.
func CreateOIDCPKCEPair() (*OIDCPKCEPair, error) {
	verifier, err := randomURLSafe(32)
	if err != nil {
		return nil, fmt.Errorf("generate pkce verifier: %w", err)
	}
	return &OIDCPKCEPair{Verifier: verifier, Challenge: pkceChallenge(verifier)}, nil
}

// CreateOIDCState generates a random OIDC state value.
func CreateOIDCState() (string, error) {
	return randomURLSafe(16)
}

// CreateOIDCNonce generates a random OIDC nonce value.
func CreateOIDCNonce() (string, error) {
	return randomURLSafe(16)
}

// OIDCAuthURLParams describes the inputs for BuildOIDCAuthorizationURL.
type OIDCAuthURLParams struct {
	AuthorizationEndpoint string
	ClientID              string
	RedirectURI           string
	Scopes                string
	State                 string
	Nonce                 string
	CodeChallenge         string
}

// BuildOIDCAuthorizationURL builds the authorization request URL.
func BuildOIDCAuthorizationURL(p OIDCAuthURLParams) string {
	scopes := strings.TrimSpace(p.Scopes)
	if scopes == "" {
		scopes = oidcDefaultScopes
	}
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURI)
	q.Set("scope", scopes)
	q.Set("state", p.State)
	q.Set("nonce", p.Nonce)
	q.Set("code_challenge", p.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	return p.AuthorizationEndpoint + "?" + q.Encode()
}

// OIDCCodeExchangeParams describes the inputs for ExchangeOIDCCode.
type OIDCCodeExchangeParams struct {
	TokenEndpoint string
	ClientID      string
	ClientSecret  string
	Code          string
	RedirectURI   string
	CodeVerifier  string
}

// ExchangeOIDCCode trades the authorization code for tokens at the IdP.
func ExchangeOIDCCode(p OIDCCodeExchangeParams, client *http.Client) (map[string]any, error) {
	if client == nil {
		client = http.DefaultClient
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", p.ClientID)
	form.Set("code", p.Code)
	form.Set("redirect_uri", p.RedirectURI)
	form.Set("code_verifier", p.CodeVerifier)
	if p.ClientSecret != "" {
		form.Set("client_secret", p.ClientSecret)
	}

	resp, err := client.PostForm(p.TokenEndpoint, form)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := ""
		if d, ok := data["error_description"].(string); ok && d != "" {
			msg = d
		} else if e, ok := data["error"].(string); ok && e != "" {
			msg = e
		} else {
			msg = fmt.Sprintf("OIDC token exchange failed (%d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return data, nil
}

// ParseIDTokenPayload extracts the payload claims from an ID token's payload.
// Signature verification is intentionally not performed here; the caller uses
// the nonce claim only to bind the OIDC login to the original request.
func ParseIDTokenPayload(idToken string) (map[string]any, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("id_token must have three segments")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode id_token payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse id_token payload: %w", err)
	}
	return claims, nil
}

// VerifyOIDCNonce returns an error if the ID token's nonce claim does not
// match the stored nonce.
func VerifyOIDCNonce(idToken, nonce string) error {
	claims, err := ParseIDTokenPayload(idToken)
	if err != nil {
		return fmt.Errorf("parse id_token: %w", err)
	}
	got, _ := claims["nonce"].(string)
	if got != nonce {
		return fmt.Errorf("oidc nonce mismatch")
	}
	return nil
}

// ValidateOIDCState returns an error when the returned state does not match
// the state stored in the OIDC cookie.
func ValidateOIDCState(stored, returned string) error {
	if stored == "" || returned == "" || stored != returned {
		return fmt.Errorf("oidc state mismatch")
	}
	return nil
}

// OIDCSecretProbeResult is the result of ProbeOIDCClientSecret.
type OIDCSecretProbeResult struct {
	Tested  bool   `json:"tested"`
	Valid   *bool  `json:"valid"`
	Message string `json:"message"`
	Raw     any    `json:"raw,omitempty"`
}

// ProbeOIDCClientSecret sends a deliberately invalid code to the token endpoint
// and classifies the error to determine whether the client secret is accepted.
func ProbeOIDCClientSecret(tokenEndpoint, clientID, clientSecret, redirectURI string, client *http.Client) (*OIDCSecretProbeResult, error) {
	if clientSecret == "" {
		return &OIDCSecretProbeResult{
			Tested:  false,
			Valid:   nil,
			Message: "No client secret was provided, so secret validation was skipped.",
		}, nil
	}

	if client == nil {
		client = http.DefaultClient
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", "__oidc_test_invalid_code__")
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", "__oidc_test_invalid_verifier__")

	resp, err := client.PostForm(tokenEndpoint, form)
	if err != nil {
		return nil, fmt.Errorf("probe token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read probe response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode probe response: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		valid := true
		return &OIDCSecretProbeResult{
			Tested:  true,
			Valid:   &valid,
			Message: "Client secret was accepted by the token endpoint.",
			Raw:     data,
		}, nil
	}

	errorCode := ""
	if e, ok := data["error"].(string); ok {
		errorCode = strings.ToLower(e)
	}
	errorDescription := ""
	if d, ok := data["error_description"].(string); ok && d != "" {
		errorDescription = d
	} else if e, ok := data["error"].(string); ok && e != "" {
		errorDescription = e
	}

	clientPattern := regexp.MustCompile(`(?i)client.*(invalid|failed|mismatch)`)

	if errorCode == "invalid_client" || errorCode == "unauthorized_client" || clientPattern.MatchString(errorDescription) {
		valid := false
		msg := errorDescription
		if msg == "" {
			msg = "Client secret is not valid."
		}
		return &OIDCSecretProbeResult{
			Tested:  true,
			Valid:   &valid,
			Message: msg,
			Raw:     data,
		}, nil
	}

	grantPattern := regexp.MustCompile(`(?i)grant|code`)
	if errorCode == "invalid_grant" || errorCode == "invalid_code" || grantPattern.MatchString(errorDescription) {
		valid := true
		return &OIDCSecretProbeResult{
			Tested:  true,
			Valid:   &valid,
			Message: "Client secret was accepted; the token exchange failed only because the test authorization code is invalid.",
			Raw:     data,
		}, nil
	}

	msg := errorDescription
	if msg == "" {
		msg = fmt.Sprintf("Token endpoint responded with %d", resp.StatusCode)
	}
	return &OIDCSecretProbeResult{
		Tested:  true,
		Valid:   nil,
		Message: msg,
		Raw:     data,
	}, nil
}

// CreateOIDCSession issues a normal opaque dashboard session for an OIDC user.
func (s *Sessions) CreateOIDCSession(userID string) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	expiresAt := time.Now().Add(s.ttl).Unix()
	if err := s.store.CreateSession(token, userID, expiresAt); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}
