package auth

import (
	"strings"
	"sync"
	"time"
)

const (
	maxFailsBeforeLock = 5
	failWindow         = time.Hour
)

var lockSteps = []time.Duration{
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
}

type entry struct {
	fails      int
	lockUntil  time.Time
	lockLevel  int
	lastFailAt time.Time
}

// LoginLimiter is an in-memory progressive lockout for dashboard login.
// It resets on process restart.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*entry
	now      func() time.Time
}

// NewLoginLimiter creates a new limiter with a real clock.
func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{
		attempts: make(map[string]*entry),
		now:      time.Now,
	}
}

func (l *LoginLimiter) getEntry(ip string) *entry {
	e, ok := l.attempts[ip]
	if !ok {
		return nil
	}
	now := l.now()
	// Auto reset if window expired and not currently locked.
	if !e.lastFailAt.IsZero() && now.Sub(e.lastFailAt) > failWindow && (e.lockUntil.IsZero() || now.After(e.lockUntil) || now.Equal(e.lockUntil)) {
		delete(l.attempts, ip)
		return nil
	}
	return e
}

// CheckLock returns whether the IP is locked and the retry-after seconds (ceil).
func (l *LoginLimiter) CheckLock(ip string) (locked bool, retryAfter int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	e := l.getEntry(ip)
	if e == nil || e.lockUntil.IsZero() {
		return false, 0
	}
	remaining := e.lockUntil.Sub(l.now())
	if remaining <= 0 {
		return false, 0
	}
	return true, int(remaining.Seconds() + 0.999999) // ceil
}

// RecordFail increments the fail count for the IP and returns remaining
// attempts before lockout.
func (l *LoginLimiter) RecordFail(ip string) (remainingBeforeLock int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	e := l.getEntry(ip)
	if e == nil {
		e = &entry{fails: 0, lockLevel: 0}
	}
	e.fails++
	e.lastFailAt = l.now()
	if e.fails >= maxFailsBeforeLock {
		idx := e.lockLevel
		if idx >= len(lockSteps) {
			idx = len(lockSteps) - 1
		}
		step := lockSteps[idx]
		e.lockUntil = l.now().Add(step)
		e.lockLevel++
		e.fails = 0
	}
	l.attempts[ip] = e
	return maxFailsBeforeLock - e.fails
}

// RecordSuccess clears any state for the IP.
func (l *LoginLimiter) RecordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}

// ClientIP extracts the client IP from forwarded headers.
// It returns the first entry of x-forwarded-for, else x-real-ip, else "unknown".
func ClientIP(xff, xRealIP string) string {
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xRealIP != "" {
		return xRealIP
	}
	return "unknown"
}
