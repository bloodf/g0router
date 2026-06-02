package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type mcpClientRequest struct {
	Name      string            `json:"name"`
	Transport mcp.Transport     `json:"transport"`
	Command   *string           `json:"command"`
	Args      []string          `json:"args"`
	URL       *string           `json:"url"`
	Env       map[string]string `json:"env"`
	IsActive  bool              `json:"is_active"`
}

type mcpInstanceRequest struct {
	Name         string            `json:"name"`
	ServerKey    string            `json:"server_key"`
	LaunchType   mcp.LaunchType    `json:"launch_type"`
	Transport    mcp.Transport     `json:"transport"`
	Command      *string           `json:"command"`
	Args         []string          `json:"args"`
	URL          *string           `json:"url"`
	Headers      map[string]string `json:"headers"`
	Env          map[string]string `json:"env"`
	CWD          *string           `json:"cwd"`
	AccountLabel *string           `json:"account_label"`
	IsActive     bool              `json:"is_active"`
}

type mcpOAuthStartRequest struct {
	AuthorizationURL string `json:"authorization_url"`
	ResourceURI      string `json:"resource_uri"`
	RedirectURI      string `json:"redirect_uri"`
}

type mcpOAuthStartResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	ExpiresAt        string `json:"expires_at"`
}

type mcpOAuthAccountResponse struct {
	ID           string   `json:"id"`
	InstanceID   string   `json:"instance_id"`
	AccountLabel string   `json:"account_label"`
	Subject      string   `json:"subject,omitempty"`
	Email        string   `json:"email,omitempty"`
	Issuer       string   `json:"issuer,omitempty"`
	ResourceURI  string   `json:"resource_uri,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	ExpiresAt    string   `json:"expires_at,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

type mcpToolExecuteRequest struct {
	Arguments json.RawMessage `json:"arguments"`
}

func MCPInstances(ctx *fasthttp.RequestCtx, s *store.Store, id string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		instances, err := s.ListMCPInstances()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list mcp instances: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[*store.MCPInstance]{Data: instances})
	case fasthttp.MethodPost:
		instance, ok := decodeMCPInstanceRequest(ctx)
		if !ok {
			return
		}
		if err := s.CreateMCPInstance(instance); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create mcp instance: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, redactedMCPInstance(instance))
	case fasthttp.MethodDelete:
		if strings.TrimSpace(id) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "mcp instance id required")
			return
		}
		if err := s.DeleteMCPInstance(id); err != nil {
			writeStoreError(ctx, "delete mcp instance", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func MCPOAuthStart(ctx *fasthttp.RequestCtx, s *store.Store, instanceID string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if strings.TrimSpace(instanceID) == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "mcp instance id required")
		return
	}

	var req mcpOAuthStartRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(req.AuthorizationURL) == "" || strings.TrimSpace(req.ResourceURI) == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "authorization_url and resource_uri are required")
		return
	}
	state, err := randomState()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create oauth state: %v", err))
		return
	}
	expiresAt := time.Now().Add(10 * time.Minute)
	authorizationURL, err := buildAuthorizationURL(req.AuthorizationURL, req.RedirectURI, req.ResourceURI, state)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	if err := s.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
		InstanceID:         instanceID,
		State:              state,
		CodeVerifierSecret: state,
		RedirectURI:        req.RedirectURI,
		AuthorizationURL:   authorizationURL,
		ResourceURI:        req.ResourceURI,
		ExpiresAt:          expiresAt,
	}); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create mcp oauth flow: %v", err))
		return
	}
	writeJSON(ctx, fasthttp.StatusCreated, mcpOAuthStartResponse{AuthorizationURL: authorizationURL, ExpiresAt: expiresAt.Format(time.RFC3339)})
}

