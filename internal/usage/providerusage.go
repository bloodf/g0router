package usage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

var numericRegexp = regexp.MustCompile(`^\d+$`)

const (
	defaultClaudeBaseURL = "https://api.anthropic.com"
	defaultGeminiBaseURL = "https://cloudcode-pa.googleapis.com"
	anthropicAPIVersion  = "2023-06-01"

	// w7-usage-quota: base URL for the antigravity Cloud-Code quota call. The
	// host is the catalog antigravity BaseURL (catalog.go:100); the path is the
	// Cloud-Code retrieveUserQuota endpoint shared with the gemini twin.
	defaultAntigravityBaseURL = "https://daily-cloudcode-pa.googleapis.com"
)

// parseModelProvider splits a tracker model key into model and provider.
// Keys are produced by Tracker.modelKey: "model (provider)" when provider is
// present, otherwise just "model".
func parseModelProvider(key string) (model, provider string) {
	if !strings.HasSuffix(key, ")") {
		return key, ""
	}
	idx := strings.LastIndex(key, " (")
	if idx == -1 {
		return key, ""
	}
	return key[:idx], key[idx+2 : len(key)-1]
}

// StatsMap returns the Stats result for the given period as a JSON-shaped map.
// It is used by the SSE usage stream to send full stats frames.
func (s *StatsService) StatsMap(period string) (map[string]any, error) {
	stats, err := s.Stats(period)
	if err != nil {
		return nil, fmt.Errorf("stats %s: %w", period, err)
	}
	b, err := json.Marshal(stats)
	if err != nil {
		return nil, fmt.Errorf("marshal stats: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal stats: %w", err)
	}
	return m, nil
}

// StreamSnapshot returns the lightweight fields that are overlaid onto cached
// stats for quick and pending SSE frames.
func (s *StatsService) StreamSnapshot() (map[string]any, error) {
	if s.tracker == nil || s.ring == nil {
		return nil, fmt.Errorf("tracker or ring not available")
	}
	entries := s.ring.Snapshot()
	recent := make([]RecentRequest, 0, len(entries))
	for _, e := range entries {
		recent = append(recent, RecentRequest{
			Timestamp:        e.Timestamp,
			Model:            e.Model,
			Provider:         e.Provider,
			PromptTokens:     e.PromptTokens,
			CompletionTokens: e.CompletionTokens,
			Status:           e.Status,
		})
	}

	active := make([]ActiveRequest, 0)
	s.tracker.mu.Lock()
	for connID, models := range s.tracker.byAccount {
		account := ""
		if s.names != nil {
			account = s.names.ConnectionName(connID)
		}
		if account == "" {
			account = accountFallback(connID)
		}
		for modelKey, count := range models {
			if count <= 0 {
				continue
			}
			model, provider := parseModelProvider(modelKey)
			active = append(active, ActiveRequest{
				Model:    model,
				Provider: provider,
				Account:  account,
				Count:    count,
			})
		}
	}
	s.tracker.mu.Unlock()

	return map[string]any{
		"active_requests": active,
		"recent_requests": DedupeRecent(recent),
		"error_provider":  s.tracker.LastErrorProvider(),
	}, nil
}

// StreamEvents exposes the tracker event emitter so the usage stream can
// subscribe to update/pending events.
func (s *StatsService) StreamEvents() *Events {
	return s.tracker.events
}

// OffEvent removes a previously registered callback. It is the counterpart to
// OnEvent and is used by the usage stream to avoid leaking subscriptions.
func (e *Events) OffEvent(fn func(kind string)) {
	target := reflect.ValueOf(fn).Pointer()
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, cb := range e.cbs {
		if reflect.ValueOf(cb).Pointer() == target {
			e.cbs = append(e.cbs[:i], e.cbs[i+1:]...)
			return
		}
	}
}

