package admin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

// productionKeepalive is the SSE keepalive interval in production.
const productionKeepalive = 25 * time.Second

// streamStatsService is the subset of the usage stats service needed by the
// usage stream. It is implemented by *usage.StatsService.
type streamStatsService interface {
	StatsMap(period string) (map[string]any, error)
	StreamSnapshot() (map[string]any, error)
	StreamEvents() *usage.Events
}

// UsageStreamHandler serves GET /api/usage/stream as a Server-Sent Events stream.
type UsageStreamHandler struct {
	Handlers *Handlers
	Keepalive time.Duration
	stats     streamStatsService
}

// UsageStream writes an SSE stream of usage updates.
func (h *UsageStreamHandler) UsageStream(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		h.serve(w, ctx.Done(), nil)
	})
}

// serve runs the SSE loop against the provided writer. If done is non-nil it is
// closed when the loop exits, which lets tests observe goroutine termination.
func (h *UsageStreamHandler) serve(w *bufio.Writer, clientDone <-chan struct{}, done chan<- struct{}) {
	if done != nil {
		defer close(done)
	}

	interval := h.Keepalive
	if interval <= 0 {
		interval = productionKeepalive
	}

	statsSvc := h.stats
	if statsSvc == nil {
		statsSvc = h.Handlers.stats
	}

	events := statsSvc.StreamEvents()
	state := &streamState{writer: w}

	if err := state.sendFull(statsSvc); err != nil {
		return
	}

	stop := make(chan struct{})
	var closeStop sync.Once

	var updateFn, pendingFn func(string)
	updateFn = func(kind string) {
		if kind != "update" {
			return
		}
		if err := state.sendQuick(statsSvc); err != nil {
			closeStop.Do(func() { close(stop) })
			return
		}
		if err := state.sendFull(statsSvc); err != nil {
			closeStop.Do(func() { close(stop) })
		}
	}
	pendingFn = func(kind string) {
		if kind != "pending" {
			return
		}
		if err := state.sendQuick(statsSvc); err != nil {
			closeStop.Do(func() { close(stop) })
		}
	}

	events.OnEvent(updateFn)
	events.OnEvent(pendingFn)
	defer func() {
		events.OffEvent(updateFn)
		events.OffEvent(pendingFn)
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-clientDone:
			return
		case <-stop:
			return
		case <-ticker.C:
			if err := state.writePing(); err != nil {
				return
			}
		}
	}
}

// streamState holds the cached stats and write mutex for a single SSE stream.
type streamState struct {
	mu      sync.Mutex
	closed  bool
	cached  map[string]any
	writer  *bufio.Writer
}

func (s *streamState) sendFull(stats streamStatsService) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	full, err := stats.StatsMap("all")
	if err != nil {
		s.closed = true
		return fmt.Errorf("full stats: %w", err)
	}
	s.cached = full
	return s.writeData(full)
}

func (s *streamState) sendQuick(stats streamStatsService) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.cached == nil {
		return nil
	}
	snap, err := stats.StreamSnapshot()
	if err != nil {
		s.closed = true
		return fmt.Errorf("stream snapshot: %w", err)
	}
	quick, err := overlayMap(s.cached, snap)
	if err != nil {
		s.closed = true
		return fmt.Errorf("overlay cached stats: %w", err)
	}
	return s.writeData(quick)
}

func (s *streamState) writePing() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if _, err := s.writer.WriteString(": ping\n\n"); err != nil {
		s.closed = true
		return err
	}
	return s.writer.Flush()
}

func (s *streamState) writeData(data map[string]any) error {
	b, err := json.Marshal(data)
	if err != nil {
		s.closed = true
		return fmt.Errorf("marshal frame: %w", err)
	}
	if _, err := s.writer.WriteString("data: "); err != nil {
		s.closed = true
		return err
	}
	if _, err := s.writer.Write(b); err != nil {
		s.closed = true
		return err
	}
	if _, err := s.writer.WriteString("\n\n"); err != nil {
		s.closed = true
		return err
	}
	if err := s.writer.Flush(); err != nil {
		s.closed = true
		return err
	}
	return nil
}

func overlayMap(base, overlay map[string]any) (map[string]any, error) {
	b, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("marshal base: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal base: %w", err)
	}
	for k, v := range overlay {
		m[k] = v
	}
	return m, nil
}
