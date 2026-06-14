package admin

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type alertChannelDTO struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	ChannelType string         `json:"channel_type"`
	Config      map[string]any `json:"config"`
	Events      []string       `json:"events"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   string         `json:"created_at"`
}

func toAlertChannelDTO(c *store.AlertChannel) alertChannelDTO {
	config := c.Config
	if config == nil {
		config = map[string]any{}
	}
	events := c.Events
	if events == nil {
		events = []string{}
	}
	return alertChannelDTO{
		ID:          c.ID,
		Name:        c.Name,
		ChannelType: c.ChannelType,
		Config:      config,
		Events:      events,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
	}
}

type alertChannelRequest struct {
	Name        string         `json:"name"`
	ChannelType string         `json:"channel_type"`
	Config      map[string]any `json:"config"`
	Events      []string       `json:"events"`
	IsActive    *bool          `json:"is_active"`
}

func (r *alertChannelRequest) channelType() string {
	if r.ChannelType == "" {
		return "webhook"
	}
	return r.ChannelType
}

func (r *alertChannelRequest) isActive() bool {
	if r.IsActive == nil {
		return true
	}
	return *r.IsActive
}

// alertDispatcher builds the alert dispatcher with the production HTTP sender.
// No New() signature change and no new global state (the auditService accessor
// precedent).
func (h *Handlers) alertDispatcher() *governance.AlertDispatcher {
	return governance.NewAlertDispatcher(governance.NewHTTPSender())
}

// ListAlertChannels handles GET /api/alert-channels. The response data is a bare array.
func (h *Handlers) ListAlertChannels(ctx *fasthttp.RequestCtx) {
	channels, err := h.store.ListAlertChannels()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list alert channels")
		return
	}
	out := make([]alertChannelDTO, 0, len(channels))
	for _, c := range channels {
		out = append(out, toAlertChannelDTO(c))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateAlertChannel handles POST /api/alert-channels.
func (h *Handlers) CreateAlertChannel(ctx *fasthttp.RequestCtx) {
	var req alertChannelRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}

	created, err := h.store.CreateAlertChannel(&store.AlertChannel{
		Name:        req.Name,
		ChannelType: req.channelType(),
		Config:      req.Config,
		Events:      req.Events,
		IsActive:    req.isActive(),
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create alert channel")
		return
	}
	h.recordAudit(ctx, "alert_channel.create", created.Name, "Created alert channel "+created.Name)
	writeData(ctx, fasthttp.StatusCreated, toAlertChannelDTO(created))
}

// GetAlertChannel handles GET /api/alert-channels/{id}.
func (h *Handlers) GetAlertChannel(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	c, err := h.store.GetAlertChannelByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load alert channel")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toAlertChannelDTO(c))
}

// UpdateAlertChannel handles PUT /api/alert-channels/{id}.
func (h *Handlers) UpdateAlertChannel(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req alertChannelRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}

	updated, err := h.store.UpdateAlertChannel(id, &store.AlertChannel{
		Name:        req.Name,
		ChannelType: req.channelType(),
		Config:      req.Config,
		Events:      req.Events,
		IsActive:    req.isActive(),
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update alert channel")
		return
	}
	h.recordAudit(ctx, "alert_channel.update", updated.Name, "Updated alert channel "+updated.Name)
	writeData(ctx, fasthttp.StatusOK, toAlertChannelDTO(updated))
}

// DeleteAlertChannel handles DELETE /api/alert-channels/{id}.
func (h *Handlers) DeleteAlertChannel(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.store.GetAlertChannelByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load alert channel")
		return
	}
	if err := h.store.DeleteAlertChannel(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete alert channel")
		return
	}
	h.recordAudit(ctx, "alert_channel.delete", existing.Name, "Deleted alert channel "+existing.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Alert channel deleted successfully"})
}

// TestAlertChannel handles POST /api/alert-channels/{id}/test. It sends a
// best-effort test notification through the dispatcher and returns {ok, message}.
// The response never echoes the channel's secret config.
func (h *Handlers) TestAlertChannel(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	c, err := h.store.GetAlertChannelByID(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load alert channel")
		return
	}

	// Use a standalone context for the outbound send: the dispatcher owns its own
	// timeout and the delivery is decoupled from the inbound request lifecycle.
	sent, message := h.alertDispatcher().Dispatch(context.Background(), c)
	h.recordAudit(ctx, "alert_channel.test", c.Name, "Sent test notification to "+c.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"ok": sent, "message": message})
}
