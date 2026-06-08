package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// DebugLogger writes structured debug/trace output to stderr.
// It is safe for concurrent use.
type DebugLogger struct {
	debug  bool
	trace  bool
	out    io.Writer
	mu     sync.Mutex
	prefix string
}

// NewDebugLogger returns a logger that emits when debug or trace is enabled.
func NewDebugLogger(debug, trace bool) *DebugLogger {
	return &DebugLogger{
		debug: debug,
		trace: trace,
		out:   os.Stderr,
	}
}

// NewDebugLoggerWithOutput returns a logger writing to out (used in tests).
func NewDebugLoggerWithOutput(debug, trace bool, out io.Writer) *DebugLogger {
	return &DebugLogger{
		debug: debug,
		trace: trace,
		out:   out,
	}
}

func (l *DebugLogger) log(level, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	fmt.Fprintf(l.out, "%s [%s] %s", ts, level, msg)
	if len(args) > 0 {
		fmt.Fprint(l.out, " |")
		for i := 0; i < len(args); i += 2 {
			key := fmt.Sprint(args[i])
			var value string
			if i+1 < len(args) {
				value = fmt.Sprint(args[i+1])
			}
			fmt.Fprintf(l.out, " %s=%q", key, value)
		}
	}
	fmt.Fprintln(l.out)
}

// Debug emits a debug line when debug mode is enabled.
func (l *DebugLogger) Debug(msg string, args ...any) {
	if !l.debug && !l.trace {
		return
	}
	l.log("DEBUG", msg, args...)
}

// Trace emits a trace line when trace mode is enabled.
func (l *DebugLogger) Trace(msg string, args ...any) {
	if !l.trace {
		return
	}
	l.log("TRACE", msg, args...)
}

// IsDebug reports whether debug logging is enabled.
func (l *DebugLogger) IsDebug() bool { return l.debug }

// IsTrace reports whether trace logging is enabled.
func (l *DebugLogger) IsTrace() bool { return l.trace }
