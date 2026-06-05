package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func fixedClock(t *time.Time) func() time.Time {
	return func() time.Time { return *t }
}

func TestKeyStableAcrossReorderedFields(t *testing.T) {
	c := NewCache(10, time.Minute, time.Now)

	a := c.Key("gpt-4o", []byte(`{"messages":[{"role":"user","content":"hi"}],"model":"gpt-4o","temperature":0.7}`))
	b := c.Key("gpt-4o", []byte(`{"temperature":0.7,"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`))
	if a != b {
		t.Fatalf("reordered JSON produced different keys: %q vs %q", a, b)
	}
}

func TestKeyDiffersByModel(t *testing.T) {
	c := NewCache(10, time.Minute, time.Now)
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	if c.Key("gpt-4o", body) == c.Key("gpt-4o-mini", body) {
		t.Fatalf("different models produced identical keys")
	}
}

func TestKeyDiffersByBody(t *testing.T) {
	c := NewCache(10, time.Minute, time.Now)
	if c.Key("m", []byte(`{"a":1}`)) == c.Key("m", []byte(`{"a":2}`)) {
		t.Fatalf("different bodies produced identical keys")
	}
}

func TestKeyHandlesInvalidJSON(t *testing.T) {
	c := NewCache(10, time.Minute, time.Now)
	// Non-JSON bodies must still hash deterministically and stably.
	a := c.Key("m", []byte("not json"))
	b := c.Key("m", []byte("not json"))
	if a != b || a == "" {
		t.Fatalf("invalid JSON key unstable or empty: %q vs %q", a, b)
	}
}

func TestGetMiss(t *testing.T) {
	c := NewCache(10, time.Minute, time.Now)
	if _, ok := c.Get("absent"); ok {
		t.Fatalf("expected miss for absent key")
	}
}

func TestSetGetHit(t *testing.T) {
	c := NewCache(10, time.Minute, time.Now)
	c.Set("k", []byte("value"))
	got, ok := c.Get("k")
	if !ok {
		t.Fatalf("expected hit after Set")
	}
	if string(got) != "value" {
		t.Fatalf("got %q, want value", got)
	}
}

func TestExpiry(t *testing.T) {
	now := time.Unix(0, 0)
	c := NewCache(10, 30*time.Second, fixedClock(&now))
	c.Set("k", []byte("v"))

	now = now.Add(29 * time.Second)
	if _, ok := c.Get("k"); !ok {
		t.Fatalf("entry should still be fresh at 29s")
	}

	now = now.Add(2 * time.Second) // 31s total
	if _, ok := c.Get("k"); ok {
		t.Fatalf("entry should be expired at 31s")
	}
}

func TestLRUEviction(t *testing.T) {
	c := NewCache(2, time.Minute, time.Now)
	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	// Touch "a" so "b" becomes least-recently-used.
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("a should be present")
	}
	c.Set("c", []byte("3")) // evicts "b"

	if _, ok := c.Get("b"); ok {
		t.Fatalf("b should have been evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("a should survive")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatalf("c should be present")
	}
}

func TestSetUpdatesExisting(t *testing.T) {
	c := NewCache(2, time.Minute, time.Now)
	c.Set("a", []byte("1"))
	c.Set("a", []byte("2"))
	got, ok := c.Get("a")
	if !ok || string(got) != "2" {
		t.Fatalf("update failed: ok=%v got=%q", ok, got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewCache(64, time.Minute, time.Now)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", n%8)
			c.Set(key, []byte(fmt.Sprintf("v%d", n)))
			c.Get(key)
			c.Key("m", []byte(`{"a":1,"b":2}`))
		}(i)
	}
	wg.Wait()
}