func MCPOAuthAccounts(ctx *fasthttp.RequestCtx, s *store.Store, instanceID string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	accounts, err := s.ListMCPOAuthAccounts(instanceID)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list mcp oauth accounts: %v", err))
		return
	}
	responses := make([]mcpOAuthAccountResponse, 0, len(accounts))
	for _, account := range accounts {
		responses = append(responses, mcpOAuthAccountResponse{
			ID:           account.ID,
			InstanceID:   account.InstanceID,
			AccountLabel: account.AccountLabel,
			Subject:      account.Subject,
			Email:        account.Email,
			Issuer:       account.Issuer,
			ResourceURI:  account.ResourceURI,
			Scopes:       append([]string(nil), account.Scopes...),
			ExpiresAt:    account.ExpiresAt.Format(time.RFC3339),
			CreatedAt:    account.CreatedAt,
			UpdatedAt:    account.UpdatedAt,
		})
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[mcpOAuthAccountResponse]{Data: responses})
}

func MCPClients(ctx *fasthttp.RequestCtx, s *store.Store, clients *mcp.ClientManager, tools *mcp.ToolManager, id string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		list, err := s.ListMCPClients()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list mcp clients: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[*store.MCPClient]{Data: list})
	case fasthttp.MethodPost:
		if clients == nil || tools == nil {
			writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp runtime unavailable")
			return
		}
		client, ok := decodeMCPClientRequest(ctx)
		if !ok {
			return
		}
		if err := s.CreateMCPClient(client); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create mcp client: %v", err))
			return
		}
		manifest, err := registerMCPClient(clients, tools, client)
		if err != nil {
			_ = s.DeleteMCPClient(client.ID)
			writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("register mcp client: %v", err))
			return
		}
		if err := s.UpdateMCPClientManifest(client.ID, manifest); err != nil {
			_ = clients.Close(client.ID)
			_ = s.DeleteMCPClient(client.ID)
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("cache mcp manifest: %v", err))
			return
		}
		got, err := s.GetMCPClient(client.ID)
		if err != nil {
			writeStoreError(ctx, "get mcp client", err)
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, got)
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "mcp client id required")
			return
		}
		if clients != nil {
			if err := clients.Close(id); err != nil && !errors.Is(err, mcp.ErrClientNotFound) {
				writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("close mcp client: %v", err))
				return
			}
		}
		if err := s.DeleteMCPClient(id); err != nil {
			writeStoreError(ctx, "delete mcp client", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func MCPTools(ctx *fasthttp.RequestCtx, s *store.Store, tools *mcp.ToolManager, name string) {
	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		instanceID := string(ctx.QueryArgs().Peek("instance_id"))
		accountLabel := string(ctx.QueryArgs().Peek("account_label"))
		compact, err := compactToolList(s, tools, instanceID, accountLabel)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list mcp tools: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[providers.Tool]{Data: compact})
	case fasthttp.MethodPost:
		if tools == nil {
			writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp tools unavailable")
			return
		}
		if name == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "mcp tool name required")
			return
		}
		req, ok := decodeMCPToolExecuteRequest(ctx)
		if !ok {
			return
		}
		result, err := tools.Call(context.Background(), name, req.Arguments)
		if err != nil {
			writeMCPToolError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, result)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func decodeMCPInstanceRequest(ctx *fasthttp.RequestCtx) (*store.MCPInstance, bool) {
	var req mcpInstanceRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return nil, false
	}
	return &store.MCPInstance{
		Name:         req.Name,
		ServerKey:    req.ServerKey,
		LaunchType:   req.LaunchType,
		Transport:    req.Transport,
		Command:      req.Command,
		Args:         req.Args,
		URL:          req.URL,
		Headers:      req.Headers,
		Env:          req.Env,
		CWD:          req.CWD,
		AccountLabel: req.AccountLabel,
		IsActive:     req.IsActive,
	}, true
}

