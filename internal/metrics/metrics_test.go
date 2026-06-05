package metrics

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// sampleValue extracts the value of a single Prometheus sample line matching
// the given metric name and (optional) label substring. It returns the raw
// value string and whether a matching line was found.
func sampleValue(t *testing.T, rendered, metric, labelSubstr string) (string, bool) {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if !strings.HasPrefix(line, metric) {
			continue
		}
		if labelSubstr != "" && !strings.Contains(line, labelSubstr) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		return fields[len(fields)-1], true
	}
	return "", false
}

func TestObserveRequestRendersSamples(t *testing.T) {
	c := NewCollector()
	c.ObserveRequest("anthropic", "claude-3", "2xx", 100, 50, 0.25, 1500*time.Millisecond)
	c.ObserveRequest("anthropic", "claude-3", "2xx", 10, 5, 0.05, 200*time.Millisecond)

	out := c.Render()

	if !strings.Contains(out, "# TYPE requests_total counter") {
		t.Fatalf("missing TYPE line for requests_total:\n%s", out)
	}
	if !strings.Contains(out, "# HELP requests_total") {
		t.Fatalf("missing HELP line for requests_total")
	}

	if v, ok := sampleValue(t, out, "requests_total", `provider="anthropic"`); !ok || v != "2" {
		t.Fatalf("requests_total = %q (found=%v), want 2", v, ok)
	}
	if v, ok := sampleValue(t, out, `tokens_total{type="input"}`, ""); !ok || v != "110" {
		t.Fatalf("tokens_total input = %q (found=%v), want 110", v, ok)
	}
	if v, ok := sampleValue(t, out, `tokens_total{type="output"}`, ""); !ok || v != "55" {
		t.Fatalf("tokens_total output = %q (found=%v), want 55", v, ok)
	}
	if v, ok := sampleValue(t, out, "cost_usd_total", ""); !ok {
		t.Fatalf("cost_usd_total missing")
	} else if !strings.HasPrefix(v, "0.3") {
		t.Fatalf("cost_usd_total = %q, want ~0.3", v)
	}
	if v, ok := sampleValue(t, out, "request_duration_seconds_count", ""); !ok || v != "2" {
		t.Fatalf("request_duration_seconds_count = %q (found=%v), want 2", v, ok)
	}
}

func TestStatusClassLabelDistinct(t *testing.T) {
	c := NewCollector()
	c.ObserveRequest("openai", "gpt", "2xx", 1, 1, 0, time.Second)
	c.ObserveRequest("openai", "gpt", "5xx", 1, 1, 0, time.Second)

	out := c.Render()
	if v, ok := sampleValue(t, out, "requests_total", `status_class="2xx"`); !ok || v != "1" {
		t.Fatalf("2xx requests_total = %q, want 1", v)
	}
	if v, ok := sampleValue(t, out, "requests_total", `status_class="5xx"`); !ok || v != "1" {
		t.Fatalf("5xx requests_total = %q, want 1", v)
	}
}

func TestIncCounters(t *testing.T) {
	c := NewCollector()
	c.IncRefreshFailure()
	c.IncRefreshFailure()
	c.IncRateLimitRejected()
	c.IncSpendCapRejected()
	c.IncSpendCapRejected()
	c.IncSpendCapRejected()

	out := c.Render()
	if v, ok := sampleValue(t, out, "oauth_refresh_failures_total", ""); !ok || v != "2" {
		t.Fatalf("oauth_refresh_failures_total = %q, want 2", v)
	}
	if v, ok := sampleValue(t, out, "rate_limit_rejected_total", ""); !ok || v != "1" {
		t.Fatalf("rate_limit_rejected_total = %q, want 1", v)
	}
	if v, ok := sampleValue(t, out, "spend_cap_rejected_total", ""); !ok || v != "3" {
		t.Fatalf("spend_cap_rejected_total = %q, want 3", v)
	}
}

func TestRenderZeroValueCountersPresent(t *testing.T) {
	c := NewCollector()
	out := c.Render()
	// Single-series counters should render at 0 even before any observation so
	// scrapers see a stable series.
	for _, m := range []string{
		"oauth_refresh_failures_total 0",
		"rate_limit_rejected_total 0",
		"spend_cap_rejected_total 0",
		"cost_usd_total 0",
	} {
		if !strings.Contains(out, m) {
			t.Fatalf("missing zero series %q in:\n%s", m, out)
		}
	}
}

func TestConcurrentObserve(t *testing.T) {
	c := NewCollector()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.ObserveRequest("p", "m", "2xx", 1, 1, 0.01, time.Millisecond)
			c.IncRateLimitRejected()
		}()
	}
	wg.Wait()

	out := c.Render()
	if v, ok := sampleValue(t, out, "requests_total", `provider="p"`); !ok || v != "50" {
		t.Fatalf("requests_total = %q, want 50", v)
	}
	if v, ok := sampleValue(t, out, "rate_limit_rejected_total", ""); !ok || v != "50" {
		t.Fatalf("rate_limit_rejected_total = %q, want 50", v)
	}
}
