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

	ok, status, reason := gate.AllowVK("g0vk-does-not-exist", "gpt-4", "openai")
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

// TestVKGateAllow is the direct table-driven gate test (Fix 4): it pins every
// major AllowVK branch without going through a handler.
func TestVKGateAllow(t *testing.T) {
	activeOpenAI := &VKInfo{
		Key: "vk-active-openai",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4o"}},
		},
		IsActive: true,
	}
	inactive := &VKInfo{
		Key: "vk-inactive",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{}},
		},
		IsActive: false,
	}
	resolver := newFakeVKResolver()
	resolver.set(activeOpenAI.Key, activeOpenAI)
	resolver.set(inactive.Key, inactive)

	t.Run("absent header allow", func(t *testing.T) {
		gate := NewVKGate(resolver, nil)
		ok, status, _ := gate.AllowVK("", "gpt-4o", "openai")
		if !ok {
			t.Fatalf("empty key should allow: status=%d", status)
		}
	})

	t.Run("nil gate allow", func(t *testing.T) {
		var gate *VKGate
		ok, status, _ := gate.AllowVK("vk-active-openai", "gpt-4o", "openai")
		if !ok {
			t.Fatalf("nil gate should allow: status=%d", status)
		}
	})

	t.Run("unknown key deny 401", func(t *testing.T) {
		gate := NewVKGate(resolver, nil)
		ok, status, reason := gate.AllowVK("g0vk-missing", "gpt-4o", "openai")
		if ok || status != 401 || reason != "unknown virtual key" {
			t.Fatalf("unknown key: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})

	t.Run("inactive deny 403", func(t *testing.T) {
		gate := NewVKGate(resolver, nil)
		ok, status, reason := gate.AllowVK(inactive.Key, "gpt-4o", "openai")
		if ok || status != 403 || reason != "virtual key inactive" {
			t.Fatalf("inactive: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})

	t.Run("resolver error deny 500", func(t *testing.T) {
		gate := NewVKGate(errVKResolver{}, nil)
		ok, status, reason := gate.AllowVK("vk-any", "gpt-4o", "openai")
		if ok || status != 500 || reason != "virtual key lookup failed" {
			t.Fatalf("resolver error: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})

	t.Run("model/provider mismatch deny 403", func(t *testing.T) {
		gate := NewVKGate(resolver, nil)
		ok, status, reason := gate.AllowVK(activeOpenAI.Key, "gpt-4o", "anthropic")
		if ok || status != 403 || reason != "provider/model not allowed for virtual key" {
			t.Fatalf("provider mismatch: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})

	t.Run("quota deny 429", func(t *testing.T) {
		quota := newFakeVKQuotaChecker(struct {
			ok     bool
			status int
			reason string
		}{ok: false, status: 429, reason: "budget exhausted"})
		gate := NewVKGate(resolver, quota)
		ok, status, reason := gate.AllowVK(activeOpenAI.Key, "gpt-4o", "openai")
		if ok || status != 429 || reason != "budget exhausted" {
			t.Fatalf("quota deny: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})
}

// TestVKGateProviderConfigEnforced verifies Fix 2: the gate checks the resolved
// provider against the VK's provider configs and the model against the config's
// AllowedModels (empty list means any model for that provider).
func TestVKGateProviderConfigEnforced(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-openai", &VKInfo{
		Key: "vk-openai",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4o"}},
		},
		IsActive: true,
	})
	resolver.set("vk-anthropic-open", &VKInfo{
		Key: "vk-anthropic-open",
		Configs: []VKProviderConfig{
			{Provider: "anthropic", AllowedModels: []string{}},
		},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker()
	gate := NewVKGate(resolver, quota)

	t.Run("provider mismatch denied", func(t *testing.T) {
		ok, status, reason := gate.AllowVK("vk-openai", "gpt-4o", "anthropic")
		if ok {
			t.Fatal("provider mismatch should be denied")
		}
		if status != 403 {
			t.Errorf("status = %d, want 403", status)
		}
		if reason != "provider/model not allowed for virtual key" {
			t.Errorf("reason = %q, want %q", reason, "provider/model not allowed for virtual key")
		}
	})

	t.Run("model mismatch denied", func(t *testing.T) {
		ok, status, _ := gate.AllowVK("vk-openai", "gpt-4", "openai")
		if ok {
			t.Fatal("model mismatch should be denied")
		}
		if status != 403 {
			t.Errorf("status = %d, want 403", status)
		}
	})

	t.Run("empty allowed models allows any model of provider", func(t *testing.T) {
		ok, status, _ := gate.AllowVK("vk-anthropic-open", "claude-opus-4", "anthropic")
		if !ok {
			t.Fatalf("empty AllowedModels for provider should allow any model: status=%d", status)
		}
	})

	t.Run("listed model allowed", func(t *testing.T) {
		ok, status, _ := gate.AllowVK("vk-openai", "gpt-4o", "openai")
		if !ok {
			t.Fatalf("listed model should be allowed: status=%d", status)
		}
	})
}
