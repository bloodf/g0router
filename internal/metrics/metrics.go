// Package metrics provides a thread-safe in-process collector that renders
// Prometheus text exposition format without external dependencies.
package metrics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// durationBuckets are the upper bounds (in seconds) for the request duration
// histogram, mirroring Prometheus client defaults closely enough for a gateway.
var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// requestKey identifies a requests_total series by its label set.
type requestKey struct {
	provider    string
	model       string
	statusClass string
}

// Collector accumulates gateway metrics and renders them in Prometheus text
// format. All methods are safe for concurrent use.
type Collector struct {
	mu sync.Mutex

	requests map[requestKey]uint64

	inputTokens  uint64
	outputTokens uint64
	costUSD      float64

	durationBucketCounts []uint64
	durationCount        uint64
	durationSum          float64

	refreshFailures   uint64
	rateLimitRejected uint64
	spendCapRejected  uint64
}

// NewCollector returns an empty Collector ready for use.
func NewCollector() *Collector {
	return &Collector{
		requests:             make(map[requestKey]uint64),
		durationBucketCounts: make([]uint64, len(durationBuckets)),
	}
}

// ObserveRequest records a completed inference request: it increments the
// per-(provider,model,status_class) request counter, adds token and cost
// totals, and observes the request duration in the histogram.
func (c *Collector) ObserveRequest(provider, model, statusClass string, inputTok, outputTok int, costUSD float64, dur time.Duration) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests[requestKey{provider: provider, model: model, statusClass: statusClass}]++
	if inputTok > 0 {
		c.inputTokens += uint64(inputTok)
	}
	if outputTok > 0 {
		c.outputTokens += uint64(outputTok)
	}
	if costUSD > 0 {
		c.costUSD += costUSD
	}

	seconds := dur.Seconds()
	for i, bound := range durationBuckets {
		if seconds <= bound {
			c.durationBucketCounts[i]++
		}
	}
	c.durationCount++
	c.durationSum += seconds
}

// IncRefreshFailure increments the OAuth refresh failure counter.
func (c *Collector) IncRefreshFailure() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.refreshFailures++
	c.mu.Unlock()
}

// IncRateLimitRejected increments the rate-limit rejection counter.
func (c *Collector) IncRateLimitRejected() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.rateLimitRejected++
	c.mu.Unlock()
}

// IncSpendCapRejected increments the spend-cap rejection counter.
func (c *Collector) IncSpendCapRejected() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.spendCapRejected++
	c.mu.Unlock()
}

// Render produces the Prometheus text exposition format (version=0.0.4) for all
// collected metrics. The output is deterministic: series are emitted in a
// stable, sorted order.
func (c *Collector) Render() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	var b strings.Builder

	b.WriteString("# HELP requests_total Total inference requests by provider, model and status class.\n")
	b.WriteString("# TYPE requests_total counter\n")
	for _, key := range sortedRequestKeys(c.requests) {
		labels := fmt.Sprintf(`{provider=%s,model=%s,status_class=%s}`,
			quote(key.provider), quote(key.model), quote(key.statusClass))
		fmt.Fprintf(&b, "requests_total%s %d\n", labels, c.requests[key])
	}

	b.WriteString("# HELP tokens_total Total tokens processed by type.\n")
	b.WriteString("# TYPE tokens_total counter\n")
	fmt.Fprintf(&b, "tokens_total{type=\"input\"} %d\n", c.inputTokens)
	fmt.Fprintf(&b, "tokens_total{type=\"output\"} %d\n", c.outputTokens)

	b.WriteString("# HELP cost_usd_total Total inference cost in USD.\n")
	b.WriteString("# TYPE cost_usd_total counter\n")
	fmt.Fprintf(&b, "cost_usd_total %s\n", formatFloat(c.costUSD))

	b.WriteString("# HELP request_duration_seconds Inference request duration in seconds.\n")
	b.WriteString("# TYPE request_duration_seconds histogram\n")
	var cumulative uint64
	for i, bound := range durationBuckets {
		cumulative = c.durationBucketCounts[i]
		fmt.Fprintf(&b, "request_duration_seconds_bucket{le=%s} %d\n", quote(formatFloat(bound)), cumulative)
	}
	fmt.Fprintf(&b, "request_duration_seconds_bucket{le=\"+Inf\"} %d\n", c.durationCount)
	fmt.Fprintf(&b, "request_duration_seconds_sum %s\n", formatFloat(c.durationSum))
	fmt.Fprintf(&b, "request_duration_seconds_count %d\n", c.durationCount)

	writeSingleCounter(&b, "oauth_refresh_failures_total", "Total OAuth connection refresh failures.", c.refreshFailures)
	writeSingleCounter(&b, "rate_limit_rejected_total", "Total requests rejected by rate limiting.", c.rateLimitRejected)
	writeSingleCounter(&b, "spend_cap_rejected_total", "Total requests rejected by daily spend cap.", c.spendCapRejected)

	return b.String()
}

func writeSingleCounter(b *strings.Builder, name, help string, value uint64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)
	fmt.Fprintf(b, "%s %d\n", name, value)
}

func sortedRequestKeys(m map[requestKey]uint64) []requestKey {
	keys := make([]requestKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].provider != keys[j].provider {
			return keys[i].provider < keys[j].provider
		}
		if keys[i].model != keys[j].model {
			return keys[i].model < keys[j].model
		}
		return keys[i].statusClass < keys[j].statusClass
	})
	return keys
}

// quote renders a Prometheus label value with the required escaping.
func quote(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + replacer.Replace(value) + `"`
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
