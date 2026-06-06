package handlers

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type promptTemplateStore interface {
	ListPromptTemplates() ([]store.PromptTemplate, error)
	GetPromptTemplate(id int64) (*store.PromptTemplate, error)
	CreatePromptTemplate(name, systemPrompt string, models []string, isActive bool) (*store.PromptTemplate, error)
	UpdatePromptTemplate(id int64, name, systemPrompt string, models []string, isActive bool) error
	DeletePromptTemplate(id int64) error
}

type promptTemplateRequest struct {
	Name         string   `json:"name"`
	SystemPrompt string   `json:"system_prompt"`
	Models       []string `json:"models"`
	IsActive     bool     `json:"is_active"`
}

type promptTemplateResponse struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	SystemPrompt string   `json:"system_prompt"`
	Models       []string `json:"models"`
	IsActive     bool     `json:"is_active"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

func toPromptTemplateResponse(t store.PromptTemplate) promptTemplateResponse {
	return promptTemplateResponse{
		ID:           t.ID,
		Name:         t.Name,
		SystemPrompt: t.SystemPrompt,
		Models:       t.Models,
		IsActive:     t.IsActive,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

// PromptTemplates handles CRUD for prompt templates.
func PromptTemplates(ctx *fasthttp.RequestCtx, s promptTemplateStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			templates, err := s.ListPromptTemplates()
			if err != nil {
				log.Printf("list prompt templates: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list prompt templates")
				return
			}
			resp := make([]promptTemplateResponse, 0, len(templates))
			for _, t := range templates {
				resp = append(resp, toPromptTemplateResponse(t))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[promptTemplateResponse]{Data: resp})
			return
		}

		tmplID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
			return
		}
		tmpl, err := s.GetPromptTemplate(tmplID)
		if err != nil {
			writeStoreError(ctx, "get prompt template", err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, toPromptTemplateResponse(*tmpl))

	case fasthttp.MethodPost:
		var req promptTemplateRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" || req.SystemPrompt == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name and system_prompt are required")
			return
		}
		tmpl, err := s.CreatePromptTemplate(req.Name, req.SystemPrompt, req.Models, req.IsActive)
		if err != nil {
			log.Printf("create prompt template: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create prompt template")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, toPromptTemplateResponse(*tmpl))

	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "id required")
			return
		}
		tmplID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
			return
		}
		var req promptTemplateRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" || req.SystemPrompt == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name and system_prompt are required")
			return
		}
		if err := s.UpdatePromptTemplate(tmplID, req.Name, req.SystemPrompt, req.Models, req.IsActive); err != nil {
			writeStoreError(ctx, "update prompt template", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)

	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "id required")
			return
		}
		tmplID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
			return
		}
		if err := s.DeletePromptTemplate(tmplID); err != nil {
			writeStoreError(ctx, "delete prompt template", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)

	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

// PromptTemplatesTest tests a prompt template against a model name.
func PromptTemplatesTest(ctx *fasthttp.RequestCtx, s promptTemplateStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "model is required")
		return
	}

	templates, err := s.ListPromptTemplates()
	if err != nil {
		log.Printf("list prompt templates for test: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list prompt templates")
		return
	}

	var matched []promptTemplateResponse
	for _, t := range templates {
		if !t.IsActive {
			continue
		}
		for _, m := range t.Models {
			if m == req.Model {
				matched = append(matched, toPromptTemplateResponse(t))
				break
			}
		}
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"model":     req.Model,
		"matched":   len(matched) > 0,
		"templates": matched,
	})
}
