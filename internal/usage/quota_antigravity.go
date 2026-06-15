package usage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
)

// fetchAntigravityUsage mirrors the SHIPPED gemini fetcher (its structural twin):
// Antigravity is a Google Cloud-Code provider (catalog base
// https://daily-cloudcode-pa.googleapis.com, format "antigravity"), so quota is
// retrieved via the Cloud-Code retrieveUserQuota endpoint with a {project} body.
// The project id is read from the connection metadata (mirroring the gemini
// projectIDFromMetadata precedent). Tokens are used transiently in the request
// and never echoed into the returned map. Degraded-but-connected states return a
// {plan,message} map rather than a hard error (the gemini precedent).
func fetchAntigravityUsage(accessToken, metadata string, client *http.Client, baseURL string) (map[string]any, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = defaultAntigravityBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	plan := "Free"

	if accessToken == "" {
		return map[string]any{
			"plan":    plan,
			"message": "Antigravity access token not available.",
		}, nil
	}

	projectID := projectIDFromMetadata(metadata)
	if projectID == "" {
		return map[string]any{
			"plan":    plan,
			"message": "Antigravity project ID not available. Reconnect Antigravity before checking quota.",
		}, nil
	}

	quotaURL := baseURL + "/v1internal:retrieveUserQuota"
	body, _ := json.Marshal(map[string]any{"project": projectID})
	req, err := http.NewRequest(http.MethodPost, quotaURL, bytes.NewReader(body))
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Antigravity error: %v", err)}, nil
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return map[string]any{"message": fmt.Sprintf("Antigravity error: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return map[string]any{
			"plan":    plan,
			"message": fmt.Sprintf("Antigravity quota error (%d).", resp.StatusCode),
		}, nil
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return map[string]any{"message": fmt.Sprintf("Antigravity error: %v", err)}, nil
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
