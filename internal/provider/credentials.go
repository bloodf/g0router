package provider

import (
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
)

func ConnectionFromOAuthToken(token oauth.TokenResult, accountLabel string) *store.Connection {
	return ConnectionFromOAuthTokenForProvider(token, accountLabel, CanonicalProviderID(token.Provider.String()))
}

func ConnectionFromOAuthTokenForProvider(token oauth.TokenResult, accountLabel, runtimeProvider string) *store.Connection {
	provider := CanonicalProviderID(runtimeProvider)
	name := strings.TrimSpace(accountLabel)
	if name == "" {
		name = provider
	}

	accessToken := token.AccessToken
	conn := &store.Connection{
		Provider:     provider,
		Name:         name,
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &accessToken,
		RefreshToken: emptyStringPtr(token.RefreshToken),
		ExpiresAt:    timePtrUnix(token.ExpiresAt),
		IsActive:     true,
		ProviderSpecificData: map[string]any{
			"oauth_provider": token.Provider.String(),
			"token_type":     token.TokenType,
			"scopes":         append([]string(nil), token.Scopes...),
		},
	}
	if token.TokenType == "api_key" {
		conn.AuthType = store.AuthTypeAPIKey
		conn.APIKey = &accessToken
		conn.AccessToken = nil
		conn.RefreshToken = nil
		conn.ExpiresAt = nil
	}
	return conn
}

func emptyStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func timePtrUnix(value time.Time) *int64 {
	if value.IsZero() {
		return nil
	}
	unix := value.Unix()
	return &unix
}
