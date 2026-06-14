package mcp

import "bytes"

// splitFrames consumes complete newline-delimited JSON frames from buf, returning
// the complete frames and the remaining partial tail (the bytes after the last
// newline, held until its own newline arrives in a later chunk). Blank lines are
// skipped. PURE — no I/O. Mirrors 9router's newline-split of proc.stdout
// (stdioSseBridge.js:151).
func splitFrames(buf []byte) (frames [][]byte, rest []byte) {
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			break
		}
		line := bytes.TrimRight(buf[:i], "\r")
		if len(line) > 0 {
			// Copy so callers may reuse/append to the original buffer safely.
			frame := make([]byte, len(line))
			copy(frame, line)
			frames = append(frames, frame)
		}
		buf = buf[i+1:]
	}
	return frames, buf
}
