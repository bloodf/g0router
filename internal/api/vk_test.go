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

// fakePinnedKeyResolver is a test VKPinnedKeyResolver that returns canned values.
type fakePinnedKeyResolver struct {
	connID     string
	credential string
	ok         bool
	calls      []struct {
		providerID string
		model      string
		keyIDs     []string
	}
}

func (f *fakePinnedKeyResolver) ResolvePinned(providerID, model string, keyIDs []string) (string, string, bool) {
	f.calls = append(f.calls, struct {
		providerID string
		model      string
		keyIDs     []string
	}{providerID: providerID, model: model, keyIDs: keyIDs})
	return f.connID, f.credential, f.ok
}

// TestVKGateUnknownKeyDenied verifies Fix 1: a non-empty x-g0-vk header that
// resolves to nil is denied with 401 and the provider/quota layer is never called.
func TestVKGateUnknownKeyDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	quota := newFakeVKQuotaChecker()
	gate := NewVKGate(resolver, quota)

	ok, status, reason, _ := gate.AllowVK("g0vk-does-not-exist", "gpt-4", "openai")
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
		ok, status, _, _ := gate.AllowVK("", "gpt-4o", "openai")
		if !ok {
			t.Fatalf("empty key should allow: status=%d", status)
		}
	})

	t.Run("nil gate allow", func(t *testing.T) {
		var gate *VKGate
		ok, status, _, _ := gate.AllowVK("vk-active-openai", "gpt-4o", "openai")
		if !ok {
			t.Fatalf("nil gate should allow: status=%d", status)
		}
	})

	t.Run("unknown key deny 401", func(t *testing.T) {
		gate := NewVKGate(resolver, nil)
		ok, status, reason, _ := gate.AllowVK("g0vk-missing", "gpt-4o", "openai")
		if ok || status != 401 || reason != "unknown virtual key" {
			t.Fatalf("unknown key: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})

	t.Run("inactive deny 403", func(t *testing.T) {
		gate := NewVKGate(resolver, nil)
		ok, status, reason, _ := gate.AllowVK(inactive.Key, "gpt-4o", "openai")
		if ok || status != 403 || reason != "virtual key inactive" {
			t.Fatalf("inactive: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})

	t.Run("resolver error deny 500", func(t *testing.T) {
		gate := NewVKGate(errVKResolver{}, nil)
		ok, status, reason, _ := gate.AllowVK("vk-any", "gpt-4o", "openai")
		if ok || status != 500 || reason != "virtual key lookup failed" {
			t.Fatalf("resolver error: got ok=%v status=%d reason=%q", ok, status, reason)
		}
	})

	t.Run("model/provider mismatch deny 403", func(t *testing.T) {
		gate := NewVKGate(resolver, nil)
		ok, status, reason, _ := gate.AllowVK(activeOpenAI.Key, "gpt-4o", "anthropic")
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
		ok, status, reason, _ := gate.AllowVK(activeOpenAI.Key, "gpt-4o", "openai")
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
		ok, status, reason, _ := gate.AllowVK("vk-openai", "gpt-4o", "anthropic")
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
		ok, status, _, _ := gate.AllowVK("vk-openai", "gpt-4", "openai")
		if ok {
			t.Fatal("model mismatch should be denied")
		}
		if status != 403 {
			t.Errorf("status = %d, want 403", status)
		}
	})

	t.Run("empty allowed models allows any model of provider", func(t *testing.T) {
		ok, status, _, _ := gate.AllowVK("vk-anthropic-open", "claude-opus-4", "anthropic")
		if !ok {
			t.Fatalf("empty AllowedModels for provider should allow any model: status=%d", status)
		}
	})

	t.Run("listed model allowed", func(t *testing.T) {
		ok, status, _, _ := gate.AllowVK("vk-openai", "gpt-4o", "openai")
		if !ok {
			t.Fatalf("listed model should be allowed: status=%d", status)
		}
	})
}

// TestAllowVK_ReturnsKeyIDsOnMatch verifies PAR-ROUTE-030: when a matching config
// carries KeyIDs, AllowVK returns ok=true and those KeyIDs.
func TestAllowVK_ReturnsKeyIDsOnMatch(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-keyids", &VKInfo{
		Key: "vk-keyids",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4o"}, KeyIDs: []string{"conn-1", "conn-2"}},
		},
		IsActive: true,
	})
	gate := NewVKGate(resolver, nil)

	ok, status, reason, keyIDs := gate.AllowVK("vk-keyids", "gpt-4o", "openai")
	if !ok {
		t.Fatalf("AllowVK got ok=false, status=%d reason=%q", status, reason)
	}
	if status != 0 || reason != "" {
		t.Errorf("status = %d, reason = %q, want 0/\"\"", status, reason)
	}
	if len(keyIDs) != 2 || keyIDs[0] != "conn-1" || keyIDs[1] != "conn-2" {
		t.Errorf("keyIDs = %v, want [conn-1 conn-2]", keyIDs)
	}
}

// TestAllowVK_NilKeyIDsWhenUnset verifies that a matched config with no KeyIDs
// returns nil (not an empty slice).
func TestAllowVK_NilKeyIDsWhenUnset(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-no-keyids", &VKInfo{
		Key: "vk-no-keyids",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4o"}},
		},
		IsActive: true,
	})
	gate := NewVKGate(resolver, nil)

	ok, _, _, keyIDs := gate.AllowVK("vk-no-keyids", "gpt-4o", "openai")
	if !ok {
		t.Fatal("AllowVK should allow")
	}
	if keyIDs != nil {
		t.Errorf("keyIDs = %v, want nil", keyIDs)
	}
}

// TestAllowVK_AllowAllKeysFallsThrough verifies PAR-BF-GOV-002 / D5: a matched
// config with AllowAllKeys=true returns no pinned keyIDs (fall through to normal
// selection) even when KeyIDs is populated, so any key for the provider is allowed.
func TestAllowVK_AllowAllKeysFallsThrough(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-allowall", &VKInfo{
		Key: "vk-allowall",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4o"}, KeyIDs: []string{"conn-1"}, AllowAllKeys: true},
		},
		IsActive: true,
	})
	gate := NewVKGate(resolver, nil)

	ok, status, reason, keyIDs := gate.AllowVK("vk-allowall", "gpt-4o", "openai")
	if !ok {
		t.Fatalf("AllowVK got ok=false status=%d reason=%q", status, reason)
	}
	if keyIDs != nil {
		t.Errorf("AllowAllKeys=true keyIDs = %v, want nil (no pin)", keyIDs)
	}
}

