package logging

import (
	"fmt"
	"io"
	"time"
)

// Debug is a production-gated debug logger.
type Debug struct {
	enabled bool
	out     io.Writer
	clock   func() time.Time
}

// NewDebug creates a debug logger that gates on NODE_ENV != "production".
func NewDebug(getenv func(string) string, out io.Writer) *Debug {
	return &Debug{
		enabled: getenv("NODE_ENV") != "production",
		out:     out,
		clock:   func() time.Time { return time.Now() },
	}
}

// Logf writes a tagged debug line when not in production.
func (d *Debug) Logf(tag, format string, args ...any) {
	if !d.enabled {
		return
	}
	ts := d.clock().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(d.out, "[%s] 🐛 [DBG:%s] %s\n", ts, tag, msg)
}
