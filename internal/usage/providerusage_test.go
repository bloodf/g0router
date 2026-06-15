package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

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
				"five_hour":        map[string]any{"utilization": 12, "resets_at": "2026-06-13T00:00:00Z"},
				"seven_day":        map[string]any{"utilization": 45},
				"seven_day_sonnet": map[string]any{"utilization": 80},
				"extra_usage":      map[string]any{"foo": "bar"},
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
		if session["remaining_percentage"] != float64(88) {
			t.Fatalf("session remaining_percentage = %v", session["remaining_percentage"])
		}
		if session["reset_at"] != "2026-06-13T00:00:00Z" {
			t.Fatalf("session reset_at = %v", session["reset_at"])
		}
		if _, ok := session["remainingPercentage"]; ok {
			t.Fatal("session quota still emits camelCase remainingPercentage")
		}
		if _, ok := session["resetAt"]; ok {
			t.Fatal("session quota still emits camelCase resetAt")
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
		if q["remaining_percentage"] != float64(75) {
			t.Fatalf("quota remaining_percentage = %v", q["remaining_percentage"])
		}
		if q["reset_at"] != "2009-02-13T23:31:30Z" {
			t.Fatalf("quota reset_at = %v", q["reset_at"])
		}
		if _, ok := q["remainingPercentage"]; ok {
			t.Fatal("quota still emits camelCase remainingPercentage")
		}
		if _, ok := q["resetAt"]; ok {
			t.Fatal("quota still emits camelCase resetAt")
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

// TestProviderUsageDispatch asserts each of the 6 w7-usage-quota provider types
// routes away from the generic "default" catch-all arm. BUILT providers reach
// their own fetcher (exercised deeply in the per-provider test file); DEFERRED
// providers return a provider-named fallback message distinct from the generic
// "Usage API not implemented for <provider>" default. No real network: the only
// network-touching arm (antigravity, BUILT) is pointed at an httptest server.
func TestProviderUsageDispatch(t *testing.T) {
	genericFor := func(p string) string {
		return fmt.Sprintf("Usage API not implemented for %s", p)
	}

	t.Run("deferred providers return a provider-named fallback", func(t *testing.T) {
		cases := []struct {
			providerType string
			wantContains string
		}{
			{"github", "GitHub"},
			{"codex", "Codex"},
			{"kiro", "Kiro"},
			{"glm", "GLM"},
			{"minimax", "MiniMax"},
		}
		for _, tc := range cases {
			t.Run(tc.providerType, func(t *testing.T) {
				conn := &store.Connection{}
				got, err := FetchProviderUsage(tc.providerType, conn, http.DefaultClient)
				if err != nil {
					t.Fatalf("fetch: %v", err)
				}
				msg, _ := got["message"].(string)
				if msg == genericFor(tc.providerType) {
					t.Fatalf("%s still falls through to the generic default arm: %q", tc.providerType, msg)
				}
				if !strings.Contains(msg, tc.wantContains) {
					t.Fatalf("%s fallback message = %q, want it to mention %q", tc.providerType, msg, tc.wantContains)
				}
			})
		}
	})

	t.Run("antigravity routes to its built fetcher", func(t *testing.T) {
		hit := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit = true
			if r.URL.Path != "/v1internal:retrieveUserQuota" {
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(map[string]any{"buckets": []map[string]any{}})
		}))
		t.Cleanup(srv.Close)

		conn := &store.Connection{
			AccessToken: "token-1",
			Metadata:    `{"projectId":"proj-ag"}`,
		}
		got, err := FetchProviderUsage("antigravity", conn, srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if !hit {
			t.Fatal("antigravity did not reach its built fetcher (no HTTP call made)")
		}
		if msg, _ := got["message"].(string); msg == genericFor("antigravity") {
			t.Fatalf("antigravity fell through to the generic default arm: %q", msg)
		}
	})
}


func noOpTimerFactory(time.Duration, func()) func() { return func() {} }

func TestStreamSnapshotActiveRequests(t *testing.T) {
	events := NewEvents()
	tracker := NewTracker(func() time.Time { return time.Now() }, noOpTimerFactory, events)
	ring := NewRing(10)
	if err := ring.Init(func() ([]*store.RequestLogEntry, error) { return nil, nil }); err != nil {
		t.Fatalf("ring init: %v", err)
	}
	names := &fakeNameSource{conn: map[string]string{"conn-known": "Known Account"}}
	svc := NewStatsService(&fakeUsageReader{}, names, tracker, ring, func() time.Time { return time.Now() })

	tracker.Start("claude-3-5-sonnet", "anthropic", "conn-known")
	tracker.Start("claude-3-opus", "anthropic", "conn-anon")

	snap, err := svc.StreamSnapshot()
	if err != nil {
		t.Fatalf("StreamSnapshot: %v", err)
	}

	active, ok := snap["active_requests"].([]ActiveRequest)
	if !ok {
		t.Fatalf("active_requests type = %T", snap["active_requests"])
	}
	if len(active) != 2 {
		t.Fatalf("active_requests len = %d, want 2: %v", len(active), active)
	}

	sort.Slice(active, func(i, j int) bool { return active[i].Model < active[j].Model })

	wantAnon := accountFallback("conn-anon")
	if active[0].Model != "claude-3-5-sonnet" || active[0].Provider != "anthropic" || active[0].Account != "Known Account" || active[0].Count != 1 {
		t.Fatalf("active[0] = %+v", active[0])
	}
	if active[1].Model != "claude-3-opus" || active[1].Provider != "anthropic" || active[1].Account != wantAnon || active[1].Count != 1 {
		t.Fatalf("active[1] = %+v", active[1])
	}
}


func TestFetchGeminiSubscriptionInfoWrapsErrors(t *testing.T) {
	t.Run("bad status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		t.Cleanup(srv.Close)

		_, err := fetchGeminiSubscriptionInfo("token", srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "gemini subscription info:") {
			t.Fatalf("error not wrapped: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not-json"))
		}))
		t.Cleanup(srv.Close)

		_, err := fetchGeminiSubscriptionInfo("token", srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "gemini subscription info:") {
			t.Fatalf("error not wrapped: %v", err)
		}
	})

	t.Run("network error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()

		_, err := fetchGeminiSubscriptionInfo("token", srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "gemini subscription info:") {
			t.Fatalf("error not wrapped: %v", err)
		}
	})
}
