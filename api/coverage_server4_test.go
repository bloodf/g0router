package api

import (
	"bytes"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/traffic"
	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

// errorListener is a net.Listener that always returns an error on Accept.
type errorListener struct {
	net.Listener
}

func (e *errorListener) Accept() (net.Conn, error) {
	return nil, errors.New("injected accept error")
}

func (e *errorListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

// TestServeErrorListenerReturnsError exercises the error branch in Serve when
// the listener returns a non-recoverable error.
func TestServeErrorListenerReturnsError(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0})
	err := srv.Serve(&errorListener{})
	if err == nil {
		t.Fatal("Serve with error listener should return error")
	}
}

// TestObserveRequestMetricNilTrafficBroker covers the branch where metrics is
// non-nil but trafficBroker is nil.
func TestObserveRequestMetricNilTrafficBroker(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0})
	srv.trafficBroker = nil
	// Must not panic when trafficBroker is nil.
	srv.observeRequestMetric(requestLogMetadata{}, &usage.Usage{InputTokens: 1, OutputTokens: 1}, nil, 200, time.Second)
}

// TestHandleTrafficStreamMarshalErrorSkipped exercises the json.Marshal error
// branch in handleTrafficStream: an event with an unmarshalable timestamp is
// skipped rather than crashing the stream.
func TestHandleTrafficStreamMarshalErrorSkipped(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s, UsageStore: s})

	// Publish an event with an out-of-range timestamp that fails json.Marshal.
	srv.trafficBroker.Publish(traffic.Event{
		Timestamp: time.Date(292277026596, 1, 1, 0, 0, 0, 0, time.UTC),
		Provider:  "openai",
		Model:     "gpt-4o",
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	srv.handleTrafficStream(&ctx)

	done := make(chan []byte)
	go func() {
		var buf bytes.Buffer
		_ = ctx.Response.BodyWriteTo(&buf)
		done <- buf.Bytes()
	}()

	// Close stopCh so the stream exits. Use a longer delay so the callback
	// goroutine has time to reach the replay loop before we signal stop.
	time.Sleep(200 * time.Millisecond)
	close(srv.stopCh)

	body := <-done
	if bytes.Contains(body, []byte("openai")) {
		t.Fatalf("marshal-error event should have been skipped: %s", body)
	}
}

// TestHandleTrafficStreamValidEventReplayed verifies that a valid event in the
// broker's ring buffer is replayed through the SSE stream.
func TestHandleTrafficStreamValidEventReplayed(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s, UsageStore: s})

	srv.trafficBroker.Publish(traffic.Event{
		Timestamp:   time.Now().UTC(),
		Provider:    "openai",
		Model:       "gpt-4o",
		StatusClass: "2xx",
		StatusCode:  200,
		LatencyMS:   42,
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	srv.handleTrafficStream(&ctx)

	done := make(chan []byte)
	go func() {
		var buf bytes.Buffer
		_ = ctx.Response.BodyWriteTo(&buf)
		done <- buf.Bytes()
	}()

	time.Sleep(200 * time.Millisecond)
	close(srv.stopCh)

	body := <-done
	if !bytes.Contains(body, []byte(`"provider":"openai"`)) {
		t.Fatalf("valid event should be replayed: %s", body)
	}
}
