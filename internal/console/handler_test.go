package console

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestEntryReachesParentAndBroker(t *testing.T) {
	broker := NewBroker(16)
	var buf bytes.Buffer
	parent := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	handler := NewTeeHandler(parent, broker, slog.LevelDebug)
	logger := slog.New(handler)

	logger.Info("test message", slog.String("key", "val"))

	// Parent handler should have received the log.
	if !strings.Contains(buf.String(), "test message") {
		t.Fatalf("parent handler missing message, got: %s", buf.String())
	}

	// Broker should have the entry.
	recent := broker.Recent()
	if len(recent) != 1 {
		t.Fatalf("broker Recent len = %d, want 1", len(recent))
	}
	if recent[0].Message != "test message" {
		t.Fatalf("broker entry Message = %q, want %q", recent[0].Message, "test message")
	}
	if recent[0].Level != "INFO" {
		t.Fatalf("broker entry Level = %q, want INFO", recent[0].Level)
	}
}

func TestLevelFiltering(t *testing.T) {
	broker := NewBroker(16)
	var buf bytes.Buffer
	parent := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	// Tee handler filters at INFO — DEBUG should not reach broker.
	handler := NewTeeHandler(parent, broker, slog.LevelInfo)
	logger := slog.New(handler)

	logger.Debug("debug message")
	logger.Info("info message")

	recent := broker.Recent()
	if len(recent) != 1 {
		t.Fatalf("broker Recent len = %d, want 1 (only INFO)", len(recent))
	}
	if recent[0].Message != "info message" {
		t.Fatalf("broker entry Message = %q, want %q", recent[0].Message, "info message")
	}
	if recent[0].Level != "INFO" {
		t.Fatalf("broker entry Level = %q, want INFO", recent[0].Level)
	}

	// Parent still receives both.
	if !strings.Contains(buf.String(), "debug message") {
		t.Fatalf("parent handler missing debug message, got: %s", buf.String())
	}
}

func TestAttributesConverted(t *testing.T) {
	broker := NewBroker(16)
	parent := slog.NewTextHandler(ioDiscard{}, &slog.HandlerOptions{Level: slog.LevelDebug})

	handler := NewTeeHandler(parent, broker, slog.LevelDebug)
	logger := slog.New(handler)

	logger.Warn("warn msg", slog.String("k1", "v1"), slog.Int("k2", 42))

	recent := broker.Recent()
	if len(recent) != 1 {
		t.Fatalf("broker Recent len = %d, want 1", len(recent))
	}
	ent := recent[0]
	if len(ent.Attrs) != 2 {
		t.Fatalf("broker entry Attrs len = %d, want 2", len(ent.Attrs))
	}

	attrMap := make(map[string]string, len(ent.Attrs))
	for _, a := range ent.Attrs {
		attrMap[a.Key] = a.Value
	}
	if attrMap["k1"] != "v1" {
		t.Fatalf("attr k1 = %q, want v1", attrMap["k1"])
	}
	if attrMap["k2"] != "42" {
		t.Fatalf("attr k2 = %q, want 42", attrMap["k2"])
	}
}

func TestWithAttrsAndWithGroupReturnSlogHandler(t *testing.T) {
	broker := NewBroker(16)
	parent := slog.NewTextHandler(ioDiscard{}, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewTeeHandler(parent, broker, slog.LevelDebug)

	ctx := context.Background()
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)

	// WithAttrs should return a slog.Handler that can Handle.
	h2 := handler.WithAttrs([]slog.Attr{slog.String("a", "b")})
	if err := h2.Handle(ctx, record); err != nil {
		t.Fatalf("WithAttrs handler Handle: %v", err)
	}

	// WithGroup should return a slog.Handler that can Handle.
	h3 := handler.WithGroup("group")
	if err := h3.Handle(ctx, record); err != nil {
		t.Fatalf("WithGroup handler Handle: %v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
