package ratelimit

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func newTestLimiter() (*Limiter, *fakeClock) {
	clk := &fakeClock{t: time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)}
	return NewLimiterWithClock(clk.now), clk
}

func TestAllowRequestNilUnlimited(t *testing.T) {
	l, _ := newTestLimiter()
	for i := 0; i < 1000; i++ {
		if !l.AllowRequest("k", nil) {
			t.Fatal("nil rpm must always allow")
		}
	}
}

func TestAllowRequestRPMBucketAndRefill(t *testing.T) {
	l, clk := newTestLimiter()
	rpm := intPtr(3)
	for i := 0; i < 3; i++ {
		if !l.AllowRequest("k", rpm) {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if l.AllowRequest("k", rpm) {
		t.Fatal("4th request should be denied")
	}
	// Different key is independent.
	if !l.AllowRequest("other", rpm) {
		t.Fatal("other key should be allowed")
	}
	// Advance under a minute: still denied.
	clk.advance(30 * time.Second)
	if l.AllowRequest("k", rpm) {
		t.Fatal("still within window, should be denied")
	}
	// Advance past the minute: refilled.
	clk.advance(31 * time.Second)
	if !l.AllowRequest("k", rpm) {
		t.Fatal("after refill should be allowed")
	}
}

func TestTokensRollingWindow(t *testing.T) {
	l, clk := newTestLimiter()
	tpm := intPtr(100)
	if !l.AllowTokens("k", tpm) {
		t.Fatal("empty window allowed")
	}
	l.AddTokens("k", 80)
	if !l.AllowTokens("k", tpm) {
		t.Fatal("80<100 allowed")
	}
	l.AddTokens("k", 30)
	if l.AllowTokens("k", tpm) {
		t.Fatal("110>=100 denied")
	}
	clk.advance(61 * time.Second)
	if !l.AllowTokens("k", tpm) {
		t.Fatal("window expired, allowed")
	}
	if !l.AllowTokens("k", nil) {
		t.Fatal("nil tpm unlimited")
	}
}

func TestSpendAccumulateAndDailyReset(t *testing.T) {
	l, clk := newTestLimiter()
	l.AddSpend("k", 1.50)
	l.AddSpend("k", 2.00)
	if got := l.SpendToday("k"); got != 3.50 {
		t.Fatalf("spend = %v, want 3.50", got)
	}
	// Advance to next UTC day.
	clk.advance(24 * time.Hour)
	if got := l.SpendToday("k"); got != 0 {
		t.Fatalf("spend after reset = %v, want 0", got)
	}
	l.AddSpend("k", 5.0)
	if got := l.SpendToday("k"); got != 5.0 {
		t.Fatalf("spend new day = %v, want 5.0", got)
	}
}

func TestWithinSpendCap(t *testing.T) {
	l, _ := newTestLimiter()
	cap := floatPtr(10.0)
	if !l.WithinSpendCap("k", nil) {
		t.Fatal("nil cap uncapped")
	}
	l.AddSpend("k", 9.99)
	if !l.WithinSpendCap("k", cap) {
		t.Fatal("9.99<10 within cap")
	}
	l.AddSpend("k", 0.02)
	if l.WithinSpendCap("k", cap) {
		t.Fatal("10.01>=10 over cap")
	}
}

func TestSeedSpend(t *testing.T) {
	l, _ := newTestLimiter()
	l.SeedSpend("k", 7.5)
	if got := l.SpendToday("k"); got != 7.5 {
		t.Fatalf("seeded spend = %v", got)
	}
	l.AddSpend("k", 2.5)
	if got := l.SpendToday("k"); got != 10.0 {
		t.Fatalf("spend after add = %v", got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	l, _ := newTestLimiter()
	rpm := intPtr(1000000)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				l.AllowRequest("k", rpm)
				l.AddTokens("k", 1)
				l.AllowTokens("k", rpm)
				l.AddSpend("k", 0.001)
				l.SpendToday("k")
			}
		}()
	}
	wg.Wait()
}
