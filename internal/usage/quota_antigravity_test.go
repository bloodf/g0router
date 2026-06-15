package usage

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAntigravityUsageFetcher(t *testing.T) {
	t.Run("project id from metadata maps buckets to snake_case quotas", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/v1internal:retrieveUserQuota" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			if auth := r.Header.Get("Authorization"); auth != "Bearer token-ag" {
				t.Fatalf("authorization = %q", auth)
			}
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("unmarshal body: %v", err)
			}
			if req["project"] != "proj-ag" {
				t.Fatalf("project = %v", req["project"])
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"buckets": []map[string]any{
					{"modelId": "gemini-2.5-pro", "remainingFraction": 0.4, "resetTime": 1234567890},
				},
			})
		}))
		t.Cleanup(srv.Close)

		got, err := fetchAntigravityUsage("token-ag", `{"projectId":"proj-ag"}`, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		quotas, ok := got["quotas"].(map[string]any)
		if !ok {
			t.Fatalf("quotas type = %T", got["quotas"])
		}
		q, ok := quotas["gemini-2.5-pro"].(map[string]any)
		if !ok {
			t.Fatalf("missing model quota: %v", quotas)
		}
		if q["used"] != float64(600) || q["total"] != float64(1000) {
			t.Fatalf("quota = %v", q)
		}
		if q["remaining_percentage"] != float64(40) {
			t.Fatalf("remaining_percentage = %v", q["remaining_percentage"])
		}
		if q["reset_at"] != "2009-02-13T23:31:30Z" {
			t.Fatalf("reset_at = %v", q["reset_at"])
		}
		if _, ok := q["remainingPercentage"]; ok {
			t.Fatal("quota still emits camelCase remainingPercentage")
		}
		if _, ok := q["resetAt"]; ok {
			t.Fatal("quota still emits camelCase resetAt")
		}
		assertNoTokenLeak(t, got, "token-ag")
	})

	t.Run("missing project id returns plan+message without a network call", func(t *testing.T) {
		called := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		t.Cleanup(srv.Close)

		got, err := fetchAntigravityUsage("token-ag", "", srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if called {
			t.Fatal("quota endpoint should not be called when project id is absent")
		}
		if got["plan"] == nil {
			t.Fatalf("expected a plan field, got %v", got)
		}
		if _, ok := got["message"].(string); !ok {
			t.Fatalf("expected a message field, got %v", got)
		}
	})

	t.Run("missing access token returns plan+message without a network call", func(t *testing.T) {
		called := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		t.Cleanup(srv.Close)

		got, err := fetchAntigravityUsage("", `{"projectId":"proj-ag"}`, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if called {
			t.Fatal("quota endpoint should not be called when access token is absent")
		}
		if _, ok := got["message"].(string); !ok {
			t.Fatalf("expected a message field, got %v", got)
		}
	})

	t.Run("non-2xx returns a graceful message", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		t.Cleanup(srv.Close)

		got, err := fetchAntigravityUsage("token-ag", `{"projectId":"proj-ag"}`, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if _, ok := got["message"].(string); !ok {
			t.Fatalf("expected a message field on non-2xx, got %v", got)
		}
		assertNoTokenLeak(t, got, "token-ag")
	})
}

// assertNoTokenLeak marshals the fetcher result and fails if the canned token
// value appears anywhere in the JSON (secret-safety: tokens are used transiently
// in the request, never echoed into the returned map).
func assertNoTokenLeak(t *testing.T, result map[string]any, token string) {
	t.Helper()
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(b), token) {
		t.Fatalf("token leaked into returned map: %s", string(b))
	}
}
