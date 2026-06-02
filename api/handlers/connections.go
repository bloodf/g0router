package handlers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type listResponse[T any] struct {
	Data []T `json:"data"`
}

type connectionRequest struct {
	Provider             string           `json:"provider"`
	Name                 string           `json:"name"`
	AuthType             store.AuthType   `json:"auth_type"`
	AccessToken          *string          `json:"access_token"`
	RefreshToken         *string          `json:"refresh_token"`
	ExpiresAt            *int64           `json:"expires_at"`
	APIKey               *string          `json:"api_key"`
	IsActive             bool             `json:"is_active"`
	ProviderSpecificData map[string]any   `json:"provider_specific_data"`
	AccountID            *string          `json:"account_id"`
	Email                *string          `json:"email"`
	UnavailableUntil     *int64           `json:"unavailable_until"`
	BackoffLevel         int              `json:"backoff_level"`
	ModelLocks           map[string]int64 `json:"model_locks"`
}

func Connections(ctx *fasthttp.RequestCtx, s *store.Store, id string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		connections, err := listConnections(s)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list connections: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[*store.Connection]{Data: connections})
	case fasthttp.MethodPost:
		conn, ok := decodeConnectionRequest(ctx)
		if !ok {
			return
		}
		if err := s.CreateConnection(conn); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create connection: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, conn)
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "connection id required")
			return
		}
		conn, ok := decodeConnectionRequest(ctx)
		if !ok {
			return
		}
		conn.ID = id
		if err := s.UpdateConnection(conn); err != nil {
			writeStoreError(ctx, "update connection", err)
			return
		}
		got, err := s.GetConnection(id)
		if err != nil {
			writeStoreError(ctx, "get connection", err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, got)
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "connection id required")
			return
		}
		if err := s.DeleteConnection(id); err != nil {
			writeStoreError(ctx, "delete connection", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func decodeConnectionRequest(ctx *fasthttp.RequestCtx) (*store.Connection, bool) {
	var req connectionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return nil, false
	}
	return &store.Connection{
		Provider:             req.Provider,
		Name:                 req.Name,
		AuthType:             req.AuthType,
		AccessToken:          req.AccessToken,
		RefreshToken:         req.RefreshToken,
		ExpiresAt:            req.ExpiresAt,
		APIKey:               req.APIKey,
		IsActive:             req.IsActive,
		ProviderSpecificData: req.ProviderSpecificData,
		AccountID:            req.AccountID,
		Email:                req.Email,
		UnavailableUntil:     req.UnavailableUntil,
		BackoffLevel:         req.BackoffLevel,
		ModelLocks:           req.ModelLocks,
	}, true
}

func listConnections(s *store.Store) ([]*store.Connection, error) {
	var connections []*store.Connection
	for _, provider := range knownProviders() {
		providerConnections, err := s.GetConnections(provider.ID)
		if err != nil {
			return nil, fmt.Errorf("get %s connections: %w", provider.ID, err)
		}
		connections = append(connections, providerConnections...)
	}
	return connections, nil
}

func writeStoreError(ctx *fasthttp.RequestCtx, action string, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "not found")
		return
	}
	writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("%s: %v", action, err))
}
