package governance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

// fakeSender records the dispatch call and returns a configurable error,
// performing no network I/O.
type fakeSender struct {
	called      bool
	channelType string
	config      map[string]any
	err         error
}

func (f *fakeSender) Send(_ context.Context, channelType string, config map[string]any) error {
	f.called = true
	f.channelType = channelType
	f.config = config
	return f.err
}

func TestAlertDispatcherSuccess(t *testing.T) {
	fs := &fakeSender{}
	d := NewAlertDispatcher(fs)

	ch := &store.AlertChannel{
		ChannelType: "webhook",
		Config:      map[string]any{"url": "https://hooks.example.com/x"},
	}
	ok, msg := d.Dispatch(context.Background(), ch)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if msg == "" {
		t.Fatal("message empty")
	}
	if !fs.called {
		t.Fatal("fake sender was not called")
	}
	if fs.channelType != "webhook" {
		t.Fatalf("channelType = %q", fs.channelType)
	}
	if fs.config["url"] != "https://hooks.example.com/x" {
		t.Fatalf("config not forwarded: %+v", fs.config)
	}
}

func TestAlertDispatcherErrorNeverLeaksSecret(t *testing.T) {
	secretURL := "https://hooks.example.com/SUPERSECRETTOKEN"
	fs := &fakeSender{err: errors.New("connection refused to " + secretURL)}
	d := NewAlertDispatcher(fs)

	ch := &store.AlertChannel{
		ChannelType: "webhook",
		Config:      map[string]any{"url": secretURL, "token": "tok-XYZSECRET"},
	}
	ok, msg := d.Dispatch(context.Background(), ch)
	if ok {
		t.Fatalf("ok = true, want false on sender error")
	}
	if strings.Contains(msg, secretURL) || strings.Contains(msg, "SUPERSECRETTOKEN") {
		t.Fatalf("message leaks secret URL: %q", msg)
	}
	if strings.Contains(msg, "tok-XYZSECRET") {
		t.Fatalf("message leaks token: %q", msg)
	}
}
