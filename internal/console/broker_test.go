package console

import (
	"sync"
	"testing"
	"time"
)

func TestPublishSubscribeReceivesEntry(t *testing.T) {
	b := NewBroker(16)
	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	ent := Entry{
		Timestamp: time.Now().UTC(),
		Level:     "INFO",
		Message:   "hello",
		Attrs:     []Attr{{Key: "k", Value: "v"}},
	}
	b.Publish(ent)

	select {
	case got := <-ch:
		if got.Message != ent.Message || got.Level != ent.Level {
			t.Fatalf("got %+v, want %+v", got, ent)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for entry")
	}
}

func TestRecentReturnsCopyInOrder(t *testing.T) {
	b := NewBroker(4)

	entries := []Entry{
		{Message: "a", Level: "INFO"},
		{Message: "b", Level: "WARN"},
		{Message: "c", Level: "ERROR"},
	}
	for _, ent := range entries {
		b.Publish(ent)
	}

	recent := b.Recent()
	if len(recent) != 3 {
		t.Fatalf("Recent() len = %d, want 3", len(recent))
	}
	for i, ent := range entries {
		if recent[i].Message != ent.Message {
			t.Fatalf("recent[%d].Message = %q, want %q", i, recent[i].Message, ent.Message)
		}
	}
}

func TestRecentRingBufferOverflow(t *testing.T) {
	ringSize := 3
	b := NewBroker(ringSize)

	for i := 0; i < 5; i++ {
		b.Publish(Entry{Message: string(rune('a' + i)), Level: "INFO"})
	}

	recent := b.Recent()
	if len(recent) != ringSize {
		t.Fatalf("Recent() len = %d, want %d", len(recent), ringSize)
	}
	want := []string{"c", "d", "e"}
	for i, w := range want {
		if recent[i].Message != w {
			t.Fatalf("recent[%d].Message = %q, want %q", i, recent[i].Message, w)
		}
	}
}

func TestRecentReturnsCopy(t *testing.T) {
	b := NewBroker(8)
	b.Publish(Entry{Message: "original", Level: "INFO"})

	r := b.Recent()
	r[0].Message = "mutated"

	r2 := b.Recent()
	if r2[0].Message == "mutated" {
		t.Fatal("Recent() returned a slice that aliases internal state")
	}
}

func TestSlowSubscriberDoesNotBlockPublish(t *testing.T) {
	b := NewBroker(16)
	id, _ := b.Subscribe()
	defer b.Unsubscribe(id)

	_, readerCh := b.Subscribe()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			b.Publish(Entry{Message: "msg", Level: "INFO"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on slow subscriber")
	}

	select {
	case <-readerCh:
	default:
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := NewBroker(16)
	id, ch := b.Subscribe()

	b.Publish(Entry{Message: "before-unsub", Level: "INFO"})

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected first entry")
	}

	b.Unsubscribe(id)

	for range ch {
	}

	b.Publish(Entry{Message: "after-unsub", Level: "INFO"})

	select {
	case ent, ok := <-ch:
		if ok {
			t.Fatalf("received live entry after Unsubscribe: %+v", ent)
		}
	case <-time.After(50 * time.Millisecond):
	}
}

func TestConcurrentPublishersRaceSafe(t *testing.T) {
	b := NewBroker(64)
	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	const goroutines = 8
	const entriesEach = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < entriesEach; i++ {
				b.Publish(Entry{Message: "msg", Level: "INFO"})
			}
		}(g)
	}

	go func() {
		for range ch {
		}
	}()

	wg.Wait()
	b.Unsubscribe(id)
}

func TestNewBrokerNonPositiveRingSizeDefaultsToOne(t *testing.T) {
	b := NewBroker(0)
	b.Publish(Entry{Message: "first", Level: "INFO"})
	b.Publish(Entry{Message: "second", Level: "INFO"})
	recent := b.Recent()
	if len(recent) != 1 {
		t.Fatalf("Recent len = %d, want 1 (ring clamped to 1)", len(recent))
	}
	if recent[0].Message != "second" {
		t.Fatalf("Recent[0].Message = %q, want second (newest)", recent[0].Message)
	}

	bNeg := NewBroker(-5)
	bNeg.Publish(Entry{Message: "x", Level: "INFO"})
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

func TestClearEmptiesRing(t *testing.T) {
	b := NewBroker(8)
	b.Publish(Entry{Message: "a", Level: "INFO"})
	b.Publish(Entry{Message: "b", Level: "WARN"})

	if got := b.Recent(); len(got) != 2 {
		t.Fatalf("before Clear: Recent len = %d, want 2", len(got))
	}

	b.Clear()

	if got := b.Recent(); got != nil {
		t.Fatalf("after Clear: Recent() = %v, want nil", got)
	}

	// Publishing after clear should work normally.
	b.Publish(Entry{Message: "c", Level: "ERROR"})
	recent := b.Recent()
	if len(recent) != 1 || recent[0].Message != "c" {
		t.Fatalf("after Clear + Publish: Recent = %+v, want 1 entry with Message=c", recent)
	}
}
