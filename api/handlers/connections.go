package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	providerids "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type listResponse[T any] struct {
	Data []T `json:"data"`
}

type connectionResponse struct {
	ID                   string
	Provider             string
	Name                 string
	AuthType             store.AuthType
	ExpiresAt            *int64
	IsActive             bool
	ProviderSpecificData map[string]any
	AccountID            *string
	Email                *string
	UnavailableUntil     *int64
	BackoffLevel         int
	ModelLocks           map[string]int64
	NeedsReauth          bool    `json:"needs_reauth"`
	LastRefreshError     *string `json:"last_refresh_error,omitempty"`
	CreatedAt            string
	UpdatedAt            string
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
			log.Printf("list connections: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to list connections")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[connectionResponse]{Data: redactConnections(connections)})
	case fasthttp.MethodPost:
		conn, ok := decodeConnectionRequest(ctx)
		if !ok {
			return
		}
		if err := s.CreateConnection(conn); err != nil {
			log.Printf("create connection: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create connection")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, redactConnection(conn))
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
		writeJSON(ctx, fasthttp.StatusOK, redactConnection(got))
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

func ConnectionTest(ctx *fasthttp.RequestCtx, s *store.Store, id string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "connection id required")
		return
	}
	conn, err := s.GetConnection(id)
	if err != nil {
		writeStoreError(ctx, "get connection", err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"ok":       conn.IsActive,
		"provider": conn.Provider,
		"name":     conn.Name,
	})
}

func decodeConnectionRequest(ctx *fasthttp.RequestCtx) (*store.Connection, bool) {
	var req connectionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return nil, false
	}
	return &store.Connection{
		Provider:             providerids.CanonicalProviderID(req.Provider),
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

func redactConnections(connections []*store.Connection) []connectionResponse {
	responses := make([]connectionResponse, 0, len(connections))
	for _, conn := range connections {
		responses = append(responses, redactConnection(conn))
	}
	return responses
}

func redactConnection(conn *store.Connection) connectionResponse {
	return connectionResponse{
		ID:                   conn.ID,
		Provider:             conn.Provider,
		Name:                 conn.Name,
		AuthType:             conn.AuthType,
		ExpiresAt:            conn.ExpiresAt,
		IsActive:             conn.IsActive,
		ProviderSpecificData: redactProviderSpecificData(conn.ProviderSpecificData),
		AccountID:            conn.AccountID,
		Email:                conn.Email,
		UnavailableUntil:     conn.UnavailableUntil,
		BackoffLevel:         conn.BackoffLevel,
		ModelLocks:           conn.ModelLocks,
		NeedsReauth:          conn.NeedsReauth,
		LastRefreshError:     conn.LastRefreshError,
		CreatedAt:            conn.CreatedAt,
		UpdatedAt:            conn.UpdatedAt,
	}
}

func redactProviderSpecificData(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	redacted := make(map[string]any, len(values))
	for key, value := range values {
		if isConnectionSecretKey(key) {
			continue
		}
		redacted[key] = redactProviderSpecificValue(value)
	}
	return redacted
}

func redactProviderSpecificValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return redactProviderSpecificData(typed)
	case map[string]string:
		redacted := make(map[string]string, len(typed))
		for key, value := range typed {
			if isConnectionSecretKey(key) {
				continue
			}
			redacted[key] = value
		}
		return redacted
	case []any:
		redacted := make([]any, 0, len(typed))
		for _, item := range typed {
			redacted = append(redacted, redactProviderSpecificValue(item))
		}
		return redacted
	default:
		return value
	}
}

func isConnectionSecretKey(key string) bool {
	normalized := strings.ToLower(key)
	for _, marker := range []string{"token", "secret", "key", "authorization", "password"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func listConnections(s *store.Store) ([]*store.Connection, error) {
	connections, err := s.ListConnections()
	if err != nil {
		return nil, err
	}
	return connections, nil
}

func writeStoreError(ctx *fasthttp.RequestCtx, action string, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "not found")
		return
	}
	log.Printf("%s: %v", action, err)
	writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("failed to %s", action))
}
