// Package traffic provides a live-traffic event broker for the dashboard topology view.
package traffic

import (
	"sync"
	"testing"
	"time"
)

func TestPublishSubscribeReceivesEvent(t *testing.T) {
	b := NewBroker(16)
	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	ev := Event{
		Timestamp:   time.Now().UTC(),
		KeyID:       "key-1",
		Provider:    "openai",
		Model:       "gpt-4o",
		StatusClass: "2xx",
		StatusCode:  200,
		LatencyMS:   42,
	}
	b.Publish(ev)

	select {
	case got := <-ch:
		if got.KeyID != ev.KeyID || got.Provider != ev.Provider || got.Model != ev.Model {
			t.Fatalf("got %+v, want %+v", got, ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestRecentReturnsCopyInOrder(t *testing.T) {
	b := NewBroker(4)

	events := []Event{
		{KeyID: "a", StatusCode: 200},
		{KeyID: "b", StatusCode: 201},
		{KeyID: "c", StatusCode: 202},
	}
	for _, ev := range events {
		b.Publish(ev)
	}

	recent := b.Recent()
	if len(recent) != 3 {
		t.Fatalf("Recent() len = %d, want 3", len(recent))
	}
	for i, ev := range events {
		if recent[i].KeyID != ev.KeyID {
			t.Fatalf("recent[%d].KeyID = %q, want %q", i, recent[i].KeyID, ev.KeyID)
		}
	}
}

func TestRecentRingBufferOverflow(t *testing.T) {
	ringSize := 3
	b := NewBroker(ringSize)

	for i := 0; i < 5; i++ {
		b.Publish(Event{KeyID: string(rune('a' + i))})
	}

	recent := b.Recent()
	if len(recent) != ringSize {
		t.Fatalf("Recent() len = %d, want %d", len(recent), ringSize)
	}
	// Should contain the last 3: c, d, e
	want := []string{"c", "d", "e"}
	for i, w := range want {
		if recent[i].KeyID != w {
			t.Fatalf("recent[%d].KeyID = %q, want %q", i, recent[i].KeyID, w)
		}
	}
}

func TestRecentReturnsCopy(t *testing.T) {
	b := NewBroker(8)
	b.Publish(Event{KeyID: "original"})

	r := b.Recent()
	r[0].KeyID = "mutated"

	r2 := b.Recent()
	if r2[0].KeyID == "mutated" {
		t.Fatal("Recent() returned a slice that aliases internal state")
	}
}

func TestSlowSubscriberDoesNotBlockPublish(t *testing.T) {
	b := NewBroker(16)
	// Subscribe but never read — channel will fill and overflow.
	id, _ := b.Subscribe()
	defer b.Unsubscribe(id)

	// Also subscribe a reader so we can confirm publish returns.
	_, readerCh := b.Subscribe()

	done := make(chan struct{})
	go func() {
		// Publish many events; slow subscriber should never block.
		for i := 0; i < 100; i++ {
			b.Publish(Event{StatusCode: i})
		}
		close(done)
	}()

	select {
	case <-done:
		// Good — publish loop completed without blocking.
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on slow subscriber")
	}

	// Reader should have received at least one event (channel buffered).
	select {
	case <-readerCh:
	default:
		// It's fine if readerCh is also overflowed; the point was non-blocking.
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := NewBroker(16)
	id, ch := b.Subscribe()

	b.Publish(Event{KeyID: "before-unsub"})

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected first event")
	}

	b.Unsubscribe(id)

	// Channel was closed by Unsubscribe. Drain remaining buffered items via
	// range — it will exit when the closed channel is empty.
	for range ch {
	}

	b.Publish(Event{KeyID: "after-unsub"})

	// Channel is already closed; any receive returns zero value immediately.
	// There is no way to deliver new events to a closed/removed subscriber,
	// so we just verify the broker did not panic and the channel is drained.
	select {
	case ev, ok := <-ch:
		if ok {
			t.Fatalf("received live event after Unsubscribe: %+v", ev)
		}
		// zero value from closed channel — expected
	case <-time.After(50 * time.Millisecond):
		// Good — channel is closed, nothing to receive.
	}
}

func TestConcurrentPublishersRaceSafe(t *testing.T) {
	b := NewBroker(64)
	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	const goroutines = 8
	const eventsEach = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < eventsEach; i++ {
				b.Publish(Event{StatusCode: g*1000 + i})
			}
		}(g)
	}

	// Drain concurrently so the channel doesn't fill.
	go func() {
		for range ch {
		}
	}()

	wg.Wait()
	b.Unsubscribe(id)
}

func TestNewBrokerNonPositiveRingSizeDefaultsToOne(t *testing.T) {
	b := NewBroker(0)
	b.Publish(Event{Provider: "openai"})
	b.Publish(Event{Provider: "anthropic"})
	recent := b.Recent()
	if len(recent) != 1 {
		t.Fatalf("Recent len = %d, want 1 (ring clamped to 1)", len(recent))
	}
	if recent[0].Provider != "anthropic" {
		t.Fatalf("Recent[0].Provider = %q, want anthropic (newest)", recent[0].Provider)
	}

	bNeg := NewBroker(-5)
	bNeg.Publish(Event{Provider: "x"})
	if got := bNeg.Recent(); len(got) != 1 {
		t.Fatalf("negative ring: Recent len = %d, want 1", len(got))
	}
}

func TestRecentEmptyReturnsNil(t *testing.T) {
	b := NewBroker(8)
	if got := b.Recent(); got != nil {
		t.Fatalf("Recent() on empty broker = %v, want nil", got)
	}
}
