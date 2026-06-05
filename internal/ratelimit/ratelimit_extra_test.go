package ratelimit

import (
	"testing"
	"time"
)

// TestNewLimiter exercises the real-clock constructor (0.0% covered).
func TestNewLimiter(t *testing.T) {
	l := NewLimiter()
	if l == nil {
		t.Fatal("NewLimiter returned nil")
	}
	// Confirm it works end-to-end with real clock.
	rpm := intPtr(1)
	if !l.AllowRequest("k", rpm) {
		t.Fatal("first request must be allowed")
	}
}

// TestAllowRequestZeroRPMUnlimited covers the *rpm <= 0 branch.
func TestAllowRequestZeroRPMUnlimited(t *testing.T) {
	l, _ := newTestLimiter()
	zero := intPtr(0)
	neg := intPtr(-5)
	for i := 0; i < 100; i++ {
		if !l.AllowRequest("k", zero) {
			t.Fatal("zero rpm must always allow")
		}
		if !l.AllowRequest("k", neg) {
			t.Fatal("negative rpm must always allow")
		}
	}
}

// TestAddTokensZeroAndNegativeNoop covers the tokens <= 0 early-return.
func TestAddTokensZeroAndNegativeNoop(t *testing.T) {
	l, _ := newTestLimiter()
	l.AddTokens("k", 0)
	l.AddTokens("k", -10)
	tpm := intPtr(1)
	// Window was never created; AllowTokens must still pass (nil window = allowed).
	if !l.AllowTokens("k", tpm) {
		t.Fatal("no tokens added, must be within tpm=1")
	}
}

// TestAllowTokensZeroTPMUnlimited covers the *tpm <= 0 branch.
func TestAllowTokensZeroTPMUnlimited(t *testing.T) {
	l, _ := newTestLimiter()
	l.AddTokens("k", 9999)
	zero := intPtr(0)
	neg := intPtr(-1)
	if !l.AllowTokens("k", zero) {
		t.Fatal("zero tpm must always allow")
	}
	if !l.AllowTokens("k", neg) {
		t.Fatal("negative tpm must always allow")
	}
}

// TestAllowTokensWindowExpiry covers the "window expired → reset" branch inside AllowTokens.
func TestAllowTokensWindowExpiry(t *testing.T) {
	l, clk := newTestLimiter()
	tpm := intPtr(5)
	l.AddTokens("k", 10) // over the limit
	if l.AllowTokens("k", tpm) {
		t.Fatal("10 tokens >= tpm=5 must be denied")
	}
	clk.advance(61 * time.Second)
	if !l.AllowTokens("k", tpm) {
		t.Fatal("after window expiry must be allowed again")
	}
}

// TestAddSpendZeroAndNegativeNoop covers the usd <= 0 early-return.
func TestAddSpendZeroAndNegativeNoop(t *testing.T) {
	l, _ := newTestLimiter()
	l.AddSpend("k", 0)
	l.AddSpend("k", -1.0)
	if got := l.SpendToday("k"); got != 0 {
		t.Fatalf("spend = %v after no-op adds, want 0", got)
	}
}

// TestWithinSpendCapNegativeCap covers the *cap < 0 unlimited branch.
func TestWithinSpendCapNegativeCap(t *testing.T) {
	l, _ := newTestLimiter()
	l.AddSpend("k", 1e9)
	neg := floatPtr(-1.0)
	if !l.WithinSpendCap("k", neg) {
		t.Fatal("negative cap must always be within cap")
	}
}

// TestWithinSpendCapZero covers *cap == 0 (any spend exceeds it).
func TestWithinSpendCapZero(t *testing.T) {
	l, _ := newTestLimiter()
	zero := floatPtr(0.0)
	// No spend yet: 0 < 0 is false → over cap.
	if l.WithinSpendCap("k", zero) {
		t.Fatal("zero cap with zero spend: 0 < 0 = false, must be blocked")
	}
}

// TestSpendTodayDifferentDay covers the "wrong day" branch in SpendToday.
func TestSpendTodayDifferentDay(t *testing.T) {
	l, clk := newTestLimiter()
	l.AddSpend("k", 3.0)
	clk.advance(25 * time.Hour)
	if got := l.SpendToday("k"); got != 0 {
		t.Fatalf("SpendToday after day change = %v, want 0", got)
	}
}

// TestAddSpendDayRollover covers the "new day" window-creation branch inside AddSpend.
func TestAddSpendDayRollover(t *testing.T) {
	l, clk := newTestLimiter()
	l.AddSpend("k", 5.0)
	clk.advance(25 * time.Hour) // roll the day
	l.AddSpend("k", 2.0)        // creates a fresh window
	if got := l.SpendToday("k"); got != 2.0 {
		t.Fatalf("spend after rollover = %v, want 2.0", got)
	}
}
