package handlers

import (
	"encoding/json"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type disabledModelStore interface {
	ListDisabledModels() ([]store.DisabledModel, error)
	CreateDisabledModel(provider, model string) (*store.DisabledModel, error)
	DeleteDisabledModel(provider, model string) error
	IsModelDisabled(provider, model string) (bool, error)
	ListCustomModels() ([]store.CustomModel, error)
	CreateCustomModel(provider, model, displayName string) (*store.CustomModel, error)
	DeleteCustomModel(id string) error
}

type disabledModelResponse struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
}

type customModelResponse struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	DisplayName string `json:"display_name,omitempty"`
	CreatedAt   string `json:"created_at"`
	IsCustom    bool   `json:"is_custom"`
}

type disabledModelRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type customModelRequest struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	DisplayName string `json:"display_name,omitempty"`
}

func DisabledModelsList(ctx *fasthttp.RequestCtx, s disabledModelStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	models, err := s.ListDisabledModels()
	if err != nil {
		log.Printf("list disabled models: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list disabled models")
		return
	}
	resp := make([]disabledModelResponse, 0, len(models))
	for _, m := range models {
		resp = append(resp, disabledModelResponse{
			ID:        m.ID,
			Provider:  m.Provider,
			Model:     m.Model,
			CreatedAt: m.CreatedAt,
		})
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[disabledModelResponse]{Data: resp})
}

func DisabledModelsCreate(ctx *fasthttp.RequestCtx, s disabledModelStore, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	var req disabledModelRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Provider == "" || req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider and model are required")
		return
	}
	m, err := s.CreateDisabledModel(req.Provider, req.Model)
	if err != nil {
		log.Printf("create disabled model: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create disabled model")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "disabled_model.create",
		Target: req.Provider + "/" + req.Model,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusCreated, disabledModelResponse{
		ID:        m.ID,
		Provider:  m.Provider,
		Model:     m.Model,
		CreatedAt: m.CreatedAt,
	})
}

func DisabledModelsDelete(ctx *fasthttp.RequestCtx, s disabledModelStore, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodDelete {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	var req disabledModelRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Provider == "" || req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider and model are required")
		return
	}
	if err := s.DeleteDisabledModel(req.Provider, req.Model); err != nil {
		writeStoreError(ctx, "delete disabled model", err)
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "disabled_model.delete",
		Target: req.Provider + "/" + req.Model,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

func CustomModelsList(ctx *fasthttp.RequestCtx, s disabledModelStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	models, err := s.ListCustomModels()
	if err != nil {
		log.Printf("list custom models: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list custom models")
		return
	}
	resp := make([]customModelResponse, 0, len(models))
	for _, m := range models {
		resp = append(resp, customModelResponse{
			ID:          m.ID,
			Provider:    m.Provider,
			Model:       m.Model,
			DisplayName: m.DisplayName,
			CreatedAt:   m.CreatedAt,
			IsCustom:    true,
		})
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[customModelResponse]{Data: resp})
}

func CustomModelsCreate(ctx *fasthttp.RequestCtx, s disabledModelStore, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	var req customModelRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Provider == "" || req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider and model are required")
		return
	}
	m, err := s.CreateCustomModel(req.Provider, req.Model, req.DisplayName)
	if err != nil {
		log.Printf("create custom model: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create custom model")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "custom_model.create",
		Target: req.Provider + "/" + req.Model,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusCreated, customModelResponse{
		ID:          m.ID,
		Provider:    m.Provider,
		Model:       m.Model,
		DisplayName: m.DisplayName,
		CreatedAt:   m.CreatedAt,
		IsCustom:    true,
	})
}

func CustomModelsDelete(ctx *fasthttp.RequestCtx, s disabledModelStore, audit auditWriter, id string) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodDelete {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "custom model id required")
		return
	}
	if err := s.DeleteCustomModel(id); err != nil {
		writeStoreError(ctx, "delete custom model", err)
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "custom_model.delete",
		Target: id,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}