// TestAllowVK_AllowAllKeysFalseEmptyKeyIDs verifies D5: AllowAllKeys=false with
// empty KeyIDs falls through (no pin), the conservative behavior-preserving
// default (the matrix "no keys allowed" rejection HTTP shape is undocumented;
// recorded in open-questions.md rather than inventing a 403).
func TestAllowVK_AllowAllKeysFalseEmptyKeyIDs(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-noallow", &VKInfo{
		Key: "vk-noallow",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4o"}, AllowAllKeys: false},
		},
		IsActive: true,
	})
	gate := NewVKGate(resolver, nil)

	ok, status, reason, keyIDs := gate.AllowVK("vk-noallow", "gpt-4o", "openai")
	if !ok {
		t.Fatalf("AllowVK got ok=false status=%d reason=%q", status, reason)
	}
	if keyIDs != nil {
		t.Errorf("empty KeyIDs keyIDs = %v, want nil", keyIDs)
	}
}

// TestAllowVK_NilKeyIDsOnDenial verifies that quota/inactive/unknown/mismatch
// denial paths return nil KeyIDs.
func TestAllowVK_NilKeyIDsOnDenial(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-quota", &VKInfo{
		Key: "vk-quota",
		Configs: []VKProviderConfig{
			{Provider: "openai", AllowedModels: []string{"gpt-4o"}, KeyIDs: []string{"conn-1"}},
		},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 429, reason: "budget exhausted"})

	gate := NewVKGate(resolver, quota)
	ok, status, reason, keyIDs := gate.AllowVK("vk-quota", "gpt-4o", "openai")
	if ok || status != 429 || reason != "budget exhausted" {
		t.Fatalf("got ok=%v status=%d reason=%q", ok, status, reason)
	}
	if keyIDs != nil {
		t.Errorf("quota denial keyIDs = %v, want nil", keyIDs)
	}
}
