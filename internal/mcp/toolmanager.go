package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
)

const toolNameSeparator = "__"

var (
	ErrInvalidManifest       = errors.New("mcp: invalid manifest")
	ErrToolAlreadyRegistered = errors.New("mcp: tool already registered")
	ErrToolNotFound          = errors.New("mcp: tool not found")
)

type ToolManager struct {
	tools   map[string]Tool
	order   []string
	clients map[string]Client
}

func NewToolManager() *ToolManager {
	return &ToolManager{
		tools:   make(map[string]Tool),
		clients: make(map[string]Client),
	}
}

func (m *ToolManager) RegisterManifest(manifest Manifest) error {
	if manifest.ClientID == "" {
		return ErrInvalidManifest
	}

	for _, tool := range manifest.Tools {
		if tool.Name == "" {
			return ErrInvalidManifest
		}
		fullName := toolFullName(manifest.ClientID, tool.Name)
		if _, ok := m.tools[fullName]; ok {
			return ErrToolAlreadyRegistered
		}
	}

	for _, tool := range manifest.Tools {
		fullName := toolFullName(manifest.ClientID, tool.Name)
		tool.ClientID = manifest.ClientID
		m.tools[fullName] = tool
		m.order = append(m.order, fullName)
	}
	return nil
}

func (m *ToolManager) CompactTools() []providers.Tool {
	tools := make([]providers.Tool, 0, len(m.order))
	for _, fullName := range m.order {
		tool := m.tools[fullName]
		tools = append(tools, providers.Tool{
			Type: "function",
			Function: providers.ToolFunction{
				Name:        fullName,
				Description: tool.Description,
			},
		})
	}
	return tools
}

func (m *ToolManager) Lookup(name string) (Tool, error) {
	tool, ok := m.tools[name]
	if !ok {
		return Tool{}, ErrToolNotFound
	}
	return tool, nil
}

func (m *ToolManager) RegisterClient(clientID string, client Client) {
	m.clients[clientID] = client
}

func (m *ToolManager) Call(ctx context.Context, name string, arguments json.RawMessage) (CallResult, error) {
	tool, err := m.Lookup(name)
	if err != nil {
		return CallResult{}, err
	}

	client, ok := m.clients[tool.ClientID]
	if !ok {
		return CallResult{}, ErrClientNotFound
	}

	result, err := client.CallTool(ctx, CallRequest{Name: tool.Name, Arguments: arguments})
	if err != nil {
		return CallResult{}, fmt.Errorf("call mcp tool %q: %w", name, err)
	}
	return result, nil
}

func toolFullName(clientID, toolName string) string {
	return clientID + toolNameSeparator + toolName
}