func decodeMCPClientRequest(ctx *fasthttp.RequestCtx) (*store.MCPClient, bool) {
	var req mcpClientRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return nil, false
	}
	return &store.MCPClient{
		Name:      req.Name,
		Transport: req.Transport,
		Command:   req.Command,
		Args:      req.Args,
		URL:       req.URL,
		Env:       req.Env,
		IsActive:  req.IsActive,
	}, true
}

func decodeMCPToolExecuteRequest(ctx *fasthttp.RequestCtx) (mcpToolExecuteRequest, bool) {
	var req mcpToolExecuteRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return mcpToolExecuteRequest{}, false
	}
	if len(req.Arguments) == 0 {
		req.Arguments = json.RawMessage(`{}`)
	}
	return req, true
}

func registerMCPClient(clients *mcp.ClientManager, tools *mcp.ToolManager, client *store.MCPClient) (mcp.Manifest, error) {
	manifest, err := clients.Register(context.Background(), client.ClientConfig())
	if err != nil {
		return mcp.Manifest{}, err
	}
	if err := tools.RegisterManifest(manifest); err != nil {
		_ = clients.Close(client.ID)
		return mcp.Manifest{}, err
	}
	registered, ok := clients.Client(client.ID)
	if !ok {
		return mcp.Manifest{}, mcp.ErrClientNotFound
	}
	tools.RegisterClient(client.ID, registered)
	return manifest, nil
}

func compactToolList(s *store.Store, tools *mcp.ToolManager, instanceID, accountLabel string) ([]providers.Tool, error) {
	if tools != nil && instanceID == "" && accountLabel == "" {
		return tools.CompactTools(), nil
	}
	if s == nil {
		return nil, nil
	}

	if instanceID != "" || accountLabel != "" {
		return compactInstanceToolList(s, instanceID, accountLabel)
	}

	clients, err := s.ListMCPClients()
	if err != nil {
		return nil, err
	}
	var compact []providers.Tool
	for _, client := range clients {
		if client.ToolManifest == nil {
			continue
		}
		manifest, err := mcp.BuildCompactManifest(*client.ToolManifest)
		if err != nil {
			return nil, err
		}
		compact = append(compact, manifest.Tools...)
	}

	instances, err := compactInstanceToolList(s, "", "")
	if err != nil {
		return nil, err
	}
	compact = append(compact, instances...)
	return compact, nil
}

func compactInstanceToolList(s *store.Store, instanceID, accountLabel string) ([]providers.Tool, error) {
	instances, err := s.ListMCPInstances()
	if err != nil {
		return nil, err
	}
	var compact []providers.Tool
	for _, instance := range instances {
		if instanceID != "" && instance.ID != instanceID {
			continue
		}
		if accountLabel != "" && stringValue(instance.AccountLabel) != accountLabel {
			continue
		}
		if instance.ToolManifest == nil {
			continue
		}
		manifest, err := mcp.BuildCompactManifest(*instance.ToolManifest)
		if err != nil {
			return nil, err
		}
		compact = append(compact, manifest.Tools...)
	}
	return compact, nil
}

func redactedMCPInstance(instance *store.MCPInstance) *store.MCPInstance {
	cfg := instance.Config().Redacted()
	redacted := *instance
	redacted.Env = cfg.Env
	redacted.Headers = cfg.Headers
	return &redacted
}

func buildAuthorizationURL(rawURL, redirectURI, resourceURI, state string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse authorization_url: %w", err)
	}
	values := parsed.Query()
	values.Set("state", state)
	values.Set("resource", resourceURI)
	if redirectURI != "" {
		values.Set("redirect_uri", redirectURI)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func randomState() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func writeMCPToolError(ctx *fasthttp.RequestCtx, err error) {
	switch {
	case errors.Is(err, mcp.ErrToolNotFound), errors.Is(err, mcp.ErrClientNotFound):
		writeError(ctx, fasthttp.StatusNotFound, err.Error())
	default:
		writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("execute mcp tool: %v", err))
	}
}
