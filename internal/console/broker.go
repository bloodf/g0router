// Package console provides a live-log entry broker for the dashboard console
// view. Entries are published by the custom slog handler and consumed by SSE
// subscribers and the ring-buffer replay endpoint.
package console

import (
	"sync"
	"time"
)

// Entry describes a single log record captured by the TeeHandler.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Attrs     []Attr    `json:"attrs,omitempty"`
}

// Attr is a key/value pair attached to a log Entry.
type Attr struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Broker fans out published entries to all current subscribers and maintains a
// fixed-size ring buffer for replay. All methods are safe for concurrent use.
type Broker struct {
	mu       sync.Mutex
	nextID   uint64
	subs     map[uint64]chan Entry
	ring     []Entry
	ringSize int
	head     int // index of the next write slot
	count    int // number of valid entries (0..ringSize)
}

// NewBroker returns a new Broker with a ring buffer of ringSize entries.
// ringSize must be > 0.
func NewBroker(ringSize int) *Broker {
	if ringSize <= 0 {
		ringSize = 1
	}
	return &Broker{
		subs:     make(map[uint64]chan Entry),
		ring:     make([]Entry, ringSize),
		ringSize: ringSize,
	}
}

// Subscribe registers a new subscriber and returns its ID and a channel on
// which entries will be delivered. The channel is buffered (size = ringSize) so
// a slow reader incurs drops rather than blocking the publish path.
func (b *Broker) Subscribe() (uint64, <-chan Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	ch := make(chan Entry, b.ringSize)
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

// Publish records ent in the ring buffer and delivers it to all current
// subscribers. Delivery to each subscriber is non-blocking: if the subscriber
// channel is full the entry is dropped for that subscriber.
func (b *Broker) Publish(ent Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Write into ring buffer.
	b.ring[b.head] = ent
	b.head = (b.head + 1) % b.ringSize
	if b.count < b.ringSize {
		b.count++
	}

	// Fan out to subscribers — non-blocking.
	for _, ch := range b.subs {
		select {
		case ch <- ent:
		default:
			// Subscriber is slow; drop rather than block the request path.
		}
	}
}

// Recent returns a copy of the ring buffer contents in insertion order
// (oldest first, most-recent last). The returned slice is a fresh copy; the
// caller may mutate it freely.
func (b *Broker) Recent() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return nil
	}
	out := make([]Entry, b.count)
	start := 0
	if b.count == b.ringSize {
		start = b.head
	}
	for i := 0; i < b.count; i++ {
		out[i] = b.ring[(start+i)%b.ringSize]
	}
	return out
}

// Clear removes all entries from the ring buffer and resets the internal
// counters. Subscribers are not affected.
func (b *Broker) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.head = 0
	b.count = 0
	for i := range b.ring {
		b.ring[i] = Entry{}
	}
}
