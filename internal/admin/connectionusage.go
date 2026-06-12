package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

// authExpiredPatterns are substrings that indicate a provider rejected the
// current access token. They must be checked case-insensitively.
var authExpiredPatterns = []string{"expired", "authentication", "unauthorized", "401", "re-authorize"}

// oauthLeadWindow mirrors auth.refreshLead so the admin handler can decide
// when to proactively refresh the requested connection's token without
// depending on unexported auth internals. Parity: anthropic=4h
// (appConstants.js:158), gemini/xai=5m, default 5m
// (tokenRefresh.js:35 TOKEN_EXPIRY_BUFFER_MS).
func oauthLeadWindow(providerType string) time.Duration {
	switch providerType {
	case "anthropic":
		return 4 * time.Hour
	case "gemini", "xai":
		return 5 * time.Minute
	default:
		return 5 * time.Minute
	}
}

// oauthTokenNeedsRefresh reports whether the connection's expiry is within
// the provider-specific lead window (or already expired).
func oauthTokenNeedsRefresh(conn *store.Connection, providerType string) bool {
	if conn.ExpiresAt == 0 {
		return false
	}
	return time.Until(time.Unix(conn.ExpiresAt, 0)) < oauthLeadWindow(providerType)
}

// ConnectionUsageHandler serves GET /api/usage/{connectionId}.
type ConnectionUsageHandler struct {
	Handlers *Handlers
	// HTTPClient is the client used for provider API calls. nil means
	// http.DefaultClient.
	HTTPClient *http.Client
	// Fetcher loads usage data for a provider connection. nil means
	// usage.FetchProviderUsage.
	Fetcher func(providerType string, conn *store.Connection, client *http.Client, baseURL ...string) (map[string]any, error)
	// Refresher refreshes the OAuth credentials for a connection and returns
	// the rotated access token. nil means
	// auth.NewCredentialResolver(store, flows).RefreshCredentials.
	Refresher func(connectionID string) (string, error)
}

// GetConnectionUsage returns provider quota/usage data for a single connection.
func (h *ConnectionUsageHandler) GetConnectionUsage(ctx *fasthttp.RequestCtx) {
	connectionID, ok := pathID(ctx.UserValue("connectionId"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid connection id")
		return
	}

	conn, err := h.Handlers.store.GetConnection(connectionID)
	if errors.Is(err, store.ErrNotFound) || conn == nil {
		writeError(ctx, fasthttp.StatusNotFound, "Connection not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "get connection")
		return
	}

	if conn.Kind != "oauth" {
		writeData(ctx, fasthttp.StatusOK, map[string]any{
			"message": "Usage not available for this connection",
		})
		return
	}

	provider, err := h.Handlers.store.GetProvider(conn.ProviderID)
	if errors.Is(err, store.ErrNotFound) || provider == nil {
		writeError(ctx, fasthttp.StatusNotFound, "Connection not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "get provider")
		return
	}

	refresher := h.Refresher
	if refresher == nil {
		resolver := auth.NewCredentialResolver(h.Handlers.store, h.Handlers.flows)
		refresher = resolver.RefreshCredentials
	}

	// Use the REQUESTED connection's own credentials. The previous
	// implementation resolved credentials by provider ID and ignored the
	// {connectionId} in the URL, which could fetch usage with a different
	// connection's token when multiple oauth connections exist for the same
	// provider. Refresh proactively when within the OAuth expiry lead window
	// (or already expired); otherwise use the stored access token as-is.
	accessToken := conn.AccessToken
	if oauthTokenNeedsRefresh(conn, provider.Type) {
		newToken, refreshErr := refresher(conn.ID)
		if refreshErr != nil {
			writeError(ctx, fasthttp.StatusUnauthorized, fmt.Sprintf("Credential refresh failed: %v", refreshErr))
			return
		}
		accessToken = newToken
	}

	fetchConn := *conn
	fetchConn.AccessToken = accessToken

	client := h.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	fetcher := h.Fetcher
	if fetcher == nil {
		fetcher = usage.FetchProviderUsage
	}

	usageData, err := fetcher(provider.Type, &fetchConn, client)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "fetch usage")
		return
	}

	// Auth-expired message + refresh token exists => force refresh and retry once.
	if isAuthExpiredMessage(usageData) && conn.RefreshToken != "" {
		newToken, refreshErr := refresher(conn.ID)
		if refreshErr == nil {
			fetchConn.AccessToken = newToken
			if retryUsage, retryErr := fetcher(provider.Type, &fetchConn, client); retryErr == nil {
				usageData = retryUsage
			}
		}
	}

	writeData(ctx, fasthttp.StatusOK, usageData)
}

func isAuthExpiredMessage(usage map[string]any) bool {
	msg, ok := usage["message"].(string)
	if !ok {
		return false
	}
	lower := strings.ToLower(msg)
	for _, p := range authExpiredPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
