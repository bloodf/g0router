package admin

import (
	"encoding/json"
	"errors"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// connectionDTO masks secrets: it only reports whether each secret is set.
type connectionDTO struct {
	ID              string `json:"id"`
	ProviderID      string `json:"provider_id"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	SecretSet       bool   `json:"secret_set"`
	AccessTokenSet  bool   `json:"access_token_set"`
	RefreshTokenSet bool   `json:"refresh_token_set"`
	ExpiresAt       int64  `json:"expires_at"`
	Metadata        string `json:"metadata"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type connectionRequest struct {
	ProviderID   string `json:"provider_id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Secret       string `json:"secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	Metadata     string `json:"metadata"`
}

func toConnectionDTO(c *store.Connection) connectionDTO {
	return connectionDTO{
		ID:              c.ID,
		ProviderID:      c.ProviderID,
		Name:            c.Name,
		Kind:            c.Kind,
		SecretSet:       c.Secret != "",
		AccessTokenSet:  c.AccessToken != "",
		RefreshTokenSet: c.RefreshToken != "",
		ExpiresAt:       c.ExpiresAt,
		Metadata:        c.Metadata,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

// ListConnections handles GET /api/connections.
func (h *Handlers) ListConnections(ctx *fasthttp.RequestCtx) {
	connections, err := h.store.ListConnections()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list connections")
		return
	}
	out := make([]connectionDTO, 0, len(connections))
	for _, c := range connections {
		out = append(out, toConnectionDTO(c))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateConnection handles POST /api/connections.
func (h *Handlers) CreateConnection(ctx *fasthttp.RequestCtx) {
	var req connectionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ProviderID == "" || req.Kind == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider_id and kind are required")
		return
	}
	if _, err := h.store.GetProvider(req.ProviderID); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusBadRequest, "unknown provider_id")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load provider")
		return
	}

	conn := &store.Connection{
		ProviderID:   req.ProviderID,
		Name:         req.Name,
		Kind:         req.Kind,
		Secret:       req.Secret,
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken,
		ExpiresAt:    req.ExpiresAt,
		Metadata:     req.Metadata,
	}
	if err := h.store.CreateConnection(conn); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create connection")
		return
	}
	writeData(ctx, fasthttp.StatusCreated, toConnectionDTO(conn))
}

// UpdateConnection handles PUT /api/connections/{id}. Empty secret fields
// in the request preserve the stored values (so the UI never needs to echo
// secrets back).
func (h *Handlers) UpdateConnection(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req connectionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}

	existing, err := h.store.GetConnection(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "connection not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load connection")
		return
	}

	if req.ProviderID != "" {
		existing.ProviderID = req.ProviderID
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Kind != "" {
		existing.Kind = req.Kind
	}
	if req.Secret != "" {
		existing.Secret = req.Secret
	}
	if req.AccessToken != "" {
		existing.AccessToken = req.AccessToken
	}
	if req.RefreshToken != "" {
		existing.RefreshToken = req.RefreshToken
	}
	if req.ExpiresAt != 0 {
		existing.ExpiresAt = req.ExpiresAt
	}
	if req.Metadata != "" {
		existing.Metadata = req.Metadata
	}

	if err := h.store.UpdateConnection(existing); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update connection")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toConnectionDTO(existing))
}

// DeleteConnection handles DELETE /api/connections/{id}.
func (h *Handlers) DeleteConnection(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	err := h.store.DeleteConnection(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "connection not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete connection")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"deleted": true})
}
