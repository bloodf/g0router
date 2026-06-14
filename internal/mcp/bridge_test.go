package mcp

import (
	"bytes"
	"errors"
	"sync"
	"testing"
)

// fakeProcess is a deterministic Process used to unit-test the Bridge WITHOUT
// spawning any real process. It records frames written to stdin and exposes a
// controllable liveness flag.
type fakeProcess struct {
	mu      sync.Mutex
	written [][]byte
	running bool
	stopped bool
	writeErr error
}

func (f *fakeProcess) Write(frame []byte) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(frame))
	copy(cp, frame)
	f.written = append(f.written, cp)
	return nil
}

func (f *fakeProcess) IsRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

func (f *fakeProcess) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = true
	f.running = false
	return nil
}

// recordingSink captures frames broadcast to one session.
type recordingSink struct {
	mu     sync.Mutex
	frames [][]byte
}

func (r *recordingSink) sink(frame []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]byte, len(frame))
	copy(cp, frame)
	r.frames = append(r.frames, cp)
	return nil
}

func (r *recordingSink) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.frames)
}

// TestBridgeBroadcastReachesAllSinks: onFrame splits the stdout buffer and
// broadcasts every complete frame to every registered session sink (PAR-MCP-007).
func TestBridgeBroadcastReachesAllSinks(t *testing.T) {
	b := newBridge(&fakeProcess{running: true})
	s1, s2 := &recordingSink{}, &recordingSink{}
	b.AddSession("a", s1.sink)
	b.AddSession("b", s2.sink)

	b.onFrame([]byte(`{"x":1}` + "\n"))
	if s1.count() != 1 || s2.count() != 1 {
		t.Fatalf("broadcast counts: s1=%d s2=%d, want 1,1", s1.count(), s2.count())
	}
	if string(s1.frames[0]) != `{"x":1}` {
		t.Fatalf("s1 frame = %q", s1.frames[0])
	}
}

// TestBridgeFailingSinkDroppedWithoutAbort: a sink that errors is removed from
// the session map and does NOT abort the broadcast to the other sinks
// (PAR-MCP-054 "ignore broken pipe").
func TestBridgeFailingSinkDroppedWithoutAbort(t *testing.T) {
	b := newBridge(&fakeProcess{running: true})
	good := &recordingSink{}
	b.AddSession("good", good.sink)
	b.AddSession("broken", func([]byte) error { return errors.New("broken pipe") })

	b.onFrame([]byte(`{"y":2}` + "\n"))
	if good.count() != 1 {
		t.Fatalf("good sink did not receive frame despite broken sibling: %d", good.count())
	}
	// The broken sink is dropped; a second frame still reaches the good sink and
	// the broken one is no longer attempted.
	b.onFrame([]byte(`{"z":3}` + "\n"))
	if good.count() != 2 {
		t.Fatalf("good sink count after second frame = %d, want 2", good.count())
	}
	if b.sessionCount() != 1 {
		t.Fatalf("broken sink not dropped: sessionCount = %d, want 1", b.sessionCount())
	}
}

// TestBridgeRemoveSession: a removed session no longer receives frames.
func TestBridgeRemoveSession(t *testing.T) {
	b := newBridge(&fakeProcess{running: true})
	s := &recordingSink{}
	b.AddSession("s", s.sink)
	b.RemoveSession("s")
	b.onFrame([]byte(`{"q":1}` + "\n"))
	if s.count() != 0 {
		t.Fatalf("removed session still received %d frames", s.count())
	}
}

