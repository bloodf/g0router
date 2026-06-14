package logging

import (
	"sync"
	"time"
)

// ConsoleLine is a single captured server log line. Timestamp is RFC3339.
type ConsoleLine struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// subscriberBuffer is the per-subscriber channel buffer. A consumer slower than
// this many pending frames drops frames rather than blocking the log path.
const subscriberBuffer = 256

// ConsoleLog is an in-process bounded ring buffer of recent log lines with a
// fan-out to live subscribers. It never blocks the log path: a slow subscriber
// drops frames instead of stalling Append. No global state, no init().
type ConsoleLog struct {
	mu          sync.Mutex
	capacity    int
	ring        []ConsoleLine
	subscribers map[int]chan ConsoleLine
	nextSubID   int
}

// NewConsoleLog creates a ConsoleLog holding up to capacity recent lines.
func NewConsoleLog(capacity int) *ConsoleLog {
	if capacity < 1 {
		capacity = 1
	}
	return &ConsoleLog{
		capacity:    capacity,
		ring:        make([]ConsoleLine, 0, capacity),
		subscribers: make(map[int]chan ConsoleLine),
	}
}

// Append records a line in the ring (dropping the oldest at capacity) and
// fans it out to all live subscribers. A subscriber whose buffer is full has
// the frame dropped — Append never blocks.
func (c *ConsoleLog) Append(level, message string) {
	line := ConsoleLine{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.mu.Lock()
	if len(c.ring) >= c.capacity {
		// Drop oldest.
		copy(c.ring, c.ring[1:])
		c.ring[len(c.ring)-1] = line
	} else {
		c.ring = append(c.ring, line)
	}
	subs := make([]chan ConsoleLine, 0, len(c.subscribers))
	for _, ch := range c.subscribers {
		subs = append(subs, ch)
	}
	c.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- line:
		default:
			// Slow consumer: drop the frame rather than block.
		}
	}
}

// Subscribe returns a buffered channel of future lines plus an unsubscribe
// closure. The closure removes the subscriber and closes the channel; it is
// safe to call more than once.
func (c *ConsoleLog) Subscribe() (<-chan ConsoleLine, func()) {
	ch := make(chan ConsoleLine, subscriberBuffer)
	c.mu.Lock()
	id := c.nextSubID
	c.nextSubID++
	c.subscribers[id] = ch
	c.mu.Unlock()

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			c.mu.Lock()
			if existing, ok := c.subscribers[id]; ok {
				delete(c.subscribers, id)
				close(existing)
			}
			c.mu.Unlock()
		})
	}
	return ch, unsub
}

// Recent returns a snapshot copy of the buffered lines, oldest first.
func (c *ConsoleLog) Recent() []ConsoleLine {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]ConsoleLine, len(c.ring))
	copy(out, c.ring)
	return out
}
