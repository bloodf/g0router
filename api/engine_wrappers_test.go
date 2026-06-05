package api

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// wrapperEngine is a minimal InferenceEngine fake recording the requests it
// receives so the unexported preprocessing/capturing wrappers can be tested
// directly without binding a socket.
type wrapperEngine struct {
	dispatchResp *providers.ChatResponse
	dispatchErr  error
	stream       chan providers.StreamChunk
	streamErr    error
	models       []providers.Model
	modelsErr    error
	lastReq      *providers.ChatRequest
}

func (e *wrapperEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	e.lastReq = req
	return e.dispatchResp, e.dispatchErr
}

func (e *wrapperEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	e.lastReq = req
	if e.streamErr != nil {
		return nil, e.streamErr
	}
	return e.stream, nil
}

func (e *wrapperEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return e.models, e.modelsErr
}

func TestUpdateSettings(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	srv := NewServer(ServerConfig{Store: s})
	settings := store.Settings{RTKEnabled: true, CavemanEnabled: true, CavemanLevel: "full"}
	if err := srv.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	srv.settingsMu.RLock()
	cached := srv.settingsCache
	srv.settingsMu.RUnlock()
	if cached == nil || !cached.RTKEnabled {
		t.Fatalf("settings cache not populated: %+v", cached)
	}
	got := srv.runtimeSettings()
	if !got.RTKEnabled || !got.CavemanEnabled {
		t.Fatalf("runtimeSettings = %+v", got)
	}
}

func TestUpdateSettingsNoStore(t *testing.T) {
	srv := NewServer(ServerConfig{})
	if err := srv.UpdateSettings(store.Settings{}); err == nil {
		t.Fatal("expected error when store is nil")
	}
}

func TestUpdateSettingsStoreError(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "closed.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	srv := NewServer(ServerConfig{Store: s})
	if err := srv.UpdateSettings(store.Settings{}); err == nil {
		t.Fatal("expected error from closed store")
	}
}

func TestPreprocessingEngineWrappers(t *testing.T) {
	base := &wrapperEngine{
		dispatchResp: &providers.ChatResponse{Model: "m"},
		models:       []providers.Model{{ID: "m1"}},
	}
	settings := store.Settings{RTKEnabled: true, CavemanEnabled: true, CavemanLevel: "full"}
	eng := preprocessingInferenceEngine{
		base:     base,
		settings: func() store.Settings { return settings },
	}

	ctx := context.Background()
	req := &providers.ChatRequest{Model: "gpt-4o", Messages: []providers.Message{{Role: "user", Content: "hi"}}}

	if _, err := eng.Dispatch(ctx, req); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if base.lastReq == nil {
		t.Fatal("base did not receive request")
	}

	if _, err := eng.DispatchStream(ctx, req); err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}

	models, err := eng.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 1 || models[0].ID != "m1" {
		t.Fatalf("ListModels = %+v", models)
	}

	// nil request path through preprocess
	if got := eng.preprocess(ctx, nil); got != nil {
		t.Fatalf("preprocess(nil) = %+v", got)
	}
}

func TestCapturingEngineDispatch(t *testing.T) {
	wantErr := errors.New("boom")
	base := &wrapperEngine{dispatchErr: wantErr}
	cap := &capturingInferenceEngine{base: base}

	req := &providers.ChatRequest{Model: "x"}
	if _, err := cap.Dispatch(context.Background(), req); !errors.Is(err, wantErr) {
		t.Fatalf("Dispatch err = %v", err)
	}
	if cap.request != req || cap.err == nil || cap.response != nil {
		t.Fatalf("capture state wrong: req=%v err=%v resp=%v", cap.request, cap.err, cap.response)
	}

	// success path
	resp := &providers.ChatResponse{Model: "ok"}
	base2 := &wrapperEngine{dispatchResp: resp}
	cap2 := &capturingInferenceEngine{base: base2}
	if _, err := cap2.Dispatch(context.Background(), req); err != nil {
		t.Fatalf("Dispatch success: %v", err)
	}
	if cap2.response != resp {
		t.Fatalf("response not captured: %v", cap2.response)
	}
}

func TestCapturingEngineListModels(t *testing.T) {
	base := &wrapperEngine{models: []providers.Model{{ID: "a"}}}
	cap := &capturingInferenceEngine{base: base}
	models, err := cap.ListModels(context.Background())
	if err != nil || len(models) != 1 {
		t.Fatalf("ListModels = %v %v", models, err)
	}
}

