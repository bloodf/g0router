package oauth

import (
	"context"
	"strings"
	"time"
)

// ProviderID identifies an OAuth-capable provider.
type ProviderID string

func (p ProviderID) String() string {
	return string(p)
}

// CanonicalProviderID maps auth-flow IDs onto runtime provider IDs.
func CanonicalProviderID(provider ProviderID) string {
	normalized := ProviderID(strings.ToLower(strings.TrimSpace(string(provider))))
	switch normalized {
	case "codex":
		return "openai"
	case "openai":
		return "openai"
	case "vertex":
		return "vertex"
	case "github":
		return "github-copilot"
	case "github-copilot":
		return "github-copilot"
	case "gitlab":
		return "gitlab-duo"
	case "gitlab-duo":
		return "gitlab-duo"
	default:
		return string(CanonicalFlowProviderID(normalized))
	}
}

// CanonicalFlowProviderID maps user-facing aliases onto OAuth flow IDs.
func CanonicalFlowProviderID(provider ProviderID) ProviderID {
	switch ProviderID(strings.ToLower(strings.TrimSpace(string(provider)))) {
	case "openai":
		return "codex"
	case "github":
		return "github-copilot"
	case "gitlab":
		return "gitlab-duo"
	case "vertex":
		return "gemini"
	default:
		return ProviderID(strings.ToLower(strings.TrimSpace(string(provider))))
	}
}

// AuthSession is returned when an OAuth flow has started.
type AuthSession struct {
	Provider     ProviderID `json:"provider"`
	AuthURL      string     `json:"auth_url,omitempty"`
	SessionID    string     `json:"session_id,omitempty"`
	UserCode     string     `json:"user_code,omitempty"`
	Verification string     `json:"verification,omitempty"`
	ExpiresIn    int        `json:"expires_in,omitempty"`
	PollInterval int        `json:"poll_interval,omitempty"`
}

// TokenResult is the credential material produced by a completed OAuth flow.
type TokenResult struct {
	Provider     ProviderID `json:"provider"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	TokenType    string     `json:"token_type,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
}

// PollStatus describes the state of a device-code style OAuth poll.
type PollStatus string

const (
	PollStatusPending  PollStatus = "pending"
	PollStatusComplete PollStatus = "complete"
	PollStatusSlowDown PollStatus = "slow_down"
	PollStatusExpired  PollStatus = "expired"
	PollStatusDenied   PollStatus = "denied"
)

func (s PollStatus) String() string {
	return string(s)
}

// PollResult is returned after checking an OAuth session for completion.
type PollResult struct {
	Status PollStatus   `json:"status"`
	Token  *TokenResult `json:"token,omitempty"`
}

// Flow is implemented by provider-specific OAuth flows.
type Flow interface {
	ProviderID() ProviderID
	Start(ctx context.Context) (AuthSession, error)
	Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error)
	Poll(ctx context.Context, session AuthSession) (PollResult, error)
}

type RefreshableFlow interface {
	Flow
	Refresh(ctx context.Context, refreshToken string) (TokenResult, error)
}
