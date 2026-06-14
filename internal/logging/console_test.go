package logging

import (
	"testing"
	"time"
)

func TestConsoleLogRecentBounded(t *testing.T) {
	c := NewConsoleLog(3)
	c.Append("info", "one")
	c.Append("info", "two")
	c.Append("warn", "three")
	c.Append("error", "four")

	recent := c.Recent()
	if len(recent) != 3 {
		t.Fatalf("Recent len = %d, want 3 (bounded ring)", len(recent))
	}
	// Newest set, oldest dropped: two, three, four (in order).
	wantMsgs := []string{"two", "three", "four"}
	for i, want := range wantMsgs {
		if recent[i].Message != want {
			t.Fatalf("recent[%d].Message = %q, want %q", i, recent[i].Message, want)
		}
	}
	if recent[2].Level != "error" {
		t.Fatalf("recent[2].Level = %q, want %q", recent[2].Level, "error")
	}
	for _, line := range recent {
		if line.Timestamp == "" {
			t.Fatalf("line %q has empty Timestamp", line.Message)
		}
		if _, err := time.Parse(time.RFC3339, line.Timestamp); err != nil {
			t.Fatalf("line %q Timestamp %q not RFC3339: %v", line.Message, line.Timestamp, err)
		}
	}
}

func TestConsoleLogSubscribeReceivesFrame(t *testing.T) {
	c := NewConsoleLog(8)
	ch, unsub := c.Subscribe()
	defer unsub()

	c.Append("warn", "hello")

	select {
	case line := <-ch:
		if line.Level != "warn" {
			t.Fatalf("Level = %q, want %q", line.Level, "warn")
		}
		if line.Message != "hello" {
			t.Fatalf("Message = %q, want %q", line.Message, "hello")
		}
		if _, err := time.Parse(time.RFC3339, line.Timestamp); err != nil {
			t.Fatalf("Timestamp %q not RFC3339: %v", line.Timestamp, err)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive subscribed frame")
	}
}

func TestConsoleLogSlowConsumerDoesNotBlock(t *testing.T) {
	c := NewConsoleLog(64)
	// Subscribe but never drain: Append must not block once the buffer fills.
	_, unsub := c.Subscribe()
	defer unsub()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 10000; i++ {
			c.Append("info", "flood")
		}
		close(done)
	}()

	select {
	case <-done:
		// Append completed despite the slow (never-draining) consumer.
	case <-time.After(2 * time.Second):
		t.Fatal("Append blocked on a slow consumer (frames must drop, not block)")
	}
}

func TestConsoleLogUnsubscribeStopsDelivery(t *testing.T) {
	c := NewConsoleLog(8)
	ch, unsub := c.Subscribe()

	c.Append("info", "first")
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("did not receive first frame")
	}

	unsub()
	c.Append("info", "second")

	// After unsubscribe, the channel must be closed and deliver nothing live.
	select {
	case line, ok := <-ch:
		if ok && line.Message == "second" {
			t.Fatal("received frame after unsubscribe")
		}
	case <-time.After(100 * time.Millisecond):
		// Acceptable: no live delivery.
	}
}
