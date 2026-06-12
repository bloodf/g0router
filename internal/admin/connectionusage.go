package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

// authExpiredPatterns are substrings that indicate a provider rejected the
// current access token. They must be checked case-insensitively.
var authExpiredPatterns = []string{"expired", "authentication", "unauthorized", "401", "re-authorize"}

// ConnectionUsageHandler serves GET /api/usage/{connectionId}.
type ConnectionUsageHandler struct {
	Handlers *Handlers
	// HTTPClient is the client used for provider API calls. nil means
	// http.DefaultClient.
	HTTPClient *http.Client
	// Fetcher loads usage data for a provider connection. nil means
	// usage.FetchProviderUsage.
	Fetcher func(providerType string, conn *store.Connection, client *http.Client, baseURL ...string) (map[string]any, error)
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

	resolver := auth.NewCredentialResolver(h.Handlers.store, h.Handlers.flows)

	// Refresh-if-needed before fetching.
	key, _, err := resolver.ResolveKey(provider.ID)
	if err != nil {
		writeError(ctx, fasthttp.StatusUnauthorized, fmt.Sprintf("Credential refresh failed: %v", err))
		return
	}

	accessToken := key.Value
	if accessToken == "" {
		accessToken = conn.AccessToken
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
		newToken, refreshErr := resolver.RefreshCredentials(conn.ID)
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
