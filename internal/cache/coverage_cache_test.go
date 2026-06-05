package cache

import (
	"testing"
	"time"
)

func TestNewCacheNilNowDefaultsToTimeNow(t *testing.T) {
	c := NewCache(4, time.Minute, nil)
	c.Set("k", []byte("v"))
	got, hit := c.Get("k")
	if !hit || string(got) != "v" {
		t.Fatalf("Get = (%q,%v), want (v,true)", got, hit)
	}
}

func TestSetZeroCapacityNoOp(t *testing.T) {
	c := NewCache(0, time.Minute, time.Now)
	c.Set("k", []byte("v"))
	if _, hit := c.Get("k"); hit {
		t.Fatalf("expected no entry stored when maxEntries=0")
	}
}
