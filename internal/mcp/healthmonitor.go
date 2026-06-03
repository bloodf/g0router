package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrInvalidHealthInterval = errors.New("mcp: invalid health interval")

type HealthStatus struct {
	ClientID string
	Healthy  bool
	Error    string
}

type HealthMonitor struct {
	manager   *ClientManager
	tools     *ToolManager
	newTicker func(time.Duration) healthTicker
	mu        sync.RWMutex
	statuses  map[string]HealthStatus
}

func NewHealthMonitor(manager *ClientManager, tools ...*ToolManager) *HealthMonitor {
	var toolManager *ToolManager
	if len(tools) > 0 {
		toolManager = tools[0]
	}
	return &HealthMonitor{
		manager:   manager,
		tools:     toolManager,
		newTicker: newRealHealthTicker,
		statuses:  make(map[string]HealthStatus),
	}
}

func (m *HealthMonitor) Check(ctx context.Context, clientID string) (HealthStatus, error) {
	client, ok := m.manager.Client(clientID)
	if !ok {
		return HealthStatus{}, ErrClientNotFound
	}

	status, err := m.checkClient(ctx, clientID, client)
	if err != nil {
		return status, err
	}
	return status, nil
}

func (m *HealthMonitor) CheckAll(ctx context.Context) map[string]HealthStatus {
	clients := m.registeredClients()
	statuses := make(map[string]HealthStatus, len(clients))

	for clientID, client := range clients {
		statuses[clientID] = m.checkClientStatus(ctx, clientID, client)
	}
	m.setStatuses(statuses)
	return statuses
}

func (m *HealthMonitor) Statuses() map[string]HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]HealthStatus, len(m.statuses))
	for clientID, status := range m.statuses {
		statuses[clientID] = status
	}
	return statuses
}

func (m *HealthMonitor) Start(ctx context.Context, interval time.Duration) (*HealthMonitorLoop, error) {
	if interval <= 0 {
		return nil, ErrInvalidHealthInterval
	}
	if ctx == nil {
		ctx = context.Background()
	}

	runCtx, cancel := context.WithCancel(ctx)
	ticker := m.newTicker(interval)
	loop := &HealthMonitorLoop{
		cancel:  cancel,
		done:    make(chan struct{}),
		updates: make(chan map[string]HealthStatus, 1),
	}

	go m.run(runCtx, ticker, loop.done, loop.updates)
	return loop, nil
}

func (m *HealthMonitor) run(ctx context.Context, ticker healthTicker, done chan<- struct{}, updates chan<- map[string]HealthStatus) {
	defer close(done)
	defer close(updates)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			statuses := m.CheckAll(ctx)
			select {
			case updates <- cloneStatuses(statuses):
			default:
			}
		}
	}
}

type HealthMonitorLoop struct {
	cancel   context.CancelFunc
	done     chan struct{}
	updates  chan map[string]HealthStatus
	stopOnce sync.Once
}

func (l *HealthMonitorLoop) Stop() {
	l.stopOnce.Do(func() {
		l.cancel()
		<-l.done
	})
}

func (l *HealthMonitorLoop) Done() <-chan struct{} {
	return l.done
}

func (l *HealthMonitorLoop) Updates() <-chan map[string]HealthStatus {
	return l.updates
}

type healthTicker interface {
	C() <-chan time.Time
	Stop()
}

type realHealthTicker struct {
	ticker *time.Ticker
}

func newRealHealthTicker(interval time.Duration) healthTicker {
	return realHealthTicker{ticker: time.NewTicker(interval)}
}

func (t realHealthTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t realHealthTicker) Stop() {
	t.ticker.Stop()
}

func (m *HealthMonitor) checkClient(ctx context.Context, clientID string, client Client) (HealthStatus, error) {
	tools, err := client.ListTools(ctx)
	if err != nil {
		wrapped := fmt.Errorf("check mcp client %q health: %w", clientID, err)
		status := HealthStatus{ClientID: clientID, Healthy: false, Error: wrapped.Error()}
		m.setStatus(status)
		return status, wrapped
	}

	if m.tools != nil {
		if err := m.tools.RefreshManifest(Manifest{ClientID: clientID, Tools: tools}); err != nil {
			wrapped := fmt.Errorf("refresh mcp client %q tools: %w", clientID, err)
			status := HealthStatus{ClientID: clientID, Healthy: false, Error: wrapped.Error()}
			m.setStatus(status)
			return status, wrapped
		}
	}
	status := HealthStatus{ClientID: clientID, Healthy: true}
	m.setStatus(status)
	return status, nil
}

func (m *HealthMonitor) checkClientStatus(ctx context.Context, clientID string, client Client) HealthStatus {
	status, _ := m.checkClient(ctx, clientID, client)
	return status
}

func (m *HealthMonitor) setStatus(status HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.statuses[status.ClientID] = status
}

func (m *HealthMonitor) setStatuses(statuses map[string]HealthStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.statuses = cloneStatuses(statuses)
}

func cloneStatuses(statuses map[string]HealthStatus) map[string]HealthStatus {
	cloned := make(map[string]HealthStatus, len(statuses))
	for clientID, status := range statuses {
		cloned[clientID] = status
	}
	return cloned
}

func (m *HealthMonitor) registeredClients() map[string]Client {
	m.manager.mu.Lock()
	defer m.manager.mu.Unlock()

	clients := make(map[string]Client, len(m.manager.clients))
	for clientID, client := range m.manager.clients {
		clients[clientID] = client
	}
	return clients
}
