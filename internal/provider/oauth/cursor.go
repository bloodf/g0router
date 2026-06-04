package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	cursorProviderID ProviderID = "cursor"
	cursorLoginURL              = "https://cursor.com/loginDeepControl"
	cursorPollURL               = "https://api2.cursor.sh/auth/poll"
	cursorRefreshURL            = "https://api2.cursor.sh/auth/exchange_user_api_key"
)

// CursorConfig configures the Cursor loginDeepControl OAuth flow.
type CursorConfig struct {
	LoginURL   string
	PollURL    string
	RefreshURL string
	HTTPClient *http.Client
	NewUUID    func() (string, error)
}

// CursorFlow implements Cursor's OMP-style loginDeepControl OAuth flow.
type CursorFlow struct {
	client     *http.Client
	loginURL   string
	pollURL    string
	refreshURL string
	newUUID    func() (string, error)
}

// NewCursorFlow returns a Cursor OAuth flow with the documented defaults.
func NewCursorFlow() *CursorFlow {
	return NewCursorFlowWithConfig(CursorConfig{})
}

// NewCursorFlowWithConfig returns a Cursor OAuth flow with testable endpoints.
func NewCursorFlowWithConfig(config CursorConfig) *CursorFlow {
	loginURL := config.LoginURL
	if loginURL == "" {
		loginURL = cursorLoginURL
	}
	pollURL := config.PollURL
	if pollURL == "" {
		pollURL = cursorPollURL
	}
	refreshURL := config.RefreshURL
	if refreshURL == "" {
		refreshURL = cursorRefreshURL
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	newUUID := config.NewUUID
	if newUUID == nil {
		newUUID = newCursorUUID
	}

	return &CursorFlow{
		client:     client,
		loginURL:   loginURL,
		pollURL:    pollURL,
		refreshURL: refreshURL,
		newUUID:    newUUID,
	}
}

func (f *CursorFlow) ProviderID() ProviderID {
	return cursorProviderID
}

func (f *CursorFlow) Start(ctx context.Context) (AuthSession, error) {
	verifier, err := randomURLToken(32)
	if err != nil {
		return AuthSession{}, fmt.Errorf("generate code verifier: %w", err)
	}
	uuid, err := f.newUUID()
	if err != nil {
		return AuthSession{}, fmt.Errorf("generate uuid: %w", err)
	}
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return AuthSession{}, fmt.Errorf("generate uuid: empty uuid")
	}

	loginURL, err := url.Parse(f.loginURL)
	if err != nil {
		return AuthSession{}, fmt.Errorf("parse login url: %w", err)
	}
	query := loginURL.Query()
	query.Set("challenge", codeChallenge(verifier))
	query.Set("uuid", uuid)
	query.Set("mode", "login")
	query.Set("redirectTarget", "cli")
	loginURL.RawQuery = query.Encode()
	authURL := loginURL.String()

	return AuthSession{
		Provider:     f.ProviderID(),
		AuthURL:      authURL,
		SessionID:    uuid + "." + verifier,
		UserCode:     uuid,
		Verification: authURL,
		PollInterval: 1,
	}, nil
}

func (f *CursorFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	if session.Provider != f.ProviderID() {
		return TokenResult{}, fmt.Errorf("cursor exchange: provider mismatch: %s", session.Provider)
	}
	return TokenResult{}, fmt.Errorf("cursor exchange: callback exchange is unsupported; use poll")
}

func (f *CursorFlow) Refresh(ctx context.Context, refreshToken string) (TokenResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return TokenResult{}, fmt.Errorf("cursor refresh: refresh token is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.refreshURL, strings.NewReader("{}"))
	if err != nil {
		return TokenResult{}, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return TokenResult{}, fmt.Errorf("refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if readErr != nil {
			return TokenResult{}, fmt.Errorf("refresh token: status %d: read body: %w", resp.StatusCode, readErr)
		}
		return TokenResult{}, fmt.Errorf("refresh token: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	token, err := decodeCursorToken(resp.Body)
	if err != nil {
		return TokenResult{}, fmt.Errorf("refresh token: %w", err)
	}
	return token, nil
}

func (f *CursorFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	if session.Provider != f.ProviderID() {
		return PollResult{}, fmt.Errorf("cursor poll: provider mismatch: %s", session.Provider)
	}
	uuid, verifier, err := parseCursorSessionID(session.SessionID)
	if err != nil {
		return PollResult{}, fmt.Errorf("cursor poll: %w", err)
	}

	pollURL, err := url.Parse(f.pollURL)
	if err != nil {
		return PollResult{}, fmt.Errorf("parse poll url: %w", err)
	}
	query := pollURL.Query()
	query.Set("uuid", uuid)
	query.Set("verifier", verifier)
	pollURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL.String(), nil)
	if err != nil {
		return PollResult{}, fmt.Errorf("create poll request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return PollResult{}, fmt.Errorf("poll token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return PollResult{Status: PollStatusPending}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if readErr != nil {
			return PollResult{}, fmt.Errorf("poll token: status %d: read body: %w", resp.StatusCode, readErr)
		}
		return PollResult{}, fmt.Errorf("poll token: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	token, err := decodeCursorToken(resp.Body)
	if err != nil {
		return PollResult{}, fmt.Errorf("poll token: %w", err)
	}
	return PollResult{Status: PollStatusComplete, Token: &token}, nil
}

type cursorTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func parseCursorSessionID(sessionID string) (string, string, error) {
	parts := strings.SplitN(sessionID, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid session id")
	}
	return parts[0], parts[1], nil
}

func decodeCursorToken(reader io.Reader) (TokenResult, error) {
	var token cursorTokenResponse
	if err := json.NewDecoder(reader).Decode(&token); err != nil {
		return TokenResult{}, fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" {
		return TokenResult{}, fmt.Errorf("decode token response: access token is required")
	}
	return TokenResult{
		Provider:     cursorProviderID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    cursorTokenExpiry(token.AccessToken, time.Now()),
	}, nil
}

func cursorTokenExpiry(accessToken string, now time.Time) time.Time {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return now.Add(time.Hour)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
	}
	if err != nil {
		return now.Add(time.Hour)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp <= 0 {
		return now.Add(time.Hour)
	}
	return time.Unix(claims.Exp, 0).Add(-5 * time.Minute)
}

func newCursorUUID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	data[6] = (data[6] & 0x0f) | 0x40
	data[8] = (data[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		data[0:4],
		data[4:6],
		data[6:8],
		data[8:10],
		data[10:16],
	), nil
}
