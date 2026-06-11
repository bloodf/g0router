package auth

import (
	"sync"
	"testing"
	"time"
)

func newTestLimiter(t *testing.T) (*LoginLimiter, func() time.Time) {
	t.Helper()
	now := time.Now()
	clock := func() time.Time { return now }
	l := NewLoginLimiter()
	l.now = clock
	return l, func() time.Time { return now }
}

func TestLimiterLocksAfterFiveFails(t *testing.T) {
	now := time.Now()
	l := NewLoginLimiter()
	l.now = func() time.Time { return now }

	ip := "1.2.3.4"
	for i := 0; i < 4; i++ {
		locked, _ := l.CheckLock(ip)
		if locked {
			t.Fatalf("locked at attempt %d", i+1)
		}
		remaining := l.RecordFail(ip)
		if remaining != 4-i {
			t.Fatalf("attempt %d: remaining = %d, want %d", i+1, remaining, 4-i)
		}
	}

	// 5th fail triggers lock.
	locked, _ := l.CheckLock(ip)
	if locked {
		t.Fatal("locked before 5th fail recorded")
	}
	remaining := l.RecordFail(ip)
	if remaining != maxFailsBeforeLock {
		t.Fatalf("5th fail remaining = %d, want %d", remaining, maxFailsBeforeLock)
	}

	locked, retryAfter := l.CheckLock(ip)
	if !locked {
		t.Fatal("not locked after 5 fails")
	}
	if retryAfter != 30 {
		t.Fatalf("retryAfter = %d, want 30", retryAfter)
	}
}

func TestLimiterProgressiveSteps(t *testing.T) {
	now := time.Now()
	l := NewLoginLimiter()
	l.now = func() time.Time { return now }

	ip := "1.2.3.4"
	steps := []int{30, 120, 600, 1800}

	for level, wantStep := range steps {
		// Trigger lock by recording 5 fails.
		for i := 0; i < 5; i++ {
			l.RecordFail(ip)
		}

		locked, retryAfter := l.CheckLock(ip)
		if !locked {
			t.Fatalf("level %d: not locked", level)
		}
		if retryAfter != wantStep {
			t.Fatalf("level %d: retryAfter = %d, want %d", level, retryAfter, wantStep)
		}

		// Advance past lock to try next level.
		now = now.Add(time.Duration(wantStep) * time.Second).Add(time.Second)
	}

	// Fifth cycle (level 4) should still use the last step (30m).
	for i := 0; i < 5; i++ {
		l.RecordFail(ip)
	}
	locked, retryAfter := l.CheckLock(ip)
	if !locked {
		t.Fatal("level 4: not locked")
	}
	if retryAfter != 1800 {
		t.Fatalf("level 4: retryAfter = %d, want 1800", retryAfter)
	}
}

func TestLimiterAutoResetAfterWindow(t *testing.T) {
	now := time.Now()
	l := NewLoginLimiter()
	l.now = func() time.Time { return now }

	ip := "1.2.3.4"
	l.RecordFail(ip)

	// Advance past the 1h fail window but not locked.
	now = now.Add(61 * time.Minute)
	locked, _ := l.CheckLock(ip)
	if locked {
		t.Fatal("unexpectedly locked")
	}
	// Entry should be gone; recording a fresh fail should show 4 remaining.
	remaining := l.RecordFail(ip)
	if remaining != 4 {
		t.Fatalf("after window reset: remaining = %d, want 4", remaining)
	}
}

func TestLimiterNoResetWhileLocked(t *testing.T) {
	now := time.Now()
	l := NewLoginLimiter()
	l.now = func() time.Time { return now }

	ip := "1.2.3.4"
	// Manually create an entry where the window expired but lock is still active.
	l.attempts[ip] = &entry{
		fails:      0,
		lockUntil:  now.Add(30 * time.Minute),
		lockLevel:  1,
		lastFailAt: now.Add(-2 * time.Hour),
	}

	locked, retryAfter := l.CheckLock(ip)
	if !locked {
		t.Fatal("should still be locked")
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter = %d, want > 0", retryAfter)
	}

	// Recording 5 more fails should progress to the next lock level (120s).
	for i := 0; i < 5; i++ {
		l.RecordFail(ip)
	}
	locked2, retryAfter2 := l.CheckLock(ip)
	if !locked2 {
		t.Fatal("should still be locked after fail during lock")
	}
	if retryAfter2 != 120 {
		t.Fatalf("retryAfter = %d, want 120 (second lock step)", retryAfter2)
	}
}

func TestLimiterRecordSuccessClears(t *testing.T) {
	now := time.Now()
	l := NewLoginLimiter()
	l.now = func() time.Time { return now }

	ip := "1.2.3.4"
	l.RecordFail(ip)
	l.RecordSuccess(ip)

	locked, _ := l.CheckLock(ip)
	if locked {
		t.Fatal("locked after success")
	}
	remaining := l.RecordFail(ip)
	if remaining != 4 {
		t.Fatalf("after clear: remaining = %d, want 4", remaining)
	}
}

func TestClientIPExtraction(t *testing.T) {
	tests := []struct {
		name    string
		xff     string
		xRealIP string
		want    string
	}{
		{"xff first entry", "  10.0.0.1 , 10.0.0.2 ", "", "10.0.0.1"},
		{"xff single", "192.168.1.1", "", "192.168.1.1"},
		{"x-real-ip fallback", "", "192.168.1.2", "192.168.1.2"},
		{"unknown", "", "", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClientIP(tt.xff, tt.xRealIP)
			if got != tt.want {
				t.Fatalf("ClientIP(%q, %q) = %q, want %q", tt.xff, tt.xRealIP, got, tt.want)
			}
		})
	}
}

func TestLimiterConcurrent(t *testing.T) {
	now := time.Now()
	l := NewLoginLimiter()
	l.now = func() time.Time { return now }

	ip := "1.2.3.4"
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.CheckLock(ip)
			l.RecordFail(ip)
			l.RecordSuccess(ip)
		}()
	}
	wg.Wait()
}
