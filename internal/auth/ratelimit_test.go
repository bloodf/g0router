package auth

import (
	"testing"
	"time"
)

func TestLoginRateLimiter_AllowedInitially(t *testing.T) {
	l := NewLoginRateLimiter()
	allowed, retryAfter := l.Check("1.2.3.4")
	if !allowed {
		t.Fatalf("expected allowed for fresh IP")
	}
	if retryAfter != 0 {
		t.Fatalf("expected retryAfter=0, got %d", retryAfter)
	}
}

func TestLoginRateLimiter_BlockedAfterMaxFailures(t *testing.T) {
	l := NewLoginRateLimiter()
	ip := "1.2.3.4"

	for i := 0; i < maxFailures; i++ {
		l.RecordFailure(ip)
	}

	allowed, retryAfter := l.Check(ip)
	if allowed {
		t.Fatalf("expected blocked after %d failures", maxFailures)
	}
	if retryAfter <= 0 {
		t.Fatalf("expected retryAfter > 0, got %d", retryAfter)
	}
}

func TestLoginRateLimiter_WindowResets(t *testing.T) {
	l := NewLoginRateLimiter()
	ip := "1.2.3.4"

	// Manually inject an expired window
	l.mu.Lock()
	l.attempts[ip] = &loginAttemptWindow{
		failures:    maxFailures,
		windowStart: time.Now().UTC().Add(-windowDuration - time.Second),
	}
	l.mu.Unlock()

	allowed, retryAfter := l.Check(ip)
	if !allowed {
		t.Fatalf("expected allowed after window expiry")
	}
	if retryAfter != 0 {
		t.Fatalf("expected retryAfter=0 after window expiry, got %d", retryAfter)
	}
}

func TestLoginRateLimiter_RecordFailureResetsExpiredWindow(t *testing.T) {
	l := NewLoginRateLimiter()
	ip := "1.2.3.4"

	// Inject expired window
	l.mu.Lock()
	l.attempts[ip] = &loginAttemptWindow{
		failures:    maxFailures,
		windowStart: time.Now().UTC().Add(-windowDuration - time.Second),
	}
	l.mu.Unlock()

	l.RecordFailure(ip)

	allowed, _ := l.Check(ip)
	if !allowed {
		t.Fatalf("expected allowed after RecordFailure resets expired window")
	}
}

func TestLoginRateLimiter_DifferentIPsIndependent(t *testing.T) {
	l := NewLoginRateLimiter()

	for i := 0; i < maxFailures; i++ {
		l.RecordFailure("1.2.3.4")
	}

	allowed, _ := l.Check("5.6.7.8")
	if !allowed {
		t.Fatalf("expected different IP to be allowed")
	}
}
