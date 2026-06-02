package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

type Transport string

const (
	TransportStdio          Transport = "stdio"
	TransportSSE            Transport = "sse"
	TransportStreamableHTTP Transport = "streamable-http"
)

var (
	ErrClientAlreadyRegistered = errors.New("mcp: client already registered")
	ErrClientNotFound          = errors.New("mcp: client not found")
	ErrInvalidClientConfig     = errors.New("mcp: invalid client config")
)

type ClientConfig struct {
	ID        string
	Name      string
	Transport Transport
	Command   string
	Args      []string
	Env       map[string]string
	URL       string
}

type Tool struct {
	ClientID    string
	Name        string
	Description string
	InputSchema json.RawMessage
}

type Manifest struct {
	ClientID string
	Tools    []Tool
}

type CallRequest struct {
	Name      string
	Arguments json.RawMessage
}

type CallResult struct {
	Content any
}

type Client interface {
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, req CallRequest) (CallResult, error)
	Close() error
}

type Connector interface {
	Connect(ctx context.Context, cfg ClientConfig) (Client, error)
}

type ClientManager struct {
	connector Connector
	mu        sync.Mutex
	clients   map[string]Client
}

func NewClientManager(connector Connector) *ClientManager {
	return &ClientManager{
		connector: connector,
		clients:   make(map[string]Client),
	}
}

func (m *ClientManager) Register(ctx context.Context, cfg ClientConfig) (Manifest, error) {
	if cfg.ID == "" || cfg.Transport == "" {
		return Manifest{}, ErrInvalidClientConfig
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[cfg.ID]; ok {
		return Manifest{}, ErrClientAlreadyRegistered
	}

	client, err := m.connector.Connect(ctx, cfg)
	if err != nil {
		return Manifest{}, fmt.Errorf("connect mcp client %q: %w", cfg.ID, err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		_ = client.Close()
		return Manifest{}, fmt.Errorf("list tools for mcp client %q: %w", cfg.ID, err)
	}
	for i := range tools {
		tools[i].ClientID = cfg.ID
	}

	m.clients[cfg.ID] = client
	return Manifest{ClientID: cfg.ID, Tools: tools}, nil
}

func (m *ClientManager) Client(id string) (Client, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[id]
	return client, ok
}

func (m *ClientManager) Close(id string) error {
	m.mu.Lock()
	client, ok := m.clients[id]
	if !ok {
		m.mu.Unlock()
		return ErrClientNotFound
	}
	delete(m.clients, id)
	m.mu.Unlock()

	if err := client.Close(); err != nil {
		return fmt.Errorf("close mcp client %q: %w", id, err)
	}
	return nil
}
