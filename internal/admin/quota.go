package admin

import (
	"net/http"

	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

// quotaDTO mirrors the dashboard UI Quota shape. It carries NO credential
// fields — quota is an aggregation over per-connection usage and never echoes a
// secret or token.
type quotaDTO struct {
	ConnectionID   string  `json:"connection_id"`
	Provider       string  `json:"provider"`
	ConnectionName string  `json:"connection_name"`
	AccountLabel   string  `json:"account_label,omitempty"`
	Plan           string  `json:"plan"`
	Used           float64 `json:"used"`
	Limit          float64 `json:"limit"`
	Unit           string  `json:"unit"`
	ResetAt        string  `json:"reset_at"`
	IsActive       bool    `json:"is_active"`
}

// QuotaHandler serves GET /api/quota by aggregating per-connection usage. The
// fetcher seam is injectable so tests drive it with a deterministic fake (no
// network), mirroring ConnectionUsageHandler.
type QuotaHandler struct {
	Handlers *Handlers
	// HTTPClient is the client used for provider API calls. nil means
	// http.DefaultClient.
	HTTPClient *http.Client
	// Fetcher loads usage data for a provider connection. nil means
	// usage.FetchProviderUsage.
	Fetcher func(providerType string, conn *store.Connection, client *http.Client, baseURL ...string) (map[string]any, error)
}

// quotaNum coerces a usage-map value into a float64, tolerating the int/float64
// variants JSON decoding and provider mappers may produce.
func quotaNum(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func quotaStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetQuota handles GET /api/quota. It enumerates all connections, resolves the
// provider type/name, and for oauth connections maps the provider usage map into
// the Quota shape via the injectable fetcher. Non-oauth connections contribute a
// card with zeroed usage. The response carries no credentials.
func (h *QuotaHandler) GetQuota(ctx *fasthttp.RequestCtx) {
	conns, err := h.Handlers.store.ListConnections()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list connections")
		return
	}

	client := h.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	fetcher := h.Fetcher
	if fetcher == nil {
		fetcher = usage.FetchProviderUsage
	}

	out := make([]quotaDTO, 0, len(conns))
	for _, conn := range conns {
		card := quotaDTO{
			ConnectionID:   conn.ID,
			ConnectionName: conn.Name,
			Unit:           "tokens",
			IsActive:       true,
		}

		provider, perr := h.Handlers.store.GetProvider(conn.ProviderID)
		if perr == nil && provider != nil {
			card.Provider = provider.Type
		}

		if conn.Kind == "oauth" && provider != nil {
			// Use the connection's own credentials; never echo them.
			fetchConn := *conn
			if usageData, ferr := fetcher(provider.Type, &fetchConn, client); ferr == nil && usageData != nil {
				card.Used = quotaNum(usageData["used"])
				card.Limit = quotaNum(usageData["limit"])
				if u := quotaStr(usageData["unit"]); u != "" {
					card.Unit = u
				}
				card.Plan = quotaStr(usageData["plan"])
				card.AccountLabel = quotaStr(usageData["account_label"])
				card.ResetAt = quotaStr(usageData["reset_at"])
			}
		}

		out = append(out, card)
	}

	writeData(ctx, fasthttp.StatusOK, out)
}
