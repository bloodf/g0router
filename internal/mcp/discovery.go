package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
)

var ErrInvalidDiscovery = errors.New("mcp: invalid discovery")

type CompactManifest struct {
	ClientID string           `json:"client_id"`
	Tools    []providers.Tool `json:"tools"`
}

type Discovery struct {
	clients *ClientManager
	tools   *ToolManager
}

func NewDiscovery(clients *ClientManager, tools *ToolManager) *Discovery {
	return &Discovery{
		clients: clients,
		tools:   tools,
	}
}

func (d *Discovery) Discover(ctx context.Context, cfg ClientConfig) (CompactManifest, error) {
	if d == nil || d.clients == nil || d.tools == nil {
		return CompactManifest{}, ErrInvalidDiscovery
	}

	manifest, err := d.clients.Register(ctx, cfg)
	if err != nil {
		return CompactManifest{}, fmt.Errorf("register mcp client %q: %w", cfg.ID, err)
	}

	if err := d.tools.RegisterManifest(manifest); err != nil {
		_ = d.clients.Close(manifest.ClientID)
		return CompactManifest{}, fmt.Errorf("register mcp tools for client %q: %w", manifest.ClientID, err)
	}

	client, ok := d.clients.Client(manifest.ClientID)
	if !ok {
		return CompactManifest{}, ErrClientNotFound
	}
	d.tools.RegisterClient(manifest.ClientID, client)

	compact, err := BuildCompactManifest(manifest)
	if err != nil {
		return CompactManifest{}, err
	}
	return compact, nil
}

func BuildCompactManifest(manifest Manifest) (CompactManifest, error) {
	if manifest.ClientID == "" {
		return CompactManifest{}, ErrInvalidManifest
	}

	tools := make([]providers.Tool, 0, len(manifest.Tools))
	for _, tool := range manifest.Tools {
		if tool.Name == "" {
			return CompactManifest{}, ErrInvalidManifest
		}
		tools = append(tools, providers.Tool{
			Type: "function",
			Function: providers.ToolFunction{
				Name:        toolFullName(manifest.ClientID, tool.Name),
				Description: tool.Description,
			},
		})
	}

	return CompactManifest{
		ClientID: manifest.ClientID,
		Tools:    tools,
	}, nil
}
