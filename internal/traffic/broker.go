// Package traffic provides a live-traffic event broker for the dashboard
// topology view. Events are published by the request-completion path and
// consumed by SSE subscribers and the ring-buffer replay endpoint.
package traffic

import (
	"sync"
	"time"
)

// Event describes a single completed inference request. Fields are kept to
// what is directly derivable at request-completion time.
type Event struct {
	Timestamp   time.Time `json:"timestamp"`
	KeyID       string    `json:"key_id"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	StatusClass string    `json:"status_class"`
	StatusCode  int       `json:"status_code"`
	LatencyMS   int64     `json:"latency_ms"`
}

// Broker fans out published events to all current subscribers and maintains a
// fixed-size ring buffer for replay. All methods are safe for concurrent use.
type Broker struct {
	mu       sync.Mutex
	nextID   uint64
	subs     map[uint64]chan Event
	ring     []Event
	ringSize int
	head     int // index of the next write slot
	count    int // number of valid entries (0..ringSize)
}

// NewBroker returns a new Broker with a ring buffer of ringSize events.
// ringSize must be > 0.
func NewBroker(ringSize int) *Broker {
	if ringSize <= 0 {
		ringSize = 1
	}
	return &Broker{
		subs:     make(map[uint64]chan Event),
		ring:     make([]Event, ringSize),
		ringSize: ringSize,
	}
}

// Subscribe registers a new subscriber and returns its ID and a channel on
// which events will be delivered. The channel is buffered (size = ringSize) so
// a slow reader incurs drops rather than blocking the publish path.
func (b *Broker) Subscribe() (uint64, <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	ch := make(chan Event, b.ringSize)
	b.subs[id] = ch
	return id, ch
}

// Unsubscribe removes the subscriber identified by id and closes its channel.
func (b *Broker) Unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subs[id]; ok {
		delete(b.subs, id)
		close(ch)
	}
}

// Publish records ev in the ring buffer and delivers it to all current
// subscribers. Delivery to each subscriber is non-blocking: if the subscriber
// channel is full the event is dropped for that subscriber.
func (b *Broker) Publish(ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Write into ring buffer.
	b.ring[b.head] = ev
	b.head = (b.head + 1) % b.ringSize
	if b.count < b.ringSize {
		b.count++
	}

	// Fan out to subscribers — non-blocking.
	for _, ch := range b.subs {
		select {
		case ch <- ev:
		default:
			// Subscriber is slow; drop rather than block the request path.
		}
	}
}

// Recent returns a copy of the ring buffer contents in insertion order
// (oldest first, most-recent last). The returned slice is a fresh copy; the
// caller may mutate it freely.
func (b *Broker) Recent() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return nil
	}
	out := make([]Event, b.count)
	// oldest slot: when the ring is full head points to the oldest entry;
	// when not full the oldest entry is at index 0.
	start := 0
	if b.count == b.ringSize {
		start = b.head
	}
	for i := 0; i < b.count; i++ {
		out[i] = b.ring[(start+i)%b.ringSize]
	}
	return out
}