// FetchProviderUsage returns usage/quota data for a single provider connection.
// Stage-1 supports anthropic (Claude) and gemini; all other providers return a
// fallback message. The optional baseURL parameter is a test seam.
func FetchProviderUsage(providerType string, conn *store.Connection, client *http.Client, baseURL ...string) (map[string]any, error) {
	if client == nil {
		client = http.DefaultClient
	}
	switch providerType {
	case "anthropic":
		return fetchClaudeUsage(conn.AccessToken, client, firstBaseURL(baseURL, defaultClaudeBaseURL))
	case "gemini":
		return fetchGeminiUsage(conn.AccessToken, conn.Metadata, client, firstBaseURL(baseURL, defaultGeminiBaseURL))
	// --- w7-usage-quota: remaining 6 provider arms (additive, before default) ---
	// BUILT: antigravity is the gemini fetcher's Cloud-Code twin (sound in-tree
	// precedent + catalog-confirmed base), so it delegates to a real fetcher.
	case "antigravity":
		return fetchAntigravityUsage(conn.AccessToken, conn.Metadata, client, firstBaseURL(baseURL, defaultAntigravityBaseURL))
	// DEFERRED: the frozen 9router route.js is absent on this host (ESC-REF-ABSENT),
	// so the concrete usage endpoint/auth/shape for these providers cannot be
	// soundly confirmed. Rather than fabricate an endpoint, each returns a clear,
	// provider-named "not available" fallback (no network call). See
	// .planning/parity/plans/open-questions.md for the per-provider deferral notes.
	case "github":
		return map[string]any{
			"message": "Usage API not yet available for GitHub Copilot.",
		}, nil
	case "codex":
		return map[string]any{
			"message": "Usage API not yet available for Codex.",
		}, nil
	case "kiro":
		return map[string]any{
			"message": "Usage API not yet available for Kiro.",
		}, nil
	case "glm":
		return map[string]any{
			"message": "Usage API not yet available for GLM.",
		}, nil
	case "minimax":
		return map[string]any{
			"message": "Usage API not yet available for MiniMax.",
		}, nil
	default:
		return map[string]any{
			"message": fmt.Sprintf("Usage API not implemented for %s", providerType),
		}, nil
	}
}

func firstBaseURL(provided []string, fallback string) string {
	if len(provided) > 0 && provided[0] != "" {
		return provided[0]
	}
	return fallback
}

// fetchClaudeUsage mirrors the ref's getClaudeUsage: try the OAuth usage
// endpoint first, then fall back to the legacy settings/org endpoint chain.
func fetchClaudeUsage(accessToken string, client *http.Client, baseURL string) (map[string]any, error) {
	if baseURL == "" {
		baseURL = defaultClaudeBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	primaryURL := baseURL + "/api/oauth/usage"
	req, err := http.NewRequest(http.MethodGet, primaryURL, nil)
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
	}
	setClaudeHeaders(req, accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
		}
		quotas := map[string]any{}
		if fh, ok := data["five_hour"].(map[string]any); ok && hasUtilization(fh) {
			quotas["session (5h)"] = createClaudeQuotaObject(fh)
		}
		if sd, ok := data["seven_day"].(map[string]any); ok && hasUtilization(sd) {
			quotas["weekly (7d)"] = createClaudeQuotaObject(sd)
		}
		for key, value := range data {
			if !strings.HasPrefix(key, "seven_day_") || key == "seven_day" {
				continue
			}
			if window, ok := value.(map[string]any); ok && hasUtilization(window) {
				model := strings.TrimPrefix(key, "seven_day_")
				quotas[fmt.Sprintf("weekly %s (7d)", model)] = createClaudeQuotaObject(window)
			}
		}
		return map[string]any{
			"plan":        "Claude Code",
			"extra_usage": data["extra_usage"],
			"quotas":      quotas,
		}, nil
	}

	// Fallback: legacy settings + org usage endpoint.
	settingsURL := baseURL + "/v1/settings"
	settingsReq, err := http.NewRequest(http.MethodGet, settingsURL, nil)
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
	}
	setClaudeHeaders(settingsReq, accessToken)

	settingsResp, err := client.Do(settingsReq)
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
	}
	defer settingsResp.Body.Close()

	if settingsResp.StatusCode != http.StatusOK {
		return map[string]any{"message": "Claude connected. Usage API requires admin permissions."}, nil
	}

	var settings map[string]any
	if err := json.NewDecoder(settingsResp.Body).Decode(&settings); err != nil {
		return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
	}

	orgID, _ := settings["organization_id"].(string)
	plan, _ := settings["plan"].(string)
	orgName, _ := settings["organization_name"].(string)
	if plan == "" {
		plan = "Unknown"
	}

	if orgID != "" {
		usageURL := fmt.Sprintf("%s/v1/organizations/%s/usage", baseURL, orgID)
		usageReq, err := http.NewRequest(http.MethodGet, usageURL, nil)
		if err != nil {
			return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
		}
		setClaudeHeaders(usageReq, accessToken)

		usageResp, err := client.Do(usageReq)
		if err != nil {
			return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
		}
		defer usageResp.Body.Close()

		if usageResp.StatusCode == http.StatusOK {
			var usageData map[string]any
			if err := json.NewDecoder(usageResp.Body).Decode(&usageData); err != nil {
				return map[string]any{"message": fmt.Sprintf("Claude connected. Unable to fetch usage: %v", err)}, nil
			}
			return map[string]any{
				"plan":         plan,
				"organization": orgName,
				"quotas":       usageData,
			}, nil
		}
	}

	return map[string]any{
		"plan":         plan,
		"organization": orgName,
		"message":      "Claude connected. Usage details require admin access.",
	}, nil
}

func setClaudeHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	if strings.Contains(req.URL.Path, "/api/oauth/usage") {
		req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	}
}

func hasUtilization(window map[string]any) bool {
	_, ok := window["utilization"].(float64)
	return ok
}

