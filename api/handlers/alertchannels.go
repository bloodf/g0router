package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/bloodf/g0router/internal/alerts"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type createAlertChannelRequest struct {
	Name        string         `json:"name"`
	ChannelType string         `json:"channel_type"`
	Config      map[string]any `json:"config"`
	Events      []string       `json:"events"`
	IsActive    bool           `json:"is_active"`
}

type updateAlertChannelRequest struct {
	Name        string         `json:"name"`
	ChannelType string         `json:"channel_type"`
	Config      map[string]any `json:"config"`
	Events      []string       `json:"events"`
	IsActive    bool           `json:"is_active"`
}

type alertChannelView struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	ChannelType string         `json:"channel_type"`
	Config      map[string]any `json:"config"`
	Events      []string       `json:"events"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   string         `json:"created_at"`
}

func newAlertChannelView(ch store.AlertChannel) alertChannelView {
	var config map[string]any
	if ch.Config != "" {
		_ = json.Unmarshal([]byte(ch.Config), &config)
	}
	return alertChannelView{
		ID:          ch.ID,
		Name:        ch.Name,
		ChannelType: ch.ChannelType,
		Config:      config,
		Events:      ch.Events,
		IsActive:    ch.IsActive,
		CreatedAt:   ch.CreatedAt,
	}
}

type alertChannelStore interface {
	ListAlertChannels() ([]store.AlertChannel, error)
	CreateAlertChannel(name, channelType, config string, events []string, isActive bool) (*store.AlertChannel, error)
	GetAlertChannel(id int64) (*store.AlertChannel, error)
	UpdateAlertChannel(id int64, name, channelType, config string, events []string, isActive bool) error
	DeleteAlertChannel(id int64) error
}

func AlertChannels(ctx *fasthttp.RequestCtx, s alertChannelStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			channels, err := s.ListAlertChannels()
			if err != nil {
				log.Printf("list alert channels: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list alert channels")
				return
			}
			views := make([]alertChannelView, 0, len(channels))
			for _, ch := range channels {
				views = append(views, newAlertChannelView(ch))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[alertChannelView]{Data: views})
			return
		}
		channelID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid alert channel id")
			return
		}
		ch, err := s.GetAlertChannel(channelID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newAlertChannelView(*ch))
	case fasthttp.MethodPost:
		var req createAlertChannelRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name is required")
			return
		}
		if strings.TrimSpace(req.ChannelType) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "channel_type is required")
			return
		}
		configJSON, _ := json.Marshal(req.Config)
		ch, err := s.CreateAlertChannel(req.Name, req.ChannelType, string(configJSON), req.Events, req.IsActive)
		if err != nil {
			log.Printf("create alert channel: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create alert channel")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, newAlertChannelView(*ch))
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "alert channel id required")
			return
		}
		channelID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid alert channel id")
			return
		}
		var req updateAlertChannelRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name is required")
			return
		}
		if strings.TrimSpace(req.ChannelType) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "channel_type is required")
			return
		}
		configJSON, _ := json.Marshal(req.Config)
		if err := s.UpdateAlertChannel(channelID, req.Name, req.ChannelType, string(configJSON), req.Events, req.IsActive); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
				return
			}
			log.Printf("update alert channel: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update alert channel")
			return
		}
		updated, err := s.GetAlertChannel(channelID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newAlertChannelView(*updated))
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "alert channel id required")
			return
		}
		channelID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid alert channel id")
			return
		}
		if err := s.DeleteAlertChannel(channelID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
				return
			}
			log.Printf("delete alert channel: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete alert channel")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func AlertChannelsTest(ctx *fasthttp.RequestCtx, s alertChannelStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "alert channel id required")
		return
	}
	channelID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid alert channel id")
		return
	}

	ch, err := s.GetAlertChannel(channelID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(ctx, fasthttp.StatusNotFound, "alert channel not found")
			return
		}
		log.Printf("get alert channel for test: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get alert channel")
		return
	}

	var configMap map[string]any
	if ch.Config != "" {
		_ = json.Unmarshal([]byte(ch.Config), &configMap)
	}

	cfg := alerts.ChannelConfig{
		Type:     alerts.ChannelType(ch.ChannelType),
		IsActive: ch.IsActive,
	}
	if url, ok := configMap["url"].(string); ok {
		cfg.URL = url
	}
	if webhookURL, ok := configMap["webhook_url"].(string); ok {
		cfg.URL = webhookURL
	}
	if token, ok := configMap["token"].(string); ok {
		cfg.Token = token
	}
	if chatID, ok := configMap["chat_id"].(string); ok {
		cfg.ChatID = chatID
	}
	for _, e := range ch.Events {
		cfg.Events = append(cfg.Events, alerts.EventType(e))
	}

	res := alerts.Dispatch(cfg, alerts.EventQuotaDepleted, map[string]any{
		"message": "This is a test alert from g0router",
	})
	if !res.Success {
		writeError(ctx, fasthttp.StatusServiceUnavailable, res.Error.Error())
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"success": true})
}
