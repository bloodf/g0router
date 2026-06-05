package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestNilCollectorGuards(t *testing.T) {
	var c *Collector
	// None of these should panic on a nil receiver.
	c.ObserveRequest("p", "m", "2xx", 1, 1, 0.1, time.Second)
	c.IncRefreshFailure()
	c.IncRateLimitRejected()
	c.IncSpendCapRejected()
	if got := c.Render(); got != "" {
		t.Fatalf("nil Render = %q, want empty", got)
	}
}

func TestSortedRequestKeysTieBreakers(t *testing.T) {
	c := NewCollector()
	// Same provider, different model exercises the model tie-breaker.
	c.ObserveRequest("openai", "gpt-4o", "2xx", 0, 0, 0, time.Millisecond)
	c.ObserveRequest("openai", "gpt-3.5", "2xx", 0, 0, 0, time.Millisecond)
	// Same provider+model, different status_class exercises the final tie-breaker.
	c.ObserveRequest("openai", "gpt-4o", "5xx", 0, 0, 0, time.Millisecond)

	out := c.Render()
	iA := strings.Index(out, `model="gpt-3.5"`)
	iB := strings.Index(out, `model="gpt-4o",status_class="2xx"`)
	iC := strings.Index(out, `model="gpt-4o",status_class="5xx"`)
	if iA < 0 || iB < 0 || iC < 0 {
		t.Fatalf("missing series in output:\n%s", out)
	}
	// gpt-3.5 sorts before gpt-4o; within gpt-4o, 2xx before 5xx.
	if !(iA < iB && iB < iC) {
		t.Fatalf("series not in sorted order: gpt-3.5=%d gpt-4o/2xx=%d gpt-4o/5xx=%d\n%s", iA, iB, iC, out)
	}
}
