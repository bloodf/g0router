package mcp

import (
	"context"
	"errors"
	"testing"
)

type fakeClient struct {
	tools      []Tool
	callResult CallResult
	err        error
	calls      []CallRequest
	closed     bool
}

func (f *fakeClient) ListTools(ctx context.Context) ([]Tool, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tools, nil
}

func (f *fakeClient) CallTool(ctx context.Context, req CallRequest) (CallResult, error) {
	if f.err != nil {
		return CallResult{}, f.err
	}
	f.calls = append(f.calls, req)
	return f.callResult, nil
}

func (f *fakeClient) Close() error {
	f.closed = true
	return f.err
}

type fakeConnector struct {
	client *fakeClient
	err    error
}

func (f fakeConnector) Connect(ctx context.Context, cfg ClientConfig) (Client, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.client, nil
}

func TestClientManagerConnectsAndRegistersManifest(t *testing.T) {
	client := &fakeClient{tools: []Tool{
		{Name: "search", Description: "Search docs", InputSchema: []byte(`{"type":"object"}`)},
	}}
	manager := NewClientManager(fakeConnector{client: client})

	manifest, err := manager.Register(context.Background(), ClientConfig{
		ID:        "docs",
		Name:      "Docs",
		Transport: TransportStdio,
		Command:   "mcp-docs",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if manifest.ClientID != "docs" {
		t.Fatalf("ClientID = %q, want docs", manifest.ClientID)
	}
	if len(manifest.Tools) != 1 || manifest.Tools[0].Name != "search" {
		t.Fatalf("manifest tools = %#v", manifest.Tools)
	}
	got, ok := manager.Client("docs")
	if !ok {
		t.Fatal("client was not registered")
	}
	if got != client {
		t.Fatal("registered client mismatch")
	}
}

func TestClientManagerRejectsDuplicateClientID(t *testing.T) {
	manager := NewClientManager(fakeConnector{client: &fakeClient{}})
	cfg := ClientConfig{ID: "docs", Name: "Docs", Transport: TransportStdio, Command: "mcp-docs"}

	if _, err := manager.Register(context.Background(), cfg); err != nil {
		t.Fatalf("Register first client: %v", err)
	}
	_, err := manager.Register(context.Background(), cfg)
	if !errors.Is(err, ErrClientAlreadyRegistered) {
		t.Fatalf("expected ErrClientAlreadyRegistered, got %v", err)
	}
}

func TestClientManagerWrapsConnectorAndToolErrors(t *testing.T) {
	connectErr := errors.New("spawn failed")
	manager := NewClientManager(fakeConnector{err: connectErr})

	_, err := manager.Register(context.Background(), ClientConfig{ID: "docs", Name: "Docs", Transport: TransportStdio})
	if !errors.Is(err, connectErr) {
		t.Fatalf("expected wrapped connect error, got %v", err)
	}

	toolErr := errors.New("list failed")
	manager = NewClientManager(fakeConnector{client: &fakeClient{err: toolErr}})
	_, err = manager.Register(context.Background(), ClientConfig{ID: "docs", Name: "Docs", Transport: TransportStdio})
	if !errors.Is(err, toolErr) {
		t.Fatalf("expected wrapped list tools error, got %v", err)
	}
}

func TestClientManagerCloseRemovesClient(t *testing.T) {
	client := &fakeClient{}
	manager := NewClientManager(fakeConnector{client: client})
	if _, err := manager.Register(context.Background(), ClientConfig{ID: "docs", Name: "Docs", Transport: TransportStdio}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := manager.Close("docs"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !client.closed {
		t.Fatal("client was not closed")
	}
	if _, ok := manager.Client("docs"); ok {
		t.Fatal("client still registered after close")
	}
}

func TestClientManagerCloseUnknownClient(t *testing.T) {
	manager := NewClientManager(fakeConnector{client: &fakeClient{}})

	err := manager.Close("missing")
	if !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("expected ErrClientNotFound, got %v", err)
	}
}

func TestClientManagerConcurrentReads(t *testing.T) {
	client := &fakeClient{tools: []Tool{{Name: "x", InputSchema: []byte(`{}`)}}}
	manager := NewClientManager(fakeConnector{client: client})
	if _, err := manager.Register(context.Background(), ClientConfig{ID: "c1", Name: "C1", Transport: TransportStdio}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	const n = 50
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, ok := manager.Client("c1")
			if !ok {
				errs <- errors.New("client not found")
				return
			}
			errs <- nil
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent read: %v", err)
		}
	}
}
