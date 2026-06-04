package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
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
	Arguments    json.RawMessage `json:"arguments"`
	AllowedTools []string        `json:"allowed_tools"`
}

type MCPInstanceRuntime interface {
	RegisterInstance(ctx context.Context, instance *store.MCPInstance) (mcp.Manifest, error)
	CloseInstance(instanceID string) error
	ReapplyInstanceCredentials(ctx context.Context, s *store.Store, instanceID string) (mcp.Manifest, error)
}

func MCPInstances(ctx *fasthttp.RequestCtx, s *store.Store, runtime MCPInstanceRuntime, id string) {
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
		if instance.IsActive {
			if runtime == nil {
				_ = s.DeleteMCPInstance(instance.ID)
				writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp instance runtime unavailable")
				return
			}
			manifest, err := runtime.RegisterInstance(requestContext(ctx), instance)
			if err != nil {
				_ = s.DeleteMCPInstance(instance.ID)
				writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("register mcp instance: %v", err))
				return
			}
			if err := s.UpdateMCPInstanceManifest(instance.ID, manifest); err != nil {
				_ = runtime.CloseInstance(instance.ID)
				_ = s.DeleteMCPInstance(instance.ID)
				writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("cache mcp manifest: %v", err))
				return
			}
			got, err := s.GetMCPInstance(instance.ID)
			if err != nil {
				writeStoreError(ctx, "get mcp instance", err)
				return
			}
			instance = got
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
		if runtime != nil {
			_ = runtime.CloseInstance(id)
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
	resourceURI := strings.TrimSpace(req.ResourceURI)
	if resourceURI == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "resource_uri is required")
		return
	}
	authorizationURL := strings.TrimSpace(req.AuthorizationURL)
	if authorizationURL == "" {
		var err error
		authorizationURL, err = mcp.DiscoverOAuthAuthorizationURL(requestContext(ctx), http.DefaultClient, resourceURI)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadGateway, "discover mcp oauth authorization url failed")
			return
		}
	}
	flow, err := mcp.BuildOAuthStartFlow(mcp.OAuthStartConfig{
		InstanceID:        instanceID,
		AuthorizationURL:  authorizationURL,
		RedirectURI:       req.RedirectURI,
		ResourceURI:       resourceURI,
		ClientID:          req.ClientID,
		ClientSecret:      req.ClientSecret,
		ExpirationSeconds: int((10 * time.Minute).Seconds()),
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	if err := s.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
		InstanceID:         instanceID,
		State:              flow.State,
		CodeVerifierSecret: flow.CodeVerifierSecret,
		RedirectURI:        flow.RedirectURI,
		AuthorizationURL:   flow.AuthorizationURL,
		ResourceURI:        flow.ResourceURI,
		ClientID:           flow.ClientID,
		ClientSecret:       flow.ClientSecret,
		ExpiresAt:          flow.ExpiresAt,
	}); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create mcp oauth flow: %v", err))
		return
	}
	writeJSON(ctx, fasthttp.StatusCreated, mcpOAuthStartResponse{AuthorizationURL: flow.AuthorizationURL, ExpiresAt: flow.ExpiresAt.Format(time.RFC3339)})
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
		writeJSON(ctx, fasthttp.StatusOK, listResponse[*store.MCPClient]{Data: redactedMCPClients(list)})
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
		manifest, err := registerMCPClient(requestContext(ctx), clients, tools, client)
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
		writeJSON(ctx, fasthttp.StatusCreated, redactedMCPClient(got))
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
		if tools != nil {
			tools.UnregisterClient(id)
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
		allowedTools := allowedToolsFromRequest(ctx)
		compact, err := compactToolList(mcpRequestContext(ctx, allowedTools), s, tools, instanceID, accountLabel, allowedTools)
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
		allowedTools := append(allowedToolsFromRequest(ctx), req.AllowedTools...)
		result, err := tools.Call(mcpRequestContext(ctx, allowedTools), name, req.Arguments)
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

func registerMCPClient(ctx context.Context, clients *mcp.ClientManager, tools *mcp.ToolManager, client *store.MCPClient) (mcp.Manifest, error) {
	manifest, err := clients.Register(ctx, client.ClientConfig())
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

func compactToolList(ctx context.Context, s *store.Store, tools *mcp.ToolManager, instanceID, accountLabel string, allowedTools []string) ([]providers.Tool, error) {
	if tools != nil && instanceID == "" && accountLabel == "" {
		return tools.CompactToolsForRequest(ctx), nil
	}
	if s == nil {
		return nil, nil
	}

	if instanceID != "" || accountLabel != "" {
		compact, err := compactInstanceToolList(s, instanceID, accountLabel)
		if err != nil {
			return nil, err
		}
		return filterCompactTools(compact, allowedTools), nil
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
	return filterCompactTools(compact, allowedTools), nil
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

func redactedMCPClients(clients []*store.MCPClient) []*store.MCPClient {
	redacted := make([]*store.MCPClient, 0, len(clients))
	for _, client := range clients {
		redacted = append(redacted, redactedMCPClient(client))
	}
	return redacted
}

func redactedMCPClient(client *store.MCPClient) *store.MCPClient {
	redacted := *client
	redacted.Env = redactMCPSecretMap(client.Env)
	return &redacted
}

func redactMCPSecretMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	redacted := make(map[string]string, len(values))
	for key, value := range values {
		if isMCPSecretKey(key) {
			redacted[key] = mcp.RedactedValue
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func isMCPSecretKey(key string) bool {
	normalized := strings.ToLower(key)
	for _, marker := range []string{"token", "secret", "key", "authorization", "password"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
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
	case errors.Is(err, mcp.ErrInvalidToolArguments):
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
	default:
		writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("execute mcp tool: %v", err))
	}
}

func mcpRequestContext(ctx *fasthttp.RequestCtx, allowedTools []string) context.Context {
	reqCtx := requestContext(ctx)
	if len(allowedTools) == 0 {
		return reqCtx
	}
	return mcp.WithAllowedTools(reqCtx, allowedTools...)
}

func allowedToolsFromRequest(ctx *fasthttp.RequestCtx) []string {
	var allowed []string
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		switch string(key) {
		case "allowed_tool":
			allowed = appendAllowedTools(allowed, string(value))
		case "allowed_tools":
			for _, name := range strings.Split(string(value), ",") {
				allowed = appendAllowedTools(allowed, name)
			}
		}
	})
	return allowed
}

func appendAllowedTools(allowed []string, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return allowed
	}
	return append(allowed, name)
}

func filterCompactTools(tools []providers.Tool, allowedTools []string) []providers.Tool {
	if len(allowedTools) == 0 {
		return tools
	}
	allowed := make(map[string]struct{}, len(allowedTools))
	for _, name := range allowedTools {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[name] = struct{}{}
		}
	}
	filtered := make([]providers.Tool, 0, len(tools))
	for _, tool := range tools {
		if _, ok := allowed[tool.Function.Name]; ok {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}
