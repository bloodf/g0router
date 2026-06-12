package api

import (
	"errors"
	"sync"
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
