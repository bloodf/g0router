package api

import (
	"errors"
	"sync"
	"testing"
)

// fakeVKResolver resolves keys to canned VKInfo values.
type fakeVKResolver struct {
	mu   sync.Mutex
	keys map[string]*VKInfo
}

func newFakeVKResolver() *fakeVKResolver {
	return &fakeVKResolver{keys: map[string]*VKInfo{}}
}

func (r *fakeVKResolver) set(key string, vk *VKInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[key] = vk
}

func (r *fakeVKResolver) ResolveVK(key string) (*VKInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	vk, ok := r.keys[key]
	if !ok {
		return nil, nil
	}
	return vk, nil
}

// fakeVKQuotaChecker returns a fixed result for the first call and optionally
// records inputs.
type fakeVKQuotaChecker struct {
	mu       sync.Mutex
	results  []struct {
		ok     bool
		status int
		reason string
	}
	calls []struct {
		vk    *VKInfo
		model string
	}
	idx int
}

func newFakeVKQuotaChecker(results ...struct {
	ok     bool
	status int
	reason string
}) *fakeVKQuotaChecker {
	return &fakeVKQuotaChecker{results: results}
}

func (q *fakeVKQuotaChecker) Allow(vk *VKInfo, model string) (bool, int, string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.calls = append(q.calls, struct {
		vk    *VKInfo
		model string
	}{vk: vk, model: model})
	if q.idx >= len(q.results) {
		return true, 0, ""
	}
	r := q.results[q.idx]
	q.idx++
	return r.ok, r.status, r.reason
}

// errVKResolver always returns an error.
type errVKResolver struct{}

func (errVKResolver) ResolveVK(string) (*VKInfo, error) {
	return nil, errors.New("lookup failed")
}

// TestVKGateUnknownKeyDenied verifies Fix 1: a non-empty x-g0-vk header that
// resolves to nil is denied with 401 and the provider/quota layer is never called.
func TestVKGateUnknownKeyDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	quota := newFakeVKQuotaChecker()
	gate := NewVKGate(resolver, quota)

	ok, status, reason := gate.AllowVK("g0vk-does-not-exist", "gpt-4")
	if ok {
		t.Fatalf("AllowVK unknown key: got ok=true, want false")
	}
	if status != 401 {
		t.Errorf("AllowVK unknown key status = %d, want 401", status)
	}
	if reason != "unknown virtual key" {
		t.Errorf("AllowVK unknown key reason = %q, want %q", reason, "unknown virtual key")
	}
	if quota.calls != nil {
		t.Errorf("quota checker called %d times, want 0", len(quota.calls))
	}
}
