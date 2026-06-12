package usage

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

func TestClaudeUsageFetcher(t *testing.T) {
	t.Run("primary oauth endpoint", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/api/oauth/usage" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			if auth := r.Header.Get("Authorization"); auth != "Bearer token-1" {
				t.Fatalf("authorization = %q", auth)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"five_hour":          map[string]any{"utilization": 12, "resets_at": "2026-06-13T00:00:00Z"},
				"seven_day":          map[string]any{"utilization": 45},
				"seven_day_sonnet":   map[string]any{"utilization": 80},
				"extra_usage":        map[string]any{"foo": "bar"},
			})
		}))
		t.Cleanup(srv.Close)

		conn := &store.Connection{AccessToken: "token-1"}
		got, err := FetchProviderUsage("anthropic", conn, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}

		if got["plan"] != "Claude Code" {
			t.Fatalf("plan = %v", got["plan"])
		}
		extra, ok := got["extra_usage"].(map[string]any)
		if !ok || extra["foo"] != "bar" {
			t.Fatalf("extra_usage = %v", got["extra_usage"])
		}
		quotas, ok := got["quotas"].(map[string]any)
		if !ok {
			t.Fatalf("quotas type = %T", got["quotas"])
		}
		if len(quotas) != 3 {
			t.Fatalf("quotas = %v", quotas)
		}
		session := quotas["session (5h)"].(map[string]any)
		if session["used"] != float64(12) || session["total"] != float64(100) {
			t.Fatalf("session quota = %v", session)
		}
	})

	t.Run("fallback to legacy", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/oauth/usage":
				w.WriteHeader(http.StatusInternalServerError)
			case "/v1/settings":
				json.NewEncoder(w).Encode(map[string]any{
					"organization_id":   "org-1",
					"organization_name": "Acme",
					"plan":              "Pro",
				})
			case "/v1/organizations/org-1/usage":
				json.NewEncoder(w).Encode(map[string]any{"requests": 1})
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		t.Cleanup(srv.Close)

		conn := &store.Connection{AccessToken: "token-1"}
		got, err := FetchProviderUsage("anthropic", conn, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if got["plan"] != "Pro" {
			t.Fatalf("plan = %v", got["plan"])
		}
		if got["organization"] != "Acme" {
			t.Fatalf("organization = %v", got["organization"])
		}
	})

	t.Run("both endpoints fail", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		t.Cleanup(srv.Close)

		conn := &store.Connection{AccessToken: "token-1"}
		got, err := FetchProviderUsage("anthropic", conn, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		msg, _ := got["message"].(string)
		if !strings.Contains(msg, "Usage API requires admin permissions") {
			t.Fatalf("message = %q", msg)
		}
	})
}

func TestGeminiUsageFetcher(t *testing.T) {
	t.Run("project id from metadata", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("unmarshal body: %v", err)
			}
			switch r.URL.Path {
			case "/v1internal:loadCodeAssist":
				t.Fatal("loadCodeAssist should not be called when projectId is present")
			case "/v1internal:retrieveUserQuota":
				if req["project"] != "proj-123" {
					t.Fatalf("project = %v", req["project"])
				}
				json.NewEncoder(w).Encode(map[string]any{
					"buckets": []map[string]any{
						{"modelId": "gemini-1.5-pro", "remainingFraction": 0.75, "resetTime": 1234567890},
					},
				})
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		t.Cleanup(srv.Close)

		conn := &store.Connection{
			AccessToken: "token-1",
			Metadata:    `{"projectId":"proj-123"}`,
		}
		got, err := FetchProviderUsage("gemini", conn, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if got["plan"] != "Free" {
			t.Fatalf("plan = %v", got["plan"])
		}
		quotas := got["quotas"].(map[string]any)
		q := quotas["gemini-1.5-pro"].(map[string]any)
		if q["used"] != float64(250) || q["total"] != float64(1000) {
			t.Fatalf("quota = %v", q)
		}
	})

	t.Run("project id from subscription", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1internal:loadCodeAssist":
				json.NewEncoder(w).Encode(map[string]any{
					"cloudaicompanionProject": "proj-sub",
					"currentTier":             map[string]any{"name": "Pro"},
				})
			case "/v1internal:retrieveUserQuota":
				json.NewEncoder(w).Encode(map[string]any{
					"buckets": []map[string]any{
						{"modelId": "gemini-1.5-flash", "remainingFraction": 1, "resetTime": nil},
					},
				})
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		}))
		t.Cleanup(srv.Close)

		conn := &store.Connection{AccessToken: "token-1"}
		got, err := FetchProviderUsage("gemini", conn, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if got["plan"] != "Pro" {
			t.Fatalf("plan = %v", got["plan"])
		}
		quotas := got["quotas"].(map[string]any)
		q := quotas["gemini-1.5-flash"].(map[string]any)
		if q["used"] != float64(0) || q["total"] != float64(1000) {
			t.Fatalf("quota = %v", q)
		}
	})
}
