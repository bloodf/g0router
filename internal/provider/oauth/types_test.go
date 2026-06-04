package oauth

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type fakeFlow struct {
	provider ProviderID
}

func (f fakeFlow) ProviderID() ProviderID {
	return f.provider
}

func (f fakeFlow) Start(ctx context.Context) (AuthSession, error) {
	return AuthSession{
		Provider:     f.provider,
		AuthURL:      "https://example.com/oauth/authorize",
		SessionID:    "session-1",
		UserCode:     "ABCD-EFGH",
		Verification: "https://example.com/device",
	}, nil
}

func (f fakeFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return TokenResult{
		Provider:     session.Provider,
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Unix(1700000000, 0),
		Scopes:       []string{"chat"},
	}, nil
}

func (f fakeFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{
		Status: PollStatusPending,
	}, nil
}

func TestProviderIDString(t *testing.T) {
	provider := ProviderID("anthropic")

	if provider.String() != "anthropic" {
		t.Errorf("provider string: %q", provider.String())
	}
}

func TestCanonicalFlowProviderIDNormalizesAuthAliases(t *testing.T) {
	tests := []struct {
		provider ProviderID
		want     ProviderID
	}{
		{provider: ProviderID("openai"), want: ProviderID("codex")},
		{provider: ProviderID("codex"), want: ProviderID("codex")},
		{provider: ProviderID("github"), want: ProviderID("github-copilot")},
		{provider: ProviderID("github-copilot"), want: ProviderID("github-copilot")},
		{provider: ProviderID("  GitHub  "), want: ProviderID("github-copilot")},
		{provider: ProviderID("vertex"), want: ProviderID("gemini")},
		{provider: ProviderID("gemini"), want: ProviderID("gemini")},
		{provider: ProviderID("minimax"), want: ProviderID("minimax")},
	}

	for _, tt := range tests {
		if got := CanonicalFlowProviderID(tt.provider); got != tt.want {
			t.Fatalf("CanonicalFlowProviderID(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestCanonicalProviderIDKeepsVertexRuntimeProvider(t *testing.T) {
	tests := []struct {
		provider ProviderID
		want     string
	}{
		{provider: ProviderID("openai"), want: "openai"},
		{provider: ProviderID("codex"), want: "openai"},
		{provider: ProviderID("github"), want: "github-copilot"},
		{provider: ProviderID("github-copilot"), want: "github-copilot"},
		{provider: ProviderID("vertex"), want: "vertex"},
		{provider: ProviderID("gemini"), want: "gemini"},
	}

	for _, tt := range tests {
		if got := CanonicalProviderID(tt.provider); got != tt.want {
			t.Fatalf("CanonicalProviderID(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestAuthSessionJSONRoundTrip(t *testing.T) {
	session := AuthSession{
		Provider:     ProviderID("codex"),
		AuthURL:      "https://example.com/oauth/authorize",
		SessionID:    "session-1",
		UserCode:     "ABCD-EFGH",
		Verification: "https://example.com/device",
		ExpiresIn:    900,
		PollInterval: 5,
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got AuthSession
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Provider != ProviderID("codex") {
		t.Errorf("provider: %q", got.Provider)
	}
	if got.AuthURL != session.AuthURL {
		t.Errorf("auth url: %q", got.AuthURL)
	}
	if got.SessionID != "session-1" {
		t.Errorf("session id: %q", got.SessionID)
	}
	if got.PollInterval != 5 {
		t.Errorf("poll interval: %d", got.PollInterval)
	}
}

func TestTokenResultJSONRoundTrip(t *testing.T) {
	expiresAt := time.Unix(1700000000, 0).UTC()
	token := TokenResult{
		Provider:     ProviderID("github-copilot"),
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		Scopes:       []string{"chat", "models"},
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got TokenResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.AccessToken != "access-token" {
		t.Errorf("access token: %q", got.AccessToken)
	}
	if got.TokenType != "Bearer" {
		t.Errorf("token type: %q", got.TokenType)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expires at: %v", got.ExpiresAt)
	}
	if len(got.Scopes) != 2 || got.Scopes[1] != "models" {
		t.Errorf("scopes: %+v", got.Scopes)
	}
}

func TestPollStatusValues(t *testing.T) {
	tests := []struct {
		status PollStatus
		want   string
	}{
		{PollStatusPending, "pending"},
		{PollStatusComplete, "complete"},
		{PollStatusSlowDown, "slow_down"},
		{PollStatusExpired, "expired"},
		{PollStatusDenied, "denied"},
	}

	for _, tt := range tests {
		if tt.status.String() != tt.want {
			t.Errorf("status %q string: %q", tt.status, tt.status.String())
		}
	}
}

func TestFlowInterface(t *testing.T) {
	var flow Flow = fakeFlow{provider: ProviderID("anthropic")}

	if flow.ProviderID() != ProviderID("anthropic") {
		t.Errorf("provider: %q", flow.ProviderID())
	}

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if session.Provider != ProviderID("anthropic") {
		t.Errorf("session provider: %q", session.Provider)
	}

	token, err := flow.Exchange(context.Background(), session, "callback-code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if token.Provider != ProviderID("anthropic") || token.AccessToken == "" {
		t.Errorf("token: %+v", token)
	}

	poll, err := flow.Poll(context.Background(), session)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if poll.Status != PollStatusPending {
		t.Errorf("poll status: %q", poll.Status)
	}
}
