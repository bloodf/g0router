package mcp

import (
	"bytes"
	"testing"
)

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
