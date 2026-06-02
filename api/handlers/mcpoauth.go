package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bloodf/g0router/internal/mcp"
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

func MCPOAuthCallback(ctx *fasthttp.RequestCtx, completer MCPOAuthCompleter) {
	instanceID := strings.TrimSpace(string(ctx.QueryArgs().Peek("instance_id")))
	if instanceID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "instance_id is required")
		return
	}
	callbackURL := string(ctx.URI().FullURI())
	if callbackURL == "" {
		callbackURL = string(ctx.URI().RequestURI())
	}
	completeMCPOAuth(ctx, completer, instanceID, callbackURL)
}

func MCPOAuthComplete(ctx *fasthttp.RequestCtx, completer MCPOAuthCompleter, instanceID string) {
	if strings.TrimSpace(instanceID) == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "instance id is required")
		return
	}

	var req mcpOAuthCompleteRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	completeMCPOAuth(ctx, completer, instanceID, req.CallbackURL)
}

func completeMCPOAuth(ctx *fasthttp.RequestCtx, completer MCPOAuthCompleter, instanceID, callbackURL string) {
	if completer == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp oauth unavailable")
		return
	}
	if err := validateCallbackURL(callbackURL); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	account, err := completer.CompleteCallback(context.Background(), instanceID, callbackURL)
	if err != nil {
		if errors.Is(err, mcp.ErrOAuthFlowNotFound) {
			writeError(ctx, fasthttp.StatusNotFound, "mcp oauth flow not found")
			return
		}
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("complete mcp oauth: %v", err))
		return
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
