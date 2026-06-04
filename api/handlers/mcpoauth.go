package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type MCPOAuthCompleter interface {
	CompleteCallback(ctx context.Context, instanceID, callbackURL string) (mcp.OAuthAccount, error)
}

type mcpOAuthCompleteRequest struct {
	CallbackURL string `json:"callback_url"`
}

type mcpOAuthCompleteResponse struct {
	InstanceID   string `json:"instance_id"`
	AccountLabel string `json:"account_label"`
}

func MCPOAuthCallback(ctx *fasthttp.RequestCtx, completer MCPOAuthCompleter, runtime MCPInstanceRuntime, s *store.Store) {
	instanceID := decodeCallbackInstanceID(strings.TrimSpace(string(ctx.QueryArgs().Peek("instance_id"))))
	if instanceID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "instance_id is required")
		return
	}
	callbackURL := string(ctx.URI().FullURI())
	if callbackURL == "" {
		callbackURL = string(ctx.URI().RequestURI())
	}
	completeMCPOAuth(ctx, completer, runtime, s, instanceID, callbackURL)
}

func MCPOAuthComplete(ctx *fasthttp.RequestCtx, completer MCPOAuthCompleter, runtime MCPInstanceRuntime, s *store.Store, instanceID string) {
	if strings.TrimSpace(instanceID) == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "instance id is required")
		return
	}

	var req mcpOAuthCompleteRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	completeMCPOAuth(ctx, completer, runtime, s, instanceID, req.CallbackURL)
}

func completeMCPOAuth(ctx *fasthttp.RequestCtx, completer MCPOAuthCompleter, runtime MCPInstanceRuntime, s *store.Store, instanceID, callbackURL string) {
	if completer == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp oauth unavailable")
		return
	}
	if err := validateCallbackURL(callbackURL); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	reqCtx := requestContext(ctx)
	account, err := completer.CompleteCallback(reqCtx, instanceID, callbackURL)
	if err != nil {
		if errors.Is(err, mcp.ErrOAuthFlowNotFound) {
			writeError(ctx, fasthttp.StatusNotFound, "mcp oauth flow not found")
			return
		}
		writeError(ctx, fasthttp.StatusBadGateway, "mcp oauth exchange failed")
		return
	}
	if runtime != nil {
		manifest, err := runtime.ReapplyInstanceCredentials(reqCtx, s, instanceID)
		if err != nil {
			if s != nil {
				_ = s.UpdateMCPInstanceHealth(instanceID, "unhealthy")
			}
			writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("reapply mcp credentials: %v", err))
			return
		}
		if s != nil {
			if err := s.UpdateMCPInstanceManifest(instanceID, manifest); err != nil {
				writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("cache mcp manifest: %v", err))
				return
			}
			_ = s.UpdateMCPInstanceHealth(instanceID, "healthy")
		}
	}
	writeJSON(ctx, fasthttp.StatusOK, mcpOAuthCompleteResponse{
		InstanceID:   account.InstanceID,
		AccountLabel: account.AccountLabel,
	})
}

func validateCallbackURL(callbackURL string) error {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return fmt.Errorf("parse callback url: %w", err)
	}
	if parsed.Query().Get("code") == "" {
		return fmt.Errorf("code is required")
	}
	if parsed.Query().Get("state") == "" {
		return fmt.Errorf("state is required")
	}
	return nil
}

func decodeCallbackInstanceID(value string) string {
	if !strings.HasPrefix(value, "b64:") {
		return value
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "b64:"))
	if err != nil {
		return ""
	}
	return string(decoded)
}
