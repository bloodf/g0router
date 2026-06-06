package provider

import (
	"testing"
	"time"
)

func TestRealClockNow(t *testing.T) {
	c := realClock{}
	got := c.Now()
	if got.IsZero() {
		t.Fatal("expected non-zero time")
	}
	if time.Since(got) > time.Second {
		t.Fatal("expected time close to now")
	}
}
