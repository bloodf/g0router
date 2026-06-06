package auth

import (
	"sync"
	"time"
)

const maxFailures = 5
const windowDuration = 15 * time.Minute

// LoginRateLimiter tracks failed login attempts per IP with a sliding window.
type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttemptWindow
}

type loginAttemptWindow struct {
	failures    int
	windowStart time.Time
}

// NewLoginRateLimiter creates a new login rate limiter.
func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts: make(map[string]*loginAttemptWindow),
	}
}

// RecordFailure increments the failure count for the given IP. If the window
// has expired, it resets to 1.
func (r *LoginRateLimiter) RecordFailure(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	w, ok := r.attempts[ip]
	if !ok || now.Sub(w.windowStart) >= windowDuration {
		r.attempts[ip] = &loginAttemptWindow{
			failures:    1,
			windowStart: now,
		}
		return
	}
	w.failures++
}

// Check returns whether the IP is allowed to attempt a login. If failures
// have reached maxFailures within the current window, allowed is false and
// retryAfterSeconds is the remaining time until the window resets.
func (r *LoginRateLimiter) Check(ip string) (allowed bool, retryAfterSeconds int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.attempts[ip]
	if !ok {
		return true, 0
	}

	now := time.Now().UTC()
	if now.Sub(w.windowStart) >= windowDuration {
		delete(r.attempts, ip)
		return true, 0
	}

	if w.failures >= maxFailures {
		remaining := windowDuration - now.Sub(w.windowStart)
		return false, int(remaining.Seconds())
	}

	return true, 0
}
