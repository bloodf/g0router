package admin

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/usage"
)

// recordingWriter is an io.Writer that can be closed to simulate a client disconnect.
type recordingWriter struct {
	mu     sync.Mutex
	closed bool
	buf    bytes.Buffer
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, &testWriteError{}
	}
	return w.buf.Write(p)
}

func (w *recordingWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
}

func (w *recordingWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Return a copy to avoid races.
	out := make([]byte, w.buf.Len())
	copy(out, w.buf.Bytes())
	return out
}

type testWriteError struct{}

func (e *testWriteError) Error() string { return "write after close" }

type streamFrame struct {
	comment bool
	data    string
}

func parseFrames(t *testing.T, raw []byte) []streamFrame {
	t.Helper()
	var frames []streamFrame
	for _, chunk := range strings.Split(string(raw), "\n\n") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		if strings.HasPrefix(chunk, ": ") {
			frames = append(frames, streamFrame{comment: true, data: strings.TrimPrefix(chunk, ": ")})
			continue
		}
		if strings.HasPrefix(chunk, "data: ") {
			frames = append(frames, streamFrame{data: strings.TrimPrefix(chunk, "data: ")})
			continue
		}
		t.Fatalf("unrecognized frame: %q", chunk)
	}
	return frames
}

func frameData(t *testing.T, f streamFrame) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(f.data), &m); err != nil {
		t.Fatalf("decode frame data: %v\nraw: %s", err, f.data)
	}
	return m
}

type fakeStreamStats struct {
	mu         sync.Mutex
	events     *usage.Events
	stats      map[string]any
	snapshot   map[string]any
	statsCalls int
	snapCalls  int
}

func (f *fakeStreamStats) StatsMap(string) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statsCalls++
	return copyMap(f.stats), nil
}

func (f *fakeStreamStats) StreamSnapshot() (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapCalls++
	return copyMap(f.snapshot), nil
}

func (f *fakeStreamStats) StreamEvents() *usage.Events { return f.events }

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func newUsageStreamHandler(t *testing.T) (*UsageStreamHandler, *fakeStreamStats, *usage.Events) {
	t.Helper()
	events := usage.NewEvents()
	fake := &fakeStreamStats{
		events: events,
		stats:  map[string]any{"total_requests": float64(7), "full": true},
		snapshot: map[string]any{
			"active_requests": []any{"a1"},
			"recent_requests": []any{"r1"},
			"error_provider":  "anthropic",
		},
	}
	return &UsageStreamHandler{stats: fake, Keepalive: time.Hour}, fake, events
}

func TestUsageStreamPushesOnUpdate(t *testing.T) {
	h, _, events := newUsageStreamHandler(t)
	rec := &recordingWriter{}
	w := bufio.NewWriter(rec)
	done := make(chan struct{})

	go h.serve(w, nil, done)

	// Wait for the initial full frame.
	var frames []streamFrame
	waitFor(t, time.Second, func() bool {
		frames = parseFrames(t, rec.Bytes())
		return len(frames) >= 1
	})
	if len(frames) != 1 {
		t.Fatalf("initial frames = %d, want 1", len(frames))
	}
	initial := frameData(t, frames[0])
	if initial["full"] != true {
		t.Fatalf("initial frame missing full: %v", initial)
	}

	time.Sleep(50 * time.Millisecond)
	events.Emit("update")

	waitFor(t, time.Second, func() bool {
		frames = parseFrames(t, rec.Bytes())
		return len(frames) >= 3
	})
	if len(frames) != 3 {
		t.Fatalf("after update frames = %d, want 3", len(frames))
	}

	quick := frameData(t, frames[1])
	if quick["full"] != true {
		t.Fatalf("quick frame did not overlay cached stats: %v", quick)
	}
	if active, ok := quick["active_requests"].([]any); !ok || len(active) != 1 || active[0] != "a1" {
		t.Fatalf("quick frame active_requests = %v", quick["active_requests"])
	}
	full := frameData(t, frames[2])
	if full["full"] != true {
		t.Fatalf("full frame missing full: %v", full)
	}

	rec.Close()
	time.Sleep(20 * time.Millisecond)
	events.Emit("update")
	waitFor(t, time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}

func TestUsageStreamPendingLightweight(t *testing.T) {
	h, fake, events := newUsageStreamHandler(t)
	rec := &recordingWriter{}
	w := bufio.NewWriter(rec)
	done := make(chan struct{})

	go h.serve(w, nil, done)

	var frames []streamFrame
	waitFor(t, time.Second, func() bool {
		frames = parseFrames(t, rec.Bytes())
		return len(frames) >= 1
	})

	time.Sleep(50 * time.Millisecond)
	events.Emit("pending")

	waitFor(t, time.Second, func() bool {
		frames = parseFrames(t, rec.Bytes())
		return len(frames) >= 2
	})
	if len(frames) != 2 {
		t.Fatalf("after pending frames = %d, want 2", len(frames))
	}

	pending := frameData(t, frames[1])
	if pending["full"] != true {
		t.Fatalf("pending frame did not start from cached stats: %v", pending)
	}
	if pending["error_provider"] != "anthropic" {
		t.Fatalf("pending frame error_provider = %v", pending["error_provider"])
	}

	fake.mu.Lock()
	statsCalls := fake.statsCalls
	fake.mu.Unlock()
	if statsCalls != 1 {
		t.Fatalf("StatsMap called %d times, want 1 (initial only)", statsCalls)
	}

	rec.Close()
	time.Sleep(20 * time.Millisecond)
	events.Emit("update")
	waitFor(t, time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}

func TestUsageStreamKeepalive(t *testing.T) {
	h, _, _ := newUsageStreamHandler(t)
	h.Keepalive = 50 * time.Millisecond
	rec := &recordingWriter{}
	w := bufio.NewWriter(rec)
	done := make(chan struct{})

	go h.serve(w, nil, done)

	var frames []streamFrame
	waitFor(t, time.Second, func() bool {
		frames = parseFrames(t, rec.Bytes())
		for _, f := range frames {
			if f.comment && f.data == "ping" {
				return true
			}
		}
		return false
	})

	rec.Close()
	waitFor(t, time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}

func TestUsageStreamUnsubscribesOnClose(t *testing.T) {
	h, _, events := newUsageStreamHandler(t)
	rec := &recordingWriter{}
	w := bufio.NewWriter(rec)
	done := make(chan struct{})

	go h.serve(w, nil, done)

	var frames []streamFrame
	waitFor(t, time.Second, func() bool {
		frames = parseFrames(t, rec.Bytes())
		return len(frames) >= 1
	})

	rec.Close()
	time.Sleep(20 * time.Millisecond)
	events.Emit("update")
	events.Emit("update")

	waitFor(t, time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})

	afterClose := len(parseFrames(t, rec.Bytes()))
	time.Sleep(100 * time.Millisecond)

	if got := len(parseFrames(t, rec.Bytes())); got != afterClose {
		t.Fatalf("frame count changed after close: %d -> %d", afterClose, got)
	}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}
