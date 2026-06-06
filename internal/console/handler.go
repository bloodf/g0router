package console

import (
	"context"
	"log/slog"
)

// TeeHandler is a slog.Handler that tees log records to both a parent handler
// and a console Broker. Only records with level >= the configured level are
// forwarded to the broker; the parent handler always receives every record.
type TeeHandler struct {
	parent slog.Handler
	broker *Broker
	level  slog.Level
}

// NewTeeHandler returns a new TeeHandler.
func NewTeeHandler(parent slog.Handler, broker *Broker, level slog.Level) *TeeHandler {
	return &TeeHandler{
		parent: parent,
		broker: broker,
		level:  level,
	}
}

// Enabled reports whether the handler handles records at the given level.
// The parent handler's decision is used.
func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.parent.Enabled(ctx, level)
}

// Handle publishes the record to the broker (if level >= h.level) and always
// delegates to the parent handler.
func (h *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= h.level {
		ent := Entry{
			Timestamp: r.Time,
			Level:     levelString(r.Level),
			Message:   r.Message,
		}
		r.Attrs(func(a slog.Attr) bool {
			ent.Attrs = append(ent.Attrs, Attr{Key: a.Key, Value: a.Value.String()})
			return true
		})
		h.broker.Publish(ent)
	}
	return h.parent.Handle(ctx, r)
}

// WithAttrs returns a new TeeHandler whose parent handler has the given attrs.
func (h *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewTeeHandler(h.parent.WithAttrs(attrs), h.broker, h.level)
}

// WithGroup returns a new TeeHandler whose parent handler has the given group.
func (h *TeeHandler) WithGroup(name string) slog.Handler {
	return NewTeeHandler(h.parent.WithGroup(name), h.broker, h.level)
}

func levelString(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARN"
	case level >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
