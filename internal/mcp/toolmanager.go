package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/bloodf/g0router/internal/providers"
)

const toolNameSeparator = "__"

var (
	ErrInvalidManifest       = errors.New("mcp: invalid manifest")
	ErrToolAlreadyRegistered = errors.New("mcp: tool already registered")
	ErrToolNotFound          = errors.New("mcp: tool not found")
	ErrInvalidToolArguments  = errors.New("mcp: invalid tool arguments")
)

type allowedToolsContextKey struct{}

type ToolManager struct {
	mu      sync.RWMutex
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

func WithAllowedTools(ctx context.Context, names ...string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	if len(allowed) == 0 {
		return ctx
	}
	return context.WithValue(ctx, allowedToolsContextKey{}, allowed)
}

func (m *ToolManager) RegisterManifest(manifest Manifest) error {
	if manifest.ClientID == "" {
		return ErrInvalidManifest
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	seen := make(map[string]struct{}, len(manifest.Tools))
	for _, tool := range manifest.Tools {
		if tool.Name == "" {
			return ErrInvalidManifest
		}
		fullName := toolFullName(manifest.ClientID, tool.Name)
		if _, ok := seen[fullName]; ok {
			return ErrToolAlreadyRegistered
		}
		seen[fullName] = struct{}{}
		if _, ok := m.tools[fullName]; ok {
			return ErrToolAlreadyRegistered
		}
	}

	for _, tool := range manifest.Tools {
		fullName := toolFullName(manifest.ClientID, tool.Name)
		tool.ClientID = manifest.ClientID
		m.tools[fullName] = cloneTool(tool)
		m.order = append(m.order, fullName)
	}
	return nil
}

func (m *ToolManager) CompactTools() []providers.Tool {
	return m.CompactToolsForRequest(context.Background())
}

func (m *ToolManager) CompactToolsForRequest(ctx context.Context) []providers.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allowed, filtered := allowedToolsFromContext(ctx)
	tools := make([]providers.Tool, 0, len(m.order))
	for _, fullName := range m.order {
		if filtered && !toolAllowed(allowed, fullName) {
			continue
		}
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
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, ok := m.tools[name]
	if !ok {
		return Tool{}, ErrToolNotFound
	}
	return cloneTool(tool), nil
}

func (m *ToolManager) RegisterClient(clientID string, client Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[clientID] = client
}

func (m *ToolManager) UnregisterClient(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clients, clientID)
	prefix := clientID + toolNameSeparator
	for name := range m.tools {
		if strings.HasPrefix(name, prefix) {
			delete(m.tools, name)
		}
	}
	order := m.order[:0]
	for _, name := range m.order {
		if !strings.HasPrefix(name, prefix) {
			order = append(order, name)
		}
	}
	m.order = order
}

func (m *ToolManager) Call(ctx context.Context, name string, arguments json.RawMessage) (CallResult, error) {
	m.mu.RLock()
	allowed, filtered := allowedToolsFromContext(ctx)
	if filtered && !toolAllowed(allowed, name) {
		m.mu.RUnlock()
		return CallResult{}, ErrToolNotFound
	}

	tool, ok := m.tools[name]
	if !ok {
		m.mu.RUnlock()
		return CallResult{}, ErrToolNotFound
	}
	tool = cloneTool(tool)

	client, ok := m.clients[tool.ClientID]
	if !ok {
		m.mu.RUnlock()
		return CallResult{}, ErrClientNotFound
	}
	m.mu.RUnlock()

	callArguments := append(json.RawMessage(nil), arguments...)
	if len(bytes.TrimSpace(tool.InputSchema)) > 0 {
		normalized := normalizedToolArguments(arguments)
		if err := validateToolArguments(tool.InputSchema, normalized); err != nil {
			return CallResult{}, err
		}
		callArguments = normalized
	}

	result, err := client.CallTool(ctx, CallRequest{Name: tool.Name, Arguments: callArguments})
	if err != nil {
		return CallResult{}, fmt.Errorf("call mcp tool %q: %w", name, err)
	}
	return result, nil
}

func toolFullName(clientID, toolName string) string {
	return clientID + toolNameSeparator + toolName
}

func allowedToolsFromContext(ctx context.Context) (map[string]struct{}, bool) {
	if ctx == nil {
		return nil, false
	}
	allowed, ok := ctx.Value(allowedToolsContextKey{}).(map[string]struct{})
	return allowed, ok
}

func toolAllowed(allowed map[string]struct{}, name string) bool {
	_, ok := allowed[name]
	return ok
}

func cloneTool(tool Tool) Tool {
	tool.InputSchema = append(json.RawMessage(nil), tool.InputSchema...)
	return tool
}

func normalizedToolArguments(arguments json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(arguments)
	if len(trimmed) == 0 {
		return json.RawMessage(`{}`)
	}
	return append(json.RawMessage(nil), trimmed...)
}

func validateToolArguments(schema json.RawMessage, arguments json.RawMessage) error {
	schemaObject, err := decodeSchemaObject(schema)
	if err != nil {
		return fmt.Errorf("%w: schema: %w", ErrInvalidToolArguments, err)
	}
	var value any
	if err := json.Unmarshal(arguments, &value); err != nil {
		return fmt.Errorf("%w: decode arguments: %w", ErrInvalidToolArguments, err)
	}
	if err := validateSchemaValue(value, schemaObject, "arguments"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidToolArguments, err)
	}
	return nil
}

func decodeSchemaObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func validateSchemaValue(value any, schema map[string]json.RawMessage, path string) error {
	if typ := schemaType(schema["type"]); typ != "" && !matchesJSONType(value, typ) {
		return fmt.Errorf("%s must be %s", path, typ)
	}

	object, isObject := value.(map[string]any)
	if !isObject {
		return nil
	}
	if err := validateRequiredProperties(object, schema["required"], path); err != nil {
		return err
	}
	properties, err := decodePropertySchemas(schema["properties"])
	if err != nil {
		return err
	}
	for name, propertySchema := range properties {
		if propertyValue, ok := object[name]; ok {
			if err := validateSchemaValue(propertyValue, propertySchema, path+"."+name); err != nil {
				return err
			}
		}
	}
	if additionalPropertiesFalse(schema["additionalProperties"]) {
		for name := range object {
			if _, ok := properties[name]; !ok {
				return fmt.Errorf("%s.%s is not allowed", path, name)
			}
		}
	}
	return nil
}

func validateRequiredProperties(object map[string]any, raw json.RawMessage, path string) error {
	var required []string
	if err := json.Unmarshal(raw, &required); err != nil {
		return nil
	}
	for _, name := range required {
		if _, ok := object[name]; !ok {
			return fmt.Errorf("%s.%s is required", path, name)
		}
	}
	return nil
}

func decodePropertySchemas(raw json.RawMessage) (map[string]map[string]json.RawMessage, error) {
	var rawProperties map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawProperties); err != nil {
		return nil, nil
	}
	properties := make(map[string]map[string]json.RawMessage, len(rawProperties))
	for name, rawSchema := range rawProperties {
		var schema map[string]json.RawMessage
		if err := json.Unmarshal(rawSchema, &schema); err != nil {
			return nil, err
		}
		properties[name] = schema
	}
	return properties, nil
}

func schemaType(raw json.RawMessage) string {
	var typ string
	_ = json.Unmarshal(raw, &typ)
	return typ
}

func matchesJSONType(value any, typ string) bool {
	switch typ {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		number, ok := value.(float64)
		if !ok {
			return false
		}
		return math.Trunc(number) == number
	case "null":
		return value == nil
	default:
		return true
	}
}

func additionalPropertiesFalse(raw json.RawMessage) bool {
	var additional bool
	if err := json.Unmarshal(raw, &additional); err != nil {
		return false
	}
	return !additional
}
