package mcp

import (
	"context"
	"errors"
	"testing"
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
