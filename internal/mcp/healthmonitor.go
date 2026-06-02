package mcp

import (
	"context"
	"fmt"
)

type HealthStatus struct {
	ClientID string
	Healthy  bool
	Error    string
}

type HealthMonitor struct {
	manager *ClientManager
}

func NewHealthMonitor(manager *ClientManager) *HealthMonitor {
	return &HealthMonitor{manager: manager}
}

func (m *HealthMonitor) Check(ctx context.Context, clientID string) (HealthStatus, error) {
	client, ok := m.manager.Client(clientID)
	if !ok {
		return HealthStatus{}, ErrClientNotFound
	}

	if _, err := client.ListTools(ctx); err != nil {
		wrapped := fmt.Errorf("check mcp client %q health: %w", clientID, err)
		return HealthStatus{ClientID: clientID, Healthy: false, Error: wrapped.Error()}, wrapped
	}

	return HealthStatus{ClientID: clientID, Healthy: true}, nil
}

func (m *HealthMonitor) CheckAll(ctx context.Context) map[string]HealthStatus {
	clients := m.registeredClients()
	statuses := make(map[string]HealthStatus, len(clients))

	for clientID, client := range clients {
		statuses[clientID] = checkClientHealth(ctx, clientID, client)
	}
	return statuses
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

func checkClientHealth(ctx context.Context, clientID string, client Client) HealthStatus {
	if _, err := client.ListTools(ctx); err != nil {
		wrapped := fmt.Errorf("check mcp client %q health: %w", clientID, err)
		return HealthStatus{ClientID: clientID, Healthy: false, Error: wrapped.Error()}
	}
	return HealthStatus{ClientID: clientID, Healthy: true}
}