func createClaudeQuotaObject(window map[string]any) map[string]any {
	used := window["utilization"].(float64)
	remaining := math.Max(0, 100-used)
	return map[string]any{
		"used":                 used,
		"total":                float64(100),
		"remaining":            remaining,
		"remaining_percentage": remaining,
		"reset_at":             parseResetTime(window["resets_at"]),
		"unlimited":            false,
	}
}

// fetchGeminiUsage mirrors the ref's getGeminiUsage: resolve a Cloud project id
// (prefer connection metadata, else loadCodeAssist) and call retrieveUserQuota.
func fetchGeminiUsage(accessToken, metadata string, client *http.Client, baseURL string) (map[string]any, error) {
	if baseURL == "" {
		baseURL = defaultGeminiBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if accessToken == "" {
		return map[string]any{
			"plan":    "Free",
			"message": "Gemini CLI access token not available.",
		}, nil
	}

	projectID := projectIDFromMetadata(metadata)
	plan := "Free"

	if projectID == "" {
		sub, err := fetchGeminiSubscriptionInfo(accessToken, client, baseURL)
		if err == nil && sub != nil {
			projectID = sub.projectID
			if sub.plan != "" {
				plan = sub.plan
			}
		}
	}

	if projectID == "" {
		return map[string]any{
			"plan":    plan,
			"message": "Gemini CLI project ID not available. Reconnect Gemini CLI, or configure a Google Cloud project with Gemini Code Assist access before checking quota.",
		}, nil
	}

	quotaURL := baseURL + "/v1internal:retrieveUserQuota"
	body, _ := json.Marshal(map[string]any{"project": projectID})
	req, err := http.NewRequest(http.MethodPost, quotaURL, bytes.NewReader(body))
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Gemini CLI error: %v", err)}, nil
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Gemini CLI error: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return map[string]any{
			"plan":    plan,
			"message": fmt.Sprintf("Gemini CLI quota error (%d).", resp.StatusCode),
		}, nil
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return map[string]any{"message": fmt.Sprintf("Gemini CLI error: %v", err)}, nil
	}

	quotas := map[string]any{}
	buckets, _ := data["buckets"].([]any)
	for _, b := range buckets {
		bucket, ok := b.(map[string]any)
		if !ok {
			continue
		}
		modelID, _ := bucket["modelId"].(string)
		if modelID == "" {
			continue
		}
		frac, ok := bucket["remainingFraction"].(float64)
		if !ok {
			continue
		}
		total := float64(1000)
		remaining := math.Round(total * frac)
		used := math.Max(0, total-remaining)
		quotas[modelID] = map[string]any{
			"used":                 used,
			"total":                total,
			"reset_at":             parseResetTime(bucket["resetTime"]),
			"remaining_percentage": frac * 100,
			"unlimited":            false,
		}
	}

	return map[string]any{"plan": plan, "quotas": quotas}, nil
}

type geminiSubscription struct {
	projectID string
	plan      string
}

func fetchGeminiSubscriptionInfo(accessToken string, client *http.Client, baseURL string) (*geminiSubscription, error) {
	url := baseURL + "/v1internal:loadCodeAssist"
	body, _ := json.Marshal(map[string]any{"metadata": map[string]any{}})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini subscription info: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini subscription info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini subscription info: loadCodeAssist status %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("gemini subscription info: %w", err)
	}

	project := normalizeCloudCodeProjectID(data["cloudaicompanionProject"])
	plan := ""
	if tier, ok := data["currentTier"].(map[string]any); ok {
		plan, _ = tier["name"].(string)
	}
	if project == "" {
		return nil, fmt.Errorf("gemini subscription info: no project")
	}
	return &geminiSubscription{projectID: project, plan: plan}, nil
}

func projectIDFromMetadata(metadata string) string {
	if metadata == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(metadata), &m); err != nil {
		return ""
	}
	return normalizeCloudCodeProjectID(m["projectId"])
}

func normalizeCloudCodeProjectID(v any) string {
	switch p := v.(type) {
	case string:
		return strings.TrimSpace(p)
	case map[string]any:
		if id, ok := p["id"].(string); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func parseResetTime(v any) any {
	if v == nil {
		return nil
	}
	switch r := v.(type) {
	case float64:
		ts := int64(r)
		if r < 1e12 {
			ts = ts * 1000
		}
		return time.UnixMilli(ts).UTC().Format(time.RFC3339)
	case string:
		if strings.TrimSpace(r) == "" {
			return nil
		}
		if numericRegexp.MatchString(r) {
			n, _ := strconv.ParseInt(r, 10, 64)
			if n < 1e12 {
				n = n * 1000
			}
			return time.UnixMilli(n).UTC().Format(time.RFC3339)
		}
		if t, err := time.Parse(time.RFC3339, r); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
		if t, err := time.Parse(time.RFC1123, r); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return nil
}
