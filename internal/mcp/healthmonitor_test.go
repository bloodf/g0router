package mcp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestHealthMonitorChecksRegisteredClient(t *testing.T) {
	manager := NewClientManager(fakeConnector{client: &fakeClient{
		tools: []Tool{{Name: "search", Description: "Search docs"}},
	}})
	if _, err := manager.Register(context.Background(), ClientConfig{ID: "docs", Transport: TransportStdio}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	monitor := NewHealthMonitor(manager)
	status, err := monitor.Check(context.Background(), "docs")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if status.ClientID != "docs" {
		t.Fatalf("ClientID = %q, want docs", status.ClientID)
	}
	if !status.Healthy {
		t.Fatalf("Healthy = false, want true")
	}
	if status.Error != "" {
		t.Fatalf("Error = %q, want empty", status.Error)
	}
}

func TestHealthMonitorReportsUnhealthyClient(t *testing.T) {
	listErr := errors.New("list failed")
	client := &fakeClient{}
	manager := NewClientManager(fakeConnector{client: client})
	if _, err := manager.Register(context.Background(), ClientConfig{ID: "docs", Transport: TransportStdio}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	client.err = listErr

	monitor := NewHealthMonitor(manager)
	status, err := monitor.Check(context.Background(), "docs")
	if !errors.Is(err, listErr) {
		t.Fatalf("expected wrapped list error, got %v", err)
	}
	if status.Healthy {
		t.Fatalf("Healthy = true, want false")
	}
	if status.Error == "" {
		t.Fatal("Error is empty, want health error message")
	}
}

func TestHealthMonitorCheckUnknownClient(t *testing.T) {
	monitor := NewHealthMonitor(NewClientManager(fakeConnector{client: &fakeClient{}}))

	_, err := monitor.Check(context.Background(), "missing")
	if !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("expected ErrClientNotFound, got %v", err)
	}
}

func TestHealthMonitorChecksAllRegisteredClients(t *testing.T) {
	manager := &ClientManager{
		clients: map[string]Client{
			"docs":   &fakeClient{tools: []Tool{{Name: "search"}}},
			"broken": &fakeClient{err: errors.New("offline")},
		},
	}
	monitor := NewHealthMonitor(manager)

	statuses := monitor.CheckAll(context.Background())

	if len(statuses) != 2 {
		t.Fatalf("statuses len = %d, want 2", len(statuses))
	}
	if !statuses["docs"].Healthy {
		t.Fatalf("docs status = %#v, want healthy", statuses["docs"])
	}
	if statuses["broken"].Healthy {
		t.Fatalf("broken status = %#v, want unhealthy", statuses["broken"])
	}
	if statuses["broken"].Error == "" {
		t.Fatal("broken error is empty, want health error message")
	}
}

func TestHealthMonitorPeriodicallyRefreshesHealthAndTools(t *testing.T) {
	client := newSequencedHealthClient([][]Tool{
		{{Name: "search", Description: "Search docs"}},
		{{Name: "read", Description: "Read docs"}},
	})
	manager := &ClientManager{clients: map[string]Client{"docs": client}}
	tools := NewToolManager()
	if err := tools.RegisterManifest(Manifest{ClientID: "docs", Tools: []Tool{{Name: "stale"}}}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	ticker := newManualHealthTicker()
	monitor := NewHealthMonitor(manager, tools)
	monitor.newTicker = func(interval time.Duration) healthTicker {
		if interval != 5*time.Second {
			t.Fatalf("interval = %v, want 5s", interval)
		}
		return ticker
	}

	loop, err := monitor.Start(context.Background(), 5*time.Second)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer loop.Stop()

	ticker.Tick()
	waitUpdate(t, loop.Updates())

	statuses := monitor.Statuses()
	if !statuses["docs"].Healthy {
		t.Fatalf("docs status = %#v, want healthy", statuses["docs"])
	}
	assertCompactToolNames(t, tools, []string{"docs__search"})

	ticker.Tick()
	waitUpdate(t, loop.Updates())

	assertCompactToolNames(t, tools, []string{"docs__read"})
}

func TestHealthMonitorStopStopsPeriodicChecks(t *testing.T) {
	client := newSequencedHealthClient([][]Tool{{{Name: "search"}}})
	manager := &ClientManager{clients: map[string]Client{"docs": client}}
	ticker := newManualHealthTicker()
	monitor := NewHealthMonitor(manager)
	monitor.newTicker = func(time.Duration) healthTicker { return ticker }

	loop, err := monitor.Start(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ticker.Tick()
	client.waitCalls(t, 1)

	loop.Stop()
	ticker.waitStopped(t)
	ticker.Tick()

	if got := client.callCount(); got != 1 {
		t.Fatalf("ListTools calls after Stop = %d, want 1", got)
	}
}

func TestHealthMonitorContextCancellationStopsPeriodicChecks(t *testing.T) {
	client := newSequencedHealthClient([][]Tool{{{Name: "search"}}})
	manager := &ClientManager{clients: map[string]Client{"docs": client}}
	ticker := newManualHealthTicker()
	monitor := NewHealthMonitor(manager)
	monitor.newTicker = func(time.Duration) healthTicker { return ticker }

	ctx, cancel := context.WithCancel(context.Background())
	loop, err := monitor.Start(ctx, time.Minute)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	cancel()
	waitDone(t, loop.Done())
	ticker.waitStopped(t)
}

type sequencedHealthClient struct {
	mu        sync.Mutex
	responses [][]Tool
	calls     int
	called    chan int
}

func newSequencedHealthClient(responses [][]Tool) *sequencedHealthClient {
	return &sequencedHealthClient{
		responses: responses,
		called:    make(chan int, 8),
	}
}

func (c *sequencedHealthClient) ListTools(ctx context.Context) ([]Tool, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	c.mu.Lock()
	c.calls++
	call := c.calls
	responseIndex := call - 1
	if responseIndex >= len(c.responses) {
		responseIndex = len(c.responses) - 1
	}
	tools := append([]Tool(nil), c.responses[responseIndex]...)
	c.mu.Unlock()

	c.called <- call
	return tools, nil
}

func (c *sequencedHealthClient) CallTool(ctx context.Context, req CallRequest) (CallResult, error) {
	return CallResult{}, nil
}

func (c *sequencedHealthClient) Close() error {
	return nil
}

func (c *sequencedHealthClient) waitCalls(t *testing.T, want int) {
	t.Helper()
	for {
		select {
		case got := <-c.called:
			if got >= want {
				return
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %d ListTools calls", want)
		}
	}
}

func (c *sequencedHealthClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls
}

type manualHealthTicker struct {
	ch       chan time.Time
	stopped  chan struct{}
	stopOnce sync.Once
}

func newManualHealthTicker() *manualHealthTicker {
	return &manualHealthTicker{
		ch:      make(chan time.Time, 8),
		stopped: make(chan struct{}),
	}
}

func (t *manualHealthTicker) C() <-chan time.Time {
	return t.ch
}

func (t *manualHealthTicker) Stop() {
	t.stopOnce.Do(func() {
		close(t.stopped)
	})
}

func (t *manualHealthTicker) Tick() {
	t.ch <- time.Unix(0, 0)
}

func (t *manualHealthTicker) waitStopped(testingT *testing.T) {
	testingT.Helper()
	waitDone(testingT, t.stopped)
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for health monitor to stop")
	}
}

func waitUpdate(t *testing.T, updates <-chan map[string]HealthStatus) map[string]HealthStatus {
	t.Helper()
	select {
	case statuses := <-updates:
		return statuses
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for health monitor update")
		return nil
	}
}

func assertCompactToolNames(t *testing.T, manager *ToolManager, want []string) {
	t.Helper()

	gotTools := manager.CompactTools()
	if len(gotTools) != len(want) {
		t.Fatalf("tool count = %d, want %d: %#v", len(gotTools), len(want), gotTools)
	}
	for i, tool := range gotTools {
		if tool.Function.Name != want[i] {
			t.Fatalf("tool[%d] = %q, want %q", i, tool.Function.Name, want[i])
		}
	}
}
