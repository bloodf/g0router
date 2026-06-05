// Package ratelimit provides an in-memory, thread-safe per-API-key limiter for
// request-rate (RPM), token-rate (TPM), and daily spend caps.
//
// State lives only in this process: it is not persisted, so a restart resets
// all counters. Spend can be seeded on boot via SeedSpend. A clock function is
// injected so the rolling windows and daily reset are deterministic in tests.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter tracks per-key request, token, and spend usage. The zero value is not
// usable; construct with NewLimiter.
type Limiter struct {
	now func() time.Time

	mu      sync.Mutex
	buckets map[string]*rpmBucket
	tokens  map[string]*rollingWindow
	spend   map[string]*spendWindow
}

type rpmBucket struct {
	windowStart time.Time
	count       int
}

type rollingWindow struct {
	windowStart time.Time
	tokens      int
}

type spendWindow struct {
	day   time.Time // UTC midnight of the active day
	total float64
}

// NewLimiter returns a Limiter using the real wall clock.
func NewLimiter() *Limiter {
	return NewLimiterWithClock(func() time.Time { return time.Now() })
}

// NewLimiterWithClock returns a Limiter driven by the supplied clock. The clock
// must never return the zero time.
func NewLimiterWithClock(now func() time.Time) *Limiter {
	return &Limiter{
		now:     now,
		buckets: make(map[string]*rpmBucket),
		tokens:  make(map[string]*rollingWindow),
		spend:   make(map[string]*spendWindow),
	}
}

// AllowRequest reports whether a request may proceed under the RPM limit and, if
// so, consumes one slot. A nil or non-positive rpm means unlimited.
func (l *Limiter) AllowRequest(keyID string, rpm *int) bool {
	if rpm == nil || *rpm <= 0 {
		return true
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[keyID]
	if bucket == nil || now.Sub(bucket.windowStart) >= time.Minute {
		bucket = &rpmBucket{windowStart: now, count: 0}
		l.buckets[keyID] = bucket
	}
	if bucket.count >= *rpm {
		return false
	}
	bucket.count++
	return true
}

// AddTokens records tokens consumed by a key in the current rolling minute.
func (l *Limiter) AddTokens(keyID string, tokens int) {
	if tokens <= 0 {
		return
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.tokens[keyID]
	if window == nil || now.Sub(window.windowStart) >= time.Minute {
		window = &rollingWindow{windowStart: now, tokens: 0}
		l.tokens[keyID] = window
	}
	window.tokens += tokens
}

// AllowTokens reports whether the key is still under its TPM limit. A nil or
// non-positive tpm means unlimited. It does not consume tokens; record actual
// usage with AddTokens after a response is known.
func (l *Limiter) AllowTokens(keyID string, tpm *int) bool {
	if tpm == nil || *tpm <= 0 {
		return true
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.tokens[keyID]
	if window == nil || now.Sub(window.windowStart) >= time.Minute {
		return true
	}
	return window.tokens < *tpm
}

// AddSpend accumulates USD spend for the key on the current UTC day.
func (l *Limiter) AddSpend(keyID string, usd float64) {
	if usd <= 0 {
		return
	}
	day := utcMidnight(l.now())
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.spend[keyID]
	if window == nil || !window.day.Equal(day) {
		window = &spendWindow{day: day, total: 0}
		l.spend[keyID] = window
	}
	window.total += usd
}

// SeedSpend sets the key's accumulated spend for the current UTC day, replacing
// any existing value. Intended for boot-time seeding from persisted usage.
func (l *Limiter) SeedSpend(keyID string, usd float64) {
	day := utcMidnight(l.now())
	l.mu.Lock()
	defer l.mu.Unlock()
	l.spend[keyID] = &spendWindow{day: day, total: usd}
}

// SpendToday returns the key's accumulated spend for the current UTC day.
func (l *Limiter) SpendToday(keyID string) float64 {
	day := utcMidnight(l.now())
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.spend[keyID]
	if window == nil || !window.day.Equal(day) {
		return 0
	}
	return window.total
}

// WithinSpendCap reports whether the key is still under its daily spend cap. A
// nil or negative cap means uncapped.
func (l *Limiter) WithinSpendCap(keyID string, cap *float64) bool {
	if cap == nil || *cap < 0 {
		return true
	}
	return l.SpendToday(keyID) < *cap
}

func utcMidnight(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