func TestCapturingEngineStreamError(t *testing.T) {
	wantErr := errors.New("stream boom")
	base := &wrapperEngine{streamErr: wantErr}
	cap := &capturingInferenceEngine{base: base}
	if _, err := cap.DispatchStream(context.Background(), &providers.ChatRequest{}); !errors.Is(err, wantErr) {
		t.Fatalf("DispatchStream err = %v", err)
	}
	if cap.streamed {
		t.Fatal("streamed should be false on error")
	}
}

func TestCapturingEngineStreamNil(t *testing.T) {
	base := &wrapperEngine{stream: nil}
	cap := &capturingInferenceEngine{base: base}
	out, err := cap.DispatchStream(context.Background(), &providers.ChatRequest{})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	if out != nil || cap.streamed {
		t.Fatalf("expected nil stream, streamed=false; got %v %v", out, cap.streamed)
	}
}

func TestCapturingEngineStreamSuccess(t *testing.T) {
	in := make(chan providers.StreamChunk, 2)
	in <- providers.StreamChunk{Model: "sm", Usage: &providers.Usage{TotalTokens: 5}}
	in <- providers.StreamChunk{Model: "sm"}
	close(in)

	var gotModel string
	var gotUsage *providers.Usage
	done := make(chan struct{})
	base := &wrapperEngine{stream: in}
	cap := &capturingInferenceEngine{
		base: base,
		onStreamComplete: func(_ *providers.ChatRequest, model string, u *providers.Usage) {
			gotModel = model
			gotUsage = u
			close(done)
		},
	}

	out, err := cap.DispatchStream(context.Background(), &providers.ChatRequest{})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}
	if !cap.streamed {
		t.Fatal("streamed should be true")
	}
	count := 0
	for range out {
		count++
	}
	<-done
	if count != 2 {
		t.Fatalf("chunk count = %d", count)
	}
	if gotModel != "sm" || gotUsage == nil || gotUsage.TotalTokens != 5 {
		t.Fatalf("stream complete model=%q usage=%v", gotModel, gotUsage)
	}
}

// TestCapturingEngineStreamConsumerDisconnect verifies that a consumer that
// stops reading does not stall the capture goroutine: cancelling the context
// (as the body-stream writer does on client disconnect) must let it exit even
// though the upstream stream still has buffered chunks and never closes.
func TestCapturingEngineStreamConsumerDisconnect(t *testing.T) {
	in := make(chan providers.StreamChunk, 2)
	in <- providers.StreamChunk{Model: "a"}
	// in is intentionally never closed and has more chunks pending than the
	// consumer will read, mimicking an upstream stream that outlives the client.
	in <- providers.StreamChunk{Model: "b"}

	completeCalled := make(chan struct{}, 1)
	base := &wrapperEngine{stream: in}
	cap := &capturingInferenceEngine{
		base:             base,
		onStreamComplete: func(*providers.ChatRequest, string, *providers.Usage) { completeCalled <- struct{}{} },
	}

	ctx, cancel := context.WithCancel(context.Background())
	out, err := cap.DispatchStream(ctx, &providers.ChatRequest{})
	if err != nil {
		t.Fatalf("DispatchStream: %v", err)
	}

	// Read one chunk then "disconnect" by cancelling and never reading again.
	<-out
	cancel()

	// out must close (capture goroutine returned) within the timeout.
	select {
	case _, ok := <-out:
		_ = ok
	case <-time.After(2 * time.Second):
		t.Fatal("capture goroutine did not exit after consumer disconnect (deadlock)")
	}
	// onStreamComplete must NOT fire on an abandoned stream.
	select {
	case <-completeCalled:
		t.Fatal("onStreamComplete fired despite consumer disconnect")
	default:
	}
}

func TestServerListener(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0})
	ln := srv.listener()
	if ln == nil {
		t.Fatal("listener returned nil for port 0")
	}
	t.Cleanup(func() { _ = ln.Close() })

	// invalid port forces net.Listen error -> nil
	bad := NewServer(ServerConfig{Port: -1})
	if got := bad.listener(); got != nil {
		_ = got.Close()
		t.Fatal("expected nil listener for invalid port")
	}
}