// TestBridgeSendWritesToProcess: Send forwards a JSON-RPC frame to the child
// process stdin via Process.Write.
func TestBridgeSendWritesToProcess(t *testing.T) {
	fp := &fakeProcess{running: true}
	b := newBridge(fp)
	if err := b.Send([]byte(`{"method":"ping"}`)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(fp.written) != 1 || string(fp.written[0]) != `{"method":"ping"}` {
		t.Fatalf("process did not receive frame: %q", fp.written)
	}
}

// TestBridgeIsRunningDelegates: IsRunning reflects the underlying process
// liveness (PAR-MCP-051).
func TestBridgeIsRunningDelegates(t *testing.T) {
	fp := &fakeProcess{running: true}
	b := newBridge(fp)
	if !b.IsRunning() {
		t.Fatalf("IsRunning = false, want true")
	}
	fp.Stop()
	if b.IsRunning() {
		t.Fatalf("IsRunning = true after stop, want false")
	}
}

// TestBridgeOnExitFiresCallback: onExit invokes the registered exit callback once
// with the child's exit code (PAR-MCP-053 — the launcher's registry-delete).
func TestBridgeOnExitFiresCallback(t *testing.T) {
	b := newBridge(&fakeProcess{running: true})
	var gotCode int
	called := 0
	b.SetOnExit(func(code int) { called++; gotCode = code })
	b.onExit(7)
	if called != 1 || gotCode != 7 {
		t.Fatalf("onExit callback: called=%d code=%d, want 1,7", called, gotCode)
	}
}

// TestBridgeOnStderrFiresCallback: stderr lines are delivered to the registered
// stderr callback (PAR-MCP-052).
func TestBridgeOnStderrFiresCallback(t *testing.T) {
	b := newBridge(&fakeProcess{running: true})
	var lines []string
	b.SetOnStderr(func(line string) { lines = append(lines, line) })
	b.onStderr("a warning")
	if len(lines) != 1 || lines[0] != "a warning" {
		t.Fatalf("onStderr lines = %v", lines)
	}
}

// TestBridgeOnFramePartialCarryover: onFrame buffers a partial frame across two
// chunks and only broadcasts once the newline arrives.
func TestBridgeOnFramePartialCarryover(t *testing.T) {
	b := newBridge(&fakeProcess{running: true})
	s := &recordingSink{}
	b.AddSession("s", s.sink)
	b.onFrame([]byte(`{"par`))
	if s.count() != 0 {
		t.Fatalf("partial frame broadcast prematurely: %d", s.count())
	}
	b.onFrame([]byte(`tial":1}` + "\n"))
	if s.count() != 1 || string(s.frames[0]) != `{"partial":1}` {
		t.Fatalf("partial carryover frame = %q", s.frames)
	}
}

// TestSplitFramesSingle: one complete newline-delimited frame is returned with
// an empty remainder.
func TestSplitFramesSingle(t *testing.T) {
	frames, rest := splitFrames([]byte(`{"a":1}` + "\n"))
	if len(frames) != 1 || string(frames[0]) != `{"a":1}` {
		t.Fatalf("frames = %q", frames)
	}
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
}

// TestSplitFramesTwoInOneChunk: two frames in one buffer are both returned.
func TestSplitFramesTwoInOneChunk(t *testing.T) {
	frames, rest := splitFrames([]byte(`{"a":1}` + "\n" + `{"b":2}` + "\n"))
	if len(frames) != 2 {
		t.Fatalf("len(frames) = %d, want 2", len(frames))
	}
	if string(frames[0]) != `{"a":1}` || string(frames[1]) != `{"b":2}` {
		t.Fatalf("frames = %q", frames)
	}
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
}

// TestSplitFramesPartialCarryover: a partial frame (no trailing newline) is held
// in rest until its newline arrives in a later chunk.
func TestSplitFramesPartialCarryover(t *testing.T) {
	frames, rest := splitFrames([]byte(`{"a":1}` + "\n" + `{"par`))
	if len(frames) != 1 || string(frames[0]) != `{"a":1}` {
		t.Fatalf("frames = %q", frames)
	}
	if string(rest) != `{"par` {
		t.Fatalf("rest = %q, want partial held", rest)
	}
	// Next chunk completes the partial frame.
	frames2, rest2 := splitFrames(append(rest, []byte(`tial":2}`+"\n")...))
	if len(frames2) != 1 || string(frames2[0]) != `{"partial":2}` {
		t.Fatalf("frames2 = %q", frames2)
	}
	if len(rest2) != 0 {
		t.Fatalf("rest2 = %q, want empty", rest2)
	}
}

// TestSplitFramesEmpty: empty input yields no frames and empty remainder.
func TestSplitFramesEmpty(t *testing.T) {
	frames, rest := splitFrames(nil)
	if len(frames) != 0 {
		t.Fatalf("frames = %q, want none", frames)
	}
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
}

// TestSplitFramesBlankLinesSkipped: blank lines between frames are skipped.
func TestSplitFramesBlankLinesSkipped(t *testing.T) {
	frames, rest := splitFrames([]byte(`{"a":1}` + "\n\n" + `{"b":2}` + "\n"))
	if len(frames) != 2 {
		t.Fatalf("len(frames) = %d, want 2 (blank skipped): %q", len(frames), frames)
	}
	if !bytes.Equal(frames[0], []byte(`{"a":1}`)) || !bytes.Equal(frames[1], []byte(`{"b":2}`)) {
		t.Fatalf("frames = %q", frames)
	}
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
}
