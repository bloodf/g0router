package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	minimaxProviderID ProviderID = "minimax"
	minimaxAuthURL               = "https://www.minimax.io/platform/user-center/basic-information/interface-key"
)

// MiniMaxFlow implements MiniMax direct API-key credential capture.
type MiniMaxFlow struct {
	apiKey *apiKeyFlow
}

// NewMiniMaxFlow returns a MiniMax API-key compatible OAuth flow.
func NewMiniMaxFlow() *MiniMaxFlow {
	return &MiniMaxFlow{
		apiKey: newAPIKeyFlow(minimaxProviderID, minimaxAuthURL),
	}
}

func (f *MiniMaxFlow) ProviderID() ProviderID {
	return minimaxProviderID
}

func (f *MiniMaxFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.apiKey.start(ctx)
}

func (f *MiniMaxFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.apiKey.exchange(ctx, session, code)
}

func (f *MiniMaxFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("minimax api key flow does not support poll")
}

type apiKeyFlow struct {
	provider ProviderID
	authURL  string
}

func newAPIKeyFlow(provider ProviderID, authURL string) *apiKeyFlow {
	return &apiKeyFlow{
		provider: provider,
		authURL:  authURL,
	}
}

func (f *apiKeyFlow) start(ctx context.Context) (AuthSession, error) {
	return AuthSession{
		Provider: f.provider,
		AuthURL:  f.authURL,
	}, nil
}

func (f *apiKeyFlow) exchange(ctx context.Context, session AuthSession, apiKey string) (TokenResult, error) {
	if session.Provider != f.provider {
		return TokenResult{}, fmt.Errorf("%s exchange: provider mismatch: %s", f.provider, session.Provider)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return TokenResult{}, fmt.Errorf("%s exchange: api key is required", f.provider)
	}

	return TokenResult{
		Provider:    f.provider,
		AccessToken: apiKey,
		TokenType:   "api_key",
	}, nil
}
