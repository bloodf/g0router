package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
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

type connectionStore interface {
	ListConnections() ([]*store.Connection, error)
	CreateConnection(*store.Connection) error
	UpdateConnection(*store.Connection) error
	GetConnection(string) (*store.Connection, error)
	DeleteConnection(string) error
	BulkDisableConnectionsByThreshold(thresholdPercent int) ([]string, error)
	BulkEnableConnectionsWithQuota() ([]string, error)
}

func Connections(ctx *fasthttp.RequestCtx, s connectionStore, id string) {
	if isStoreNil(s) {
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

type bulkDisableRequest struct {
	ThresholdPercent int `json:"threshold_percent"`
}

type bulkActionResponse struct {
	Affected []string `json:"affected"`
}

func ConnectionsBulkDisable(ctx *fasthttp.RequestCtx, s connectionStore, audit auditWriter) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	var req bulkDisableRequest
	if len(ctx.PostBody()) > 0 {
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
	}

	threshold := req.ThresholdPercent
	if threshold == 0 {
		threshold = 5
	}
	if threshold < 0 || threshold > 100 {
		writeError(ctx, fasthttp.StatusBadRequest, "threshold_percent must be between 0 and 100")
		return
	}

	affected, err := s.BulkDisableConnectionsByThreshold(threshold)
	if err != nil {
		log.Printf("bulk disable connections: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to disable connections")
		return
	}

	if audit != nil && len(affected) > 0 {
		if err := audit.AppendAudit(store.AuditEntry{
			Action:  "connection.bulk_disable",
			Details: fmt.Sprintf("threshold=%d affected=%v", threshold, affected),
		}); err != nil {
			log.Printf("append audit: %v", err)
		}
	}

	writeJSON(ctx, fasthttp.StatusOK, bulkActionResponse{Affected: affected})
}

func ConnectionsBulkEnable(ctx *fasthttp.RequestCtx, s connectionStore, audit auditWriter) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	affected, err := s.BulkEnableConnectionsWithQuota()
	if err != nil {
		log.Printf("bulk enable connections: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to enable connections")
		return
	}

	if audit != nil && len(affected) > 0 {
		if err := audit.AppendAudit(store.AuditEntry{
			Action:  "connection.bulk_enable",
			Details: fmt.Sprintf("affected=%v", affected),
		}); err != nil {
			log.Printf("append audit: %v", err)
		}
	}

	writeJSON(ctx, fasthttp.StatusOK, bulkActionResponse{Affected: affected})
}

func ConnectionTest(ctx *fasthttp.RequestCtx, s connectionStore, id string) {
	if isStoreNil(s) {
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

func ProviderConnections(ctx *fasthttp.RequestCtx, s connectionStore, providerID string) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	providerID = providerids.CanonicalProviderID(providerID)
	_, ok := providerids.ProviderMatrix().Provider(providerID)
	if !ok {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}

	connections, err := s.ListConnections()
	if err != nil {
		log.Printf("list connections: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list connections")
		return
	}

	filtered := make([]*store.Connection, 0)
	for _, conn := range connections {
		if conn.Provider == providerID {
			filtered = append(filtered, conn)
		}
	}

	writeJSON(ctx, fasthttp.StatusOK, listResponse[connectionResponse]{Data: redactConnections(filtered)})
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

func listConnections(s connectionStore) ([]*store.Connection, error) {
	connections, err := s.ListConnections()
	if err != nil {
		return nil, err
	}
	return connections, nil
}

func isStoreNil(s interface{}) bool {
	if s == nil {
		return true
	}
	rv := reflect.ValueOf(s)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

func writeStoreError(ctx *fasthttp.RequestCtx, action string, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "not found")
		return
	}
	log.Printf("%s: %v", action, err)
	writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("failed to %s", action))
}
